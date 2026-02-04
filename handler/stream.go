


// package handler

// import (
// 	"NexusGateway/config"
// 	"NexusGateway/handler/gemini"
// 	"bufio"
	
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"strings"
// 	"sync"
// 	"time"
// )

// // Global Discovery Cache
// var discoveredModelMap = sync.Map{}

// // --- MAIN HANDLER ---

// func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
// 	startTime := time.Now()
// 	cfg := config.LoadConfig()
// 	ctx := context.Background()
// 	userKey := getStreamAPIKey(r)
// 	redisClient := GetClient()

// 	// 1. CAPTURE BYOK & AUTH
// 	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
// 	userGroqKey := r.Header.Get("x-nexus-groq-key")
// 	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
// 	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

// 	// Set SSE Headers
// 	w.Header().Set("Content-Type", "text/event-stream")
// 	w.Header().Set("X-Accel-Buffering", "no") 
// 	w.Header().Set("Cache-Control", "no-cache")
// 	w.Header().Set("Connection", "keep-alive")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	var userReq ChatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
// 		LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0, "", "", "NONE", "FAILED")
// 		return
// 	}
// 	if userReq.Model == "" { userReq.Model = "llama-3.3-70b-versatile" }

// 	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
// 	gov := EvaluateConstitution(userReq.Message)
// 	if !gov.Allowed {
// 		log.Printf("🚫 [REFUSAL] Constitution Blocked: %s", gov.RuleID)
// 		streamSimulatedResponse(w, gov.RefusalMsg)
// 		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")
// 		return
// 	}

// 	// Governance Result mapping for logs
// 	userReq.Message = gov.ModifiedText
// 	govAction := "PERMITTED"
// 	triggeredRule := "NONE"
// 	if gov.RuleID != "NONE" {
// 		govAction = "REDACTED"
// 		triggeredRule = gov.RuleID
// 	}

// 	log.Printf("📥 [REQUEST] User: %s... | Model: %s | Gov: %s", userKey[:6], userReq.Model, govAction)

// 	// --- 2. HYBRID CACHE LAYER 0 (REDIS) ---
// 	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
// 	msgHash := GenerateHash(cleanMsg)
// 	if redisClient != nil {
// 		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
// 			log.Printf("🚀 [HIT] Redis serving instantly.")
// 			streamSimulatedResponse(w, cached)
// 			if gov.Disclaimer != "" { streamSimulatedResponse(w, gov.Disclaimer) }

// 			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cached)
// 			sav := CalculateSavings(userReq.Model, pT, rT)
// 			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()), userReq.Message, cached, triggeredRule, govAction)
// 			return
// 		}
// 	}

// 	log.Printf("🐢 [MISS] %s. Checking Semantic Cache...", msgHash[:8])

// 	// --- 3. VECTOR DB SEARCH (LAYER 1) ---
// 	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)
// 	if vector != nil && cfg.PineconeKey != "" {
// 		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
// 		if err == nil && score > 0.75 {
// 			log.Printf("⚡ [PINECONE HIT] Score: %.2f", score)
// 			streamSimulatedResponse(w, cachedAnswer)
// 			if gov.Disclaimer != "" { streamSimulatedResponse(w, gov.Disclaimer) }

// 			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cachedAnswer)
// 			sav := CalculateSavings(userReq.Model, pT, rT)
// 			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()), userReq.Message, cachedAnswer, triggeredRule, govAction)
// 			return
// 		}
// 	}

// 	// --- 4. UNIVERSAL ROUTER (PRECISION KEY RESOLUTION) ---
// 	var providerReq *http.Request
// 	var err error
// 	modelLower := strings.ToLower(userReq.Model)
// 	isGemini := strings.Contains(modelLower, "gemini")
// 	isGroq := strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral")

// 	targetKey := strings.TrimSpace(cfg.OpenAIKey)
	
// 	if isGemini {
// 		targetKey = strings.TrimSpace(cfg.GeminiKey)
// 		if userGeminiKey != "" { targetKey = strings.TrimSpace(userGeminiKey) }
// 		modelID := "gemini-1.5-flash"
// 		if cached, ok := discoveredModelMap.Load(targetKey); ok { modelID = cached.(string) }
// 		modelID = strings.TrimPrefix(modelID, "models/")
// 		providerReq, err = gemini.PrepareNativeRequest(userReq.Message, modelID, targetKey, "v1")
// 	} else if isGroq {
// 		// 🚀 THE GROQ FIX: Ensure stable model ID and correct key
// 		targetKey = strings.TrimSpace(cfg.GroqKey)
// 		if userGroqKey != "" { targetKey = strings.TrimSpace(userGroqKey) }
		
// 		// Force modern model to avoid 404
// 		stableGroqModel := "llama-3.3-70b-versatile"
// 		log.Printf("🛰️ [GROQ ROUTE] Using: %s", stableGroqModel)
		
// 		p := GetProvider(userReq.Model)
// 		providerReq, err = p.PrepareRequest(userReq.Message, stableGroqModel, targetKey, "")
// 		if providerReq != nil {
// 			providerReq.Header.Set("Authorization", "Bearer "+targetKey)
// 		}
// 	} else {
// 		// OpenAI Standard
// 		log.Printf("🔀 [STANDARD ROUTE] Routing to %s", userReq.Model)
// 		if userOpenAIKey != "" { targetKey = strings.TrimSpace(userOpenAIKey) }
// 		p := GetProvider(userReq.Model)
// 		providerReq, err = p.PrepareRequest(userReq.Message, userReq.Model, targetKey, "")
// 		if providerReq != nil {
// 			providerReq.Header.Set("Authorization", "Bearer "+targetKey)
// 		}
// 	}

// 	if err != nil {
// 		log.Printf("❌ [ROUTING ERROR] %v", err)
// 		return
// 	}

// 	// --- 5. EXECUTE ---
// 	client := &http.Client{Timeout: 90 * time.Second}
// 	resp, err := client.Do(providerReq)

// 	// AUTO-RECOVERY: If V1 fails, use Discovery from gemini_discovery.go
// 	if isGemini && (err != nil || (resp != nil && resp.StatusCode == 404)) {
// 		log.Printf("🔄 [HEALING] Version fail. Attempting Adaptive Discovery...")
// 		workingModel := DiscoverWorkingModel(targetKey) // 🚀 Calls gemini_discovery.go
// 		discoveredModelMap.Store(targetKey, workingModel)
		
// 		adapter := &GeminiAdapter{}
// 		providerReq, _ = adapter.PrepareRequest(userReq.Message, workingModel, targetKey, "v1beta")
// 		resp, err = client.Do(providerReq)
// 	}

// 	if err != nil || (resp != nil && resp.StatusCode != 200) {
// 		body, _ := io.ReadAll(resp.Body)
// 		log.Printf("❌ [PROVIDER ERROR] Status: %d | Body: %s", 0, string(body))
// 		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(body), triggeredRule, "FAILED")
// 		fmt.Fprintf(w, "data: {\"error\": \"Inference failed\", \"details\": %s}\n\n", string(body))
// 		return
// 	}
// 	defer resp.Body.Close()
// 	log.Printf("📡 [STREAM] Provider Connected. Capturing Chunks...")

// 	// --- 6. UNIFIED PARSER ---
// 	reader := bufio.NewReader(resp.Body)
// 	var fullResponseBuilder strings.Builder
// 	for {
// 		line, err := reader.ReadBytes('\n')
// 		if err != nil { break }
// 		lineStr := string(line)
// 		if strings.HasPrefix(lineStr, "data: ") {
// 			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
// 			if cleanLine == "" || cleanLine == "[DONE]" { 
// 				fmt.Fprintf(w, "data: [DONE]\n\n")
// 				if f, ok := w.(http.Flusher); ok { f.Flush() }
// 				break 
// 			}

// 			var extracted string
// 			if isGemini {
// 				var gRes gemini.GeminiResponse
// 				if err := json.Unmarshal([]byte(cleanLine), &gRes); err == nil && len(gRes.Candidates) > 0 {
// 					if len(gRes.Candidates[0].Content.Parts) > 0 {
// 						extracted = gRes.Candidates[0].Content.Parts[0].Text
// 					}
// 				}
// 			} else {
// 				p := GetProvider(userReq.Model)
// 				extracted, _ = p.ParseStreamChunk(lineStr)
// 			}

// 			if extracted != "" {
// 				fullResponseBuilder.WriteString(extracted)
// 				chunk := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": extracted}}}}
// 				jsonChunk, _ := json.Marshal(chunk)
// 				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
// 				if f, ok := w.(http.Flusher); ok { f.Flush() }
// 				if isGemini { time.Sleep(8 * time.Millisecond) } 
// 			}
// 		}
// 	}

// 	// 7. FINAL TELEMETRY LEDGER
// 	finalText := fullResponseBuilder.String()
// 	if finalText != "" {
// 		go func() {
// 			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(finalText)
// 			latency := int(time.Since(startTime).Milliseconds())

// 			if !usingOwnKey {
// 			IncrementUsage(userKey) // Database mein credits -1
// 			if redisClient != nil {
// 				redisClient.Incr(ctx, "stats:total_requests") // Global Stats update
// 			}
// 		}

// 			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, finalText, 24*time.Hour) }
// 			if vector != nil && cfg.PineconeKey != "" { SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, finalText) }
			
// 			// Log telemetry with 12 parameters
// 			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, latency, userReq.Message, finalText, "NONE", "PERMITTED")
// 		}()
// 	}
// }

// // --- HELPERS (Essential) ---

// func getStreamAPIKey(r *http.Request) string {
// 	auth := r.Header.Get("Authorization")
// 	parts := strings.Split(auth, " ")
// 	if len(parts) == 2 { return parts[1] }
// 	return ""
// }

// func streamSimulatedResponse(w http.ResponseWriter, text string) {
// 	words := strings.Split(text, " ")
// 	for _, word := range words {
// 		formatted := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": word + " "}}}}
// 		jsonChunk, _ := json.Marshal(formatted)
// 		fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
// 		if f, ok := w.(http.Flusher); ok { f.Flush() }
// 		time.Sleep(15 * time.Millisecond)
// 	}
// 	fmt.Fprintf(w, "data: [DONE]\n\n")
// }





package handler

import (
	"NexusGateway/config"
	"NexusGateway/handler/gemini"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Global Discovery Cache
var discoveredModelMap = sync.Map{}

// --- MAIN HANDLER ---

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	cfg := config.LoadConfig()
	ctx := context.Background()
	userKey := getStreamAPIKey(r)
	redisClient := GetClient()

	// 1. CAPTURE BYOK & AUTH HEADERS
	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	userGroqKey := r.Header.Get("x-nexus-groq-key")
	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
	userAnthropicKey := r.Header.Get("x-nexus-anthropic-key")
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "" || userAnthropicKey != "")

	// Set Professional SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no") 
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0, "", "", "NONE", "FAILED")
		return
	}
	if userReq.Model == "" { userReq.Model = "llama-3.3-70b-versatile" } // High-speed default

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
	gov := EvaluateConstitution(userReq.Message)
	if !gov.Allowed {
		log.Printf("🚫 [REFUSAL] Constitution Blocked: %s", gov.RuleID)
		streamSimulatedResponse(w, gov.RefusalMsg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")
		return
	}

	// Apply Governance
	userReq.Message = gov.ModifiedText
	govAction := "PERMITTED"
	triggeredRule := "NONE"
	if gov.RuleID != "NONE" {
		govAction = "REDACTED"
		triggeredRule = gov.RuleID
	}

	// 🛡️ IRON DOME: PREMIUM GATE
	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
		msg := "Premium Model Locked. Use BYOK or Upgrade."
		streamSimulatedResponse(w, msg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, msg, "IRON_DOME", "BLOCKED")
		return
	}

	log.Printf("📥 [REQUEST] User: %s... | Model: %s | Gov: %s", userKey[:6], userReq.Model, govAction)

	// --- 2. HYBRID CACHE LAYER 0 (REDIS) ---
	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
	msgHash := GenerateHash(cleanMsg)
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 [HIT] Redis serving instantly.")
			streamSimulatedResponse(w, cached)
			if gov.Disclaimer != "" { streamSimulatedResponse(w, gov.Disclaimer) }

			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cached)
			sav := CalculateSavings(userReq.Model, pT, rT)
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()), userReq.Message, cached, triggeredRule, govAction)
			return
		}
	}

	log.Printf("🐢 [MISS] %s. Routing to provider...", userReq.Model)

	// 🚀 PRE-GENERATE VECTOR (Used for Layer 1 and Background Save)
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)

	// --- 3. VECTOR DB SEARCH (LAYER 1) ---
	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		if err == nil && score > 0.75 {
			log.Printf("⚡ [PINECONE HIT] Score: %.2f", score)
			streamSimulatedResponse(w, cachedAnswer)
			if gov.Disclaimer != "" { streamSimulatedResponse(w, gov.Disclaimer) }
			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cachedAnswer)
			sav := CalculateSavings(userReq.Model, pT, rT)
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()), userReq.Message, cachedAnswer, triggeredRule, govAction)
			return
		}
	}

	// --- 4. UNIVERSAL ROUTER (PRECISION MAPPING) ---
	var providerReq *http.Request
	var err error
	modelLower := strings.ToLower(userReq.Model)
	isGemini := strings.Contains(modelLower, "gemini")
	isGroq := strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral")

	targetKey := strings.TrimSpace(cfg.OpenAIKey)
	
	if isGemini {
		targetKey = strings.TrimSpace(cfg.GeminiKey)
		if userGeminiKey != "" { targetKey = strings.TrimSpace(userGeminiKey) }
		modelID := "gemini-1.5-flash"
		if cached, ok := discoveredModelMap.Load(targetKey); ok { modelID = cached.(string) }
		modelID = strings.TrimPrefix(modelID, "models/")

		log.Printf("🛰️ [NATIVE] Routing Gemini -> %s", modelID)
		// FIXED: Passing 4 arguments (v1 as default anchor)
		providerReq, err = gemini.PrepareNativeRequest(userReq.Message, modelID, targetKey, "v1")
	} else if isGroq {
		targetKey = strings.TrimSpace(cfg.GroqKey)
		if userGroqKey != "" { targetKey = strings.TrimSpace(userGroqKey) }
		
		stableModel := "llama-3.3-70b-versatile"
		p := GetProvider(userReq.Model)
		// FIXED: Passing 4 arguments
		providerReq, err = p.PrepareRequest(userReq.Message, stableModel, targetKey, "")
		if providerReq != nil {
			providerReq.Header.Set("Authorization", "Bearer "+targetKey)
		}
	} else {
		if userOpenAIKey != "" { targetKey = strings.TrimSpace(userOpenAIKey) }
		p := GetProvider(userReq.Model)
		// FIXED: Passing 4 arguments
		providerReq, err = p.PrepareRequest(userReq.Message, userReq.Model, targetKey, "")
		if providerReq != nil {
			providerReq.Header.Set("Authorization", "Bearer "+targetKey)
		}
	}

	if err != nil {
		log.Printf("❌ [ROUTING ERROR] %v", err)
		return 
	}

	// --- 5. EXECUTE ---
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(providerReq)

	// Self-Healing for Gemini 404
	if isGemini && (err != nil || (resp != nil && resp.StatusCode == 404)) {
		log.Printf("🔄 [HEALING] Version fail. Attempting Adaptive Discovery...")
		workingModel := DiscoverWorkingModel(targetKey)
		discoveredModelMap.Store(targetKey, workingModel)
		// FIXED: Passing 4 arguments to healing path
		providerReq, _ = gemini.PrepareNativeRequest(userReq.Message, workingModel, targetKey, "v1beta")
		resp, err = client.Do(providerReq)
	}

	if err != nil || (resp != nil && resp.StatusCode != 200) {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [PROVIDER ERROR] Status: %d | Body: %s", 0, string(body))
		// 🚀 THE ALERT HOOK
		Notify(fmt.Sprintf("❌ PROVIDER FAIL: %s\nStatus: %d\nDetails: %s", userReq.Model, resp.StatusCode, string(body)[:100]))

		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(body), "NONE", "FAILED")
		fmt.Fprintf(w, "data: {\"error\": \"Inference failed\", \"details\": %s}\n\n", string(body))
		return
	}
	defer resp.Body.Close()
	log.Printf("📡 [STREAM] Provider Connected. Capturing Chunks...")

	// --- 6. STREAM PARSER (UNIFIED) ---
	reader := bufio.NewReader(resp.Body)
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)
		if strings.HasPrefix(lineStr, "data: ") {
			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
			
			if cleanLine == "[DONE]" { 
				fmt.Fprintf(w, "data: [DONE]\n\n")
				break 
			}
			if cleanLine == "" { continue }

			var extracted string
			if isGemini {
				var gRes gemini.GeminiResponse
				if err := json.Unmarshal([]byte(cleanLine), &gRes); err == nil && len(gRes.Candidates) > 0 {
					if len(gRes.Candidates[0].Content.Parts) > 0 {
						extracted = gRes.Candidates[0].Content.Parts[0].Text
					}
				}
			} else {
				// Safe Dynamic Parsing for OpenAI and Groq
				var raw map[string]interface{}
				if err := json.Unmarshal([]byte(cleanLine), &raw); err == nil {
					if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
						choice := choices[0].(map[string]interface{})
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							if content, ok := delta["content"].(string); ok {
								extracted = content
							}
						}
					}
				}
			}

			if extracted != "" {
				fullResponseBuilder.WriteString(extracted)
				chunk := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": extracted}}}}
				jsonChunk, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				if f, ok := w.(http.Flusher); ok { f.Flush() }
				if isGemini { time.Sleep(8 * time.Millisecond) }
			}
		}
	}

	// --- 7. LEDGER ---
	finalText := fullResponseBuilder.String()
	if finalText != "" {
		if gov.Disclaimer != "" { finalText += gov.Disclaimer }
		go func() {
			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(finalText)
			latency := int(time.Since(startTime).Milliseconds())
			
			// 🚀 THE FIX: Single Charge (+1) logic here
			if !usingOwnKey {
				IncrementUsage(userKey) 
				if redisClient != nil {
					redisClient.Incr(ctx, "stats:total_requests") 
				}
			}

			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, finalText, 24*time.Hour) }
			if vector != nil && cfg.PineconeKey != "" { SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, finalText) }
			
			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, latency, userReq.Message, finalText, triggeredRule, govAction)
		}()
	}
}

// --- HELPERS (Essential) ---

func getStreamAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	parts := strings.Split(auth, " ")
	if len(parts) == 2 { return parts[1] }
	return ""
}

func streamSimulatedResponse(w http.ResponseWriter, text string) {
	words := strings.Split(text, " ")
	for _, word := range words {
		formatted := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": word + " "}}}}
		jsonChunk, _ := json.Marshal(formatted)
		fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
		if f, ok := w.(http.Flusher); ok { f.Flush() }
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
}
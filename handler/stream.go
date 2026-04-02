
// package handler

// import (
// 	"NexusGateway/config"
// 	"bufio"
// 	"bytes"
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

// // Global Discovery Cache - Maps API Key to its working Model ID
// var discoveredModelMap = sync.Map{}

// // --- MAIN HANDLER ---

// func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
// 	startTime := time.Now()
// 	cfg := config.LoadConfig()
// 	ctx := context.Background()
// 	userKey := getStreamAPIKey(r)
// 	redisClient := GetClient()

// 	// 1. CAPTURE BYOK & AUTH HEADERS
// 	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
// 	userGroqKey := r.Header.Get("x-nexus-groq-key")
// 	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
// 	userAnthropicKey := r.Header.Get("x-nexus-anthropic-key")
// 	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "" || userAnthropicKey != "")

// 	// Set Professional SSE Headers
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

// 	// Governance Setup for Logs
// 	userReq.Message = gov.ModifiedText
// 	govAction := "PERMITTED"
// 	triggeredRule := "NONE"
// 	if gov.RuleID != "NONE" {
// 		govAction = "REDACTED"
// 		triggeredRule = gov.RuleID
// 	}

// 	// 🛡️ IRON DOME: PREMIUM MODEL GATE
// 	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
// 		msg := "Premium Model Locked. Use BYOK or Upgrade."
// 		streamSimulatedResponse(w, msg)
// 		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, msg, "IRON_DOME", "BLOCKED")
// 		return
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
// 			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, CalculateSavings(userReq.Model, pT, rT), int(time.Since(startTime).Milliseconds()), userReq.Message, cached, triggeredRule, govAction)
// 			return
// 		}
// 	}

// 	// --- 🚀 NEW: LAYER 1 (INTENT CACHE) ---
// 	//intentKey := GenerateIntentSignature(userReq.Message)
// 	intentKey := GenerateIntentSignature(userReq.Message, cfg.GroqKey)
// 	if redisClient!= nil {
// 		if intentCached, _ := redisClient.Get(ctx, "intent:"+intentKey).Result(); intentCached!= "" {
// 			log.Printf("🧠 Intent Cache serving instantly.")
// 			streamSimulatedResponse(w, intentCached)
// 			if gov.Disclaimer!= "" {
// 				streamSimulatedResponse(w, gov.Disclaimer)
// 			}
// 			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(intentCached)
// 			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, CalculateSavings(userReq.Model, pT, rT), int(time.Since(startTime).Milliseconds()), userReq.Message, intentCached, triggeredRule, govAction)
// 			return
// 		}
// 	}

	

// 	// --- 🚀 NEW: LAYER 2 (SEMANTIC PINECONE CACHE) ---
// 	vector, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
// 	if err == nil {
// 		avgSim := 0.65
// 		dynamicThresh := CalculateDynamicThreshold(userReq.Message, 0.70, avgSim)

// 		answer, score, searchErr := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
// 		// 🚩 MOVE THIS LOG ABOVE THE IF STATEMENT
// 		log.Printf("🔍 [DEBUG] Score: %.4f | Thresh: %.2f | Err: %v", score, dynamicThresh, searchErr)

// 		if searchErr == nil && score >= dynamicThresh && answer != "" {
// 			log.Printf("🌌 Pinecone Semantic Match (Score: %.2f)", score)
// 			// ... rest of your hit logic ...
// 			return 
// 		}
// 	}



// 	// --- 🚀 3. SOVEREIGN ROUTER (INTELLIGENT ORCHESTRATION) ---

// 	// 1. Get the intelligent adapter for this specific model
// 	provider := GetProvider(userReq.Model)

// 	// 2. Map the correct API Key from your Config
// 	var activeKey string
// 	modelLower := strings.ToLower(userReq.Model)

// 	switch {
// 	case strings.Contains(modelLower, "deepseek"):
// 		activeKey = cfg.DeepSeekKey
// 	case strings.Contains(modelLower, "gemini"):
// 		activeKey = cfg.GeminiKey
// 	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral"):
// 		activeKey = cfg.GroqKey
// 	default:
// 		activeKey = cfg.OpenAIKey
// 	}

// 	// 3. Let the Provider build the request for its own API (URL, Headers, Payload)
// 	req, err := provider.PrepareRequest(userReq.Message, userReq.Model, activeKey, "v1")
// 	if err != nil {
// 		log.Printf("❌ [ROUTER ERROR] %v", err)
// 		return
// 	}

// 	log.Printf("📡 [SOVEREIGN] Routing %s via %T", userReq.Model, provider)



// 	// --- 3. UNIVERSAL ROUTER (NATIVE BRIDGING) ---
// 	// var targetURL string
// 	// var payloadBytes []byte
// 	// var targetKey string
// 	// modelLower := strings.ToLower(userReq.Model)
// 	// isGemini := strings.Contains(modelLower, "gemini")

// 	// if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral") {
// 	// 	// GROQ BRIDGE
// 	// 	targetURL = "https://api.groq.com/openai/v1/chat/completions"
// 	// 	targetKey = cfg.GroqKey
// 	// 	if userGroqKey != "" { targetKey = userGroqKey }
// 	// 	payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: "llama-3.3-70b-versatile", Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
// 	// 	log.Printf("🔀 [ROUTING] Groq Bridge Active")

// 	// } else if isGemini {
// 	// 	// 🚀 THE NATIVE GOOGLE BRIDGE (Solves 404 Errors)
// 	// 	targetKey = cfg.GeminiKey
// 	// 	if userGeminiKey != "" { targetKey = userGeminiKey }
		
// 	// 	modelID := "gemini-2.0-flash-exp" // Default to Next-Gen
// 	// 	if cached, ok := discoveredModelMap.Load(targetKey); ok { modelID = cached.(string) }
// 	// 	modelID = strings.TrimPrefix(modelID, "models/")

// 	// 	targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, targetKey)
		
// 	// 	// Map OpenAI Payload to Google Native Protocol
// 	// 	geminiPayload := map[string]any{
// 	// 		"contents": []map[string]any{{"parts": []map[string]string{{"text": userReq.Message}}}},
// 	// 	}
// 	// 	payloadBytes, _ = json.Marshal(geminiPayload)
// 	// 	log.Printf("🛰️ [NATIVE] Routing Gemini -> %s", modelID)

// 	// } else {
// 	// 	// OPENAI BRIDGE
// 	// 	targetURL = "https://api.openai.com/v1/chat/completions"
// 	// 	targetKey = cfg.OpenAIKey
// 	// 	if userOpenAIKey != "" { targetKey = userOpenAIKey }
// 	// 	payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
// 	// 	log.Printf("🔀 [ROUTING] OpenAI Bridge Active")
// 	// }

// 	// --- 4. EXECUTE ---
// 	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
// 	req.Header.Set("Content-Type", "application/json")
// 	if !isGemini { req.Header.Set("Authorization", "Bearer "+targetKey) }

// 	client := &http.Client{Timeout: 90 * time.Second}
// 	resp, err := client.Do(req)

// 	// SELF-HEALING: If V1 fails, trigger discovery (Gemini only)
// 	if isGemini && (err != nil || (resp != nil && resp.StatusCode == 404)) {
// 		log.Printf("🔄 [HEALING] Path failed. Attempting Adaptive Discovery...")
// 		workingModel := DiscoverWorkingModel(targetKey)
// 		discoveredModelMap.Store(targetKey, workingModel)
		
// 		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", workingModel, targetKey)
// 		req, _ = http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
// 		resp, err = client.Do(req)
// 	}

// 	if err != nil || (resp != nil && resp.StatusCode != 200) {
// 		body, _ := io.ReadAll(resp.Body)
// 		log.Printf("❌ [PROVIDER ERROR] Status: %d | Body: %s", resp.StatusCode, string(body))
// 		Notify(fmt.Sprintf("❌ PROVIDER FAIL: %s\nStatus: %d\nDetails: %s", userReq.Model, resp.StatusCode, string(body)[:100]))
// 		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(body), "NONE", "FAILED")
// 		fmt.Fprintf(w, "data: {\"error\": \"Inference failed\"}\n\n")
// 		return
// 	}
// 	defer resp.Body.Close()
// 	log.Printf("📡 [STREAM] Provider Connected. Piping data...")

// 	// --- 5. UNIFIED STREAM PARSER ---
// 	reader := bufio.NewReader(resp.Body)
// 	var fullResponseBuilder strings.Builder
// 	for {
// 		line, err := reader.ReadBytes('\n')
// 		if err != nil { break }
// 		lineStr := string(line)
// 		if strings.HasPrefix(lineStr, "data: ") {
// 			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
			
// 			if cleanLine == "[DONE]" { 
// 				fmt.Fprintf(w, "data: [DONE]\n\n")
// 				break 
// 			}
// 			if cleanLine == "" { continue }

// 			var extracted string
// 			if isGemini {
// 				// Parse Google Native Format
// 				var gRes struct {
// 					Candidates []struct { Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"` } `json:"candidates"`
// 				}
// 				if err := json.Unmarshal([]byte(cleanLine), &gRes); err == nil && len(gRes.Candidates) > 0 {
// 					extracted = gRes.Candidates[0].Content.Parts[0].Text
// 				}
// 			} else {
// 				// Parse OpenAI format (GPT/Groq)
// 				var raw map[string]interface{}
// 				if err := json.Unmarshal([]byte(cleanLine), &raw); err == nil {
// 					if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
// 						choice := choices[0].(map[string]interface{})
// 						if delta, ok := choice["delta"].(map[string]interface{}); ok {
// 							if content, ok := delta["content"].(string); ok {
// 								extracted = content
// 							}
// 						}
// 					}
// 				}
// 			}

// 			// --- 🚀 THE SMOOTH-STREAM PACING ---
// 			if extracted != "" {
// 				fullResponseBuilder.WriteString(extracted)
// 				words := strings.Split(extracted, " ")
// 				for i, word := range words {
// 					content := word
// 					if i < len(words)-1 { content += " " }
// 					formatted := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": content}}}}
// 					jsonChunk, _ := json.Marshal(formatted)
// 					fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
// 					if f, ok := w.(http.Flusher); ok { f.Flush() }
// 					// Delay only for paragraph dumps (Gemini)
// 					if len(words) > 1 { time.Sleep(10 * time.Millisecond) }
// 				}
// 			}
// 		}
// 	}



// 	// --- 6. LEDGER & TELEMETRY ---
// 	finalText := fullResponseBuilder.String()
// 	if finalText != "" {
// 		if gov.Disclaimer != "" { finalText += gov.Disclaimer }
// 		go func() {
// 			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(finalText)
// 			latency := int(time.Since(startTime).Milliseconds())
			
// 			// 1. Log to Supabase
// 			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, latency, userReq.Message, finalText, triggeredRule, govAction)
			
// 			// 2. Cache logic
// 			if redisClient != nil { 
// 				redisClient.Set(ctx, "intent:"+intentKey, finalText, 24*time.Hour) 
// 			}
			
// 			// 3. Save to Pinecone Hybrid
// 			denseVec, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
// 			// if err == nil {
// 			// 	sparseIndices, sparseValues := GenerateSparseVector(userReq.Message)
// 			// 	SaveToPineconeHybrid(cfg.PineconeHost, cfg.PineconeKey, intentKey, denseVec, sparseIndices, sparseValues, finalText)
// 			// 	log.Println("💾 [DEBUG] Data saved to Pinecone Hybrid Cache")
// 			// }
// 			// stream.go logic
// 			if err == nil && denseVec != nil {
// 				sparseIndices, sparseValues := GenerateSparseVector(userReq.Message)
// 				// This matches the 7-argument master function in vector.go
// 				SaveToPineconeHybrid(cfg.PineconeHost, cfg.PineconeKey, intentKey, denseVec, sparseIndices, sparseValues, finalText)
// 				log.Println("💾 [DEBUG] Data saved to Pinecone Hybrid Cache")
// 			}
// 		}()
// 	}
// }

// // --- ALL HELPERS SYNCED ---

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

// Global Discovery Cache - Maps API Key to its working Model ID
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
	userDeepSeekKey := r.Header.Get("x-nexus-deepseek-key")
    usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "" || userAnthropicKey != "" || userDeepSeekKey != "")
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
	if userReq.Model == "" { userReq.Model = "llama-3.3-70b-versatile" }

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
	gov := EvaluateConstitution(userReq.Message)
	if !gov.Allowed {
		log.Printf("🚫 [REFUSAL] Constitution Blocked: %s", gov.RuleID)
		streamSimulatedResponse(w, gov.RefusalMsg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")
		return
	}

	userReq.Message = gov.ModifiedText
	govAction := "PERMITTED"
	triggeredRule := "NONE"
	if gov.RuleID != "NONE" {
		govAction = "REDACTED"
		triggeredRule = gov.RuleID
	}

	// 🛡️ IRON DOME: PREMIUM MODEL GATE
	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
		msg := "Premium Model Locked. Use BYOK or Upgrade."
		streamSimulatedResponse(w, msg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, msg, "IRON_DOME", "BLOCKED")
		return
	}

	log.Printf("📥 [REQUEST] User: %s... | Model: %s | Gov: %s", userKey[:6], userReq.Model, govAction)

	// --- 2. HYBRID CACHE LAYER (REDIS & SEMANTIC) ---
	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
	msgHash := GenerateHash(cleanMsg)
	intentKey := GenerateIntentSignature(userReq.Message, cfg.GroqKey)

	// Redis Exact/Intent Check
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 [HIT] Redis serving instantly.")
			streamSimulatedResponse(w, cached)
			return
		}
	}

	// Semantic Pinecone Check
	vector, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
	if err == nil {
		dynamicThresh := CalculateDynamicThreshold(userReq.Message, 0.70, 0.65)
		answer, score, searchErr := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
		log.Printf("🔍 [DEBUG] Score: %.4f | Thresh: %.2f", score, dynamicThresh)

		if searchErr == nil && score >= dynamicThresh && answer != "" {
			log.Printf("🌌 Pinecone Semantic Match (Score: %.2f)", score)
			streamSimulatedResponse(w, answer)
			return 
		}
	}

	// --- 🚀 3. SOVEREIGN ROUTER (INTELLIGENT ORCHESTRATION) ---
	provider := GetProvider(userReq.Model)
	var activeKey string
	modelLower := strings.ToLower(userReq.Model)

	// Resolve API Key Priority
	switch {
	case strings.Contains(modelLower, "deepseek"):
		activeKey = cfg.DeepSeekKey
		if userDeepSeekKey != "" { activeKey = userDeepSeekKey }
	case strings.Contains(modelLower, "gemini"):
		activeKey = cfg.GeminiKey
		if userGeminiKey != "" { activeKey = userGeminiKey }
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral"):
		activeKey = cfg.GroqKey
		if userGroqKey != "" { activeKey = userGroqKey }
	default:
		activeKey = cfg.OpenAIKey
		if userOpenAIKey != "" { activeKey = userOpenAIKey }
	}

	// Build Native Request via Adapter
	req, err := provider.PrepareRequest(userReq.Message, userReq.Model, activeKey, "v1")
	if err != nil {
		log.Printf("❌ [ROUTER ERROR] %v", err)
		return
	}

	// --- 🚀 4. EXECUTION & SELF-HEALING ---
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)

	// Self-Healing for Gemini Models
	if strings.Contains(modelLower, "gemini") && (err != nil || (resp != nil && resp.StatusCode == 404)) {
		log.Printf("🔄 [HEALING] Path failed. Attempting Adaptive Discovery...")
		workingModel := DiscoverWorkingModel(activeKey)
		discoveredModelMap.Store(activeKey, workingModel)
		req, _ = provider.PrepareRequest(userReq.Message, workingModel, activeKey, "v1beta")
		resp, err = client.Do(req)
	}

	if err != nil || (resp != nil && resp.StatusCode != 200) {
		errorBody := "Inference Failed"
		if resp != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errorBody = string(bodyBytes)
		}
		log.Printf("❌ [PROVIDER ERROR] Status: %d | Body: %s", resp.StatusCode, errorBody)
		fmt.Fprintf(w, "data: {\"error\": \"Provider unavailable\"}\n\n")
		return
	}
	defer resp.Body.Close()
	log.Printf("📡 [STREAM] %T Connected. Piping data...", provider)

	// --- 🚀 5. UNIFIED STREAM PARSER & OUTPUT ---
	scanner := bufio.NewScanner(resp.Body)
	var fullResponseBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" { continue }

		// Use the Adapter's parser to handle model-specific JSON
		extracted, isDone := provider.ParseStreamChunk(line)
		
		if isDone {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			break
		}

		if extracted != "" {
			fullResponseBuilder.WriteString(extracted)
			
			// Format to OpenAI-standard JSON for frontend compatibility
			formatted := map[string]any{
				"choices": []map[string]any{
					{"delta": map[string]string{"content": extracted}},
				},
			}
			jsonChunk, _ := json.Marshal(formatted)
			fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
			
			if f, ok := w.(http.Flusher); ok { f.Flush() }
		}
	}

	// --- 💾 6. LEDGER & TELEMETRY (ASYNC) ---
	finalText := fullResponseBuilder.String()
	if finalText != "" {
		go func(reply string, v []float32) {
			pT, rT, cost := provider.GetPricing(userReq.Message, reply, userReq.Model)
			latency := int(time.Since(startTime).Milliseconds())

			// 1. Log to DB
			LogRequest(userKey, userReq.Model, 200, false, pT, rT, cost, latency, userReq.Message, reply, triggeredRule, govAction)
			
			// 2. Exact Cache
			if redisClient != nil { 
				redisClient.Set(ctx, "exact:"+msgHash, reply, 24*time.Hour) 
			}
			
			// 3. Semantic Memory (Pinecone Hybrid)
			if v != nil {
				sparseIndices, sparseValues := GenerateSparseVector(userReq.Message)
				SaveToPineconeHybrid(cfg.PineconeHost, cfg.PineconeKey, intentKey, v, sparseIndices, sparseValues, reply)
				log.Println("💾 [DEBUG] Data saved to Pinecone Hybrid Cache")
			}
		}(finalText, vector)
	}
}

// --- HELPERS ---

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
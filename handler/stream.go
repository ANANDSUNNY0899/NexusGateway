// package handler

// import (
// 	"NexusGateway/config"
// 	"NexusGateway/handler/gemini"
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

// var discoveredModelMap = sync.Map{}

// func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
// 	startTime := time.Now()
// 	cfg := config.LoadConfig()
// 	ctx := context.Background()
// 	userKey := getStreamAPIKey(r)
// 	redisClient := GetClient()

// 	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
// 	userGroqKey := r.Header.Get("x-nexus-groq-key")
// 	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
// 	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

// 	w.Header().Set("Content-Type", "text/event-stream")
// 	w.Header().Set("X-Accel-Buffering", "no")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	var userReq ChatRequest
// 	json.NewDecoder(r.Body).Decode(&userReq)
// 	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

// 	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
// 	gov := EvaluateConstitution(userReq.Message)
// 	if !gov.Allowed {
// 		log.Printf("🚫 [REFUSAL] Blocked by Governance")
// 		streamSimulatedResponse(w, gov.RefusalMsg)
// 		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")
// 		return
// 	}

// 	// 🛡️ IRON DOME: PREMIUM GATE
// 	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
// 		log.Printf("🛡️ [IRON DOME] Blocking premium model")
// 		fmt.Fprintf(w, "data: {\"error\": \"Premium Model Locked. Use BYOK or Upgrade.\"}\n\n")
// 		return
// 	}

// 	userReq.Message = gov.ModifiedText
// 	govAction := "PERMITTED"
// 	if gov.RuleID != "NONE" { govAction = "REDACTED" }

// 	// 2. CACHE LAYER 0 (REDIS)
// 	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
// 	msgHash := GenerateHash(cleanMsg)
// 	if redisClient != nil {
// 		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
// 			log.Printf("🚀 [HIT] Redis")
// 			streamSimulatedResponse(w, cached)
// 			go LogRequest(userKey, userReq.Model, 200, true, len(userReq.Message)/4, len(cached)/4, CalculateSavings(userReq.Model, len(userReq.Message)/4, len(cached)/4), int(time.Since(startTime).Milliseconds()), userReq.Message, cached, gov.RuleID, govAction)
// 			return
// 		}
// 	}

// 	log.Printf("🐢 [MISS] No Redis match found for: %s", msgHash[:8])

// 	// 3. UNIVERSAL ROUTER
// 	var targetURL string
// 	var payloadBytes []byte
// 	var targetKey string
// 	modelLower := strings.ToLower(userReq.Model)
// 	isGemini := strings.Contains(modelLower, "gemini")

// 	if isGemini {
// 		targetKey = cfg.GeminiKey
// 		if userGeminiKey != "" { targetKey = userGeminiKey }
		
// 		modelID := "gemini-1.5-flash"
// 		if val, ok := discoveredModelMap.Load(targetKey); ok {
// 			modelID = val.(string)
// 		} else {
// 			modelID = DiscoverWorkingModel(targetKey)
// 			discoveredModelMap.Store(targetKey, modelID)
// 		}

// 		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, targetKey)
// 		log.Printf("🛰️ [NATIVE] Adaptive Path -> %s", modelID)
		
// 		geminiPayload := map[string]any{"contents": []map[string]any{{"parts": []map[string]string{{"text": userReq.Message}}}}}
// 		payloadBytes, _ = json.Marshal(geminiPayload)

// 	} else if strings.Contains(modelLower, "llama") {
// 		targetURL = "https://api.groq.com/openai/v1/chat/completions"
// 		targetKey = cfg.GroqKey
// 		if userGroqKey != "" { targetKey = userGroqKey }
// 		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: "llama-3.3-70b-versatile", Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
// 		log.Printf("🔀 [ROUTING] Groq Bridge")
// 	} else {
// 		targetURL = "https://api.openai.com/v1/chat/completions"
// 		targetKey = cfg.OpenAIKey
// 		if userOpenAIKey != "" { targetKey = userOpenAIKey }
// 		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
// 		log.Printf("🔀 [ROUTING] OpenAI Bridge")
// 	}

// 	// 4. EXECUTE
// 	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
// 	req.Header.Set("Content-Type", "application/json")
// 	if !isGemini { req.Header.Set("Authorization", "Bearer "+targetKey) }

// 	client := &http.Client{Timeout: 90 * time.Second}
// 	resp, err := client.Do(req)

// 	if isGemini && (err != nil || (resp != nil && resp.StatusCode == 404)) {
// 		log.Printf("🔄 [HEALING] Connection Failed. Re-probing...")
// 		modelID := DiscoverWorkingModel(targetKey)
// 		discoveredModelMap.Store(targetKey, modelID)
// 		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, targetKey)
// 		req, _ = http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
// 		resp, err = client.Do(req)
// 	}

// 	if err != nil || resp.StatusCode != 200 {
// 		errorBody, _ := io.ReadAll(resp.Body)
// 		log.Printf("❌ [PROVIDER ERROR] %s", string(errorBody))
// 		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(errorBody), gov.RuleID, "FAILED")
// 		return
// 	}
// 	defer resp.Body.Close()

// 	// 5. PARSER
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
// 				var oRes struct { Choices []struct { Delta struct { Content string `json:"content"` } `json:"delta"` } `json:"choices"` }
// 				if err := json.Unmarshal([]byte(cleanLine), &oRes); err == nil && len(oRes.Choices) > 0 {
// 					extracted = oRes.Choices[0].Delta.Content
// 				}
// 			}

// 			if extracted != "" {
// 				fullResponseBuilder.WriteString(extracted)
// 				chunk := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": extracted}}}}
// 				jsonChunk, _ := json.Marshal(chunk)
// 				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
// 				if f, ok := w.(http.Flusher); ok { f.Flush() }
// 				if isGemini { time.Sleep(10 * time.Millisecond) }
// 			}
// 		}
// 	}

// 	// 6. LEDGER
// 	finalText := fullResponseBuilder.String()
// 	if finalText != "" {
// 		if gov.Disclaimer != "" { finalText += gov.Disclaimer }
// 		go func() {
// 			if !usingOwnKey && redisClient != nil {
// 				redisClient.Incr(ctx, "stats:total_requests")
// 				redisClient.Incr(ctx, "stats:cache_misses")
// 			}
// 			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, finalText, 24*time.Hour) }
// 			LogRequest(userKey, userReq.Model, 200, false, len(userReq.Message)/4, len(finalText)/4, 0, int(time.Since(startTime).Milliseconds()), userReq.Message, finalText, gov.RuleID, govAction)
// 		}()
// 	}
// }

// // --- HELPERS ---

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
	"bytes"
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

	// 1. CAPTURE BYOK & AUTH
	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	userGroqKey := r.Header.Get("x-nexus-groq-key")
	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

	// SSE Headers
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
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
	gov := EvaluateConstitution(userReq.Message)
	if !gov.Allowed {
		log.Printf("🚫 [REFUSAL] Constitution Blocked: %s", gov.RuleID)
		streamSimulatedResponse(w, gov.RefusalMsg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")
		return
	}

	// 🛡️ IRON DOME: PREMIUM MODEL GATE
	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
		log.Printf("🛡️ [IRON DOME] Blocking premium model")
		msg := "Premium Model Locked. Use BYOK or Upgrade."
		streamSimulatedResponse(w, msg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, msg, "IRON_DOME", "BLOCKED")
		return
	}

	// Governance Setup for Logs
	userReq.Message = gov.ModifiedText
	govAction := "PERMITTED"
	triggeredRule := "NONE"
	if gov.RuleID != "NONE" {
		govAction = "REDACTED"
		triggeredRule = gov.RuleID
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
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, CalculateSavings(userReq.Model, pT, rT), int(time.Since(startTime).Milliseconds()), userReq.Message, cached, triggeredRule, govAction)
			return
		}
	}

	log.Printf("🐢 [MISS] No local match found for: %s. Checking Semantic Cache...", msgHash[:8])

	// --- 3. VECTOR DB SEARCH (LAYER 1) ---
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)
	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		if err == nil && score > 0.75 {
			log.Printf("⚡ [PINECONE HIT] Score: %.2f", score)
			streamSimulatedResponse(w, cachedAnswer)
			if gov.Disclaimer != "" { streamSimulatedResponse(w, gov.Disclaimer) }

			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cachedAnswer)
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, CalculateSavings(userReq.Model, pT, rT), int(time.Since(startTime).Milliseconds()), userReq.Message, cachedAnswer, triggeredRule, govAction)
			return
		}
	}

	// --- 4. UNIVERSAL ROUTER (SELF-HEALING) ---
	var targetURL string
	var payloadBytes []byte
	var targetKey string
	modelLower := strings.ToLower(userReq.Model)
	isGemini := strings.Contains(modelLower, "gemini")

	if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral") {
		targetURL = "https://api.groq.com/openai/v1/chat/completions"
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: "llama-3.3-70b-versatile", Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] Groq Bridge")

	} else if isGemini {
		targetKey = cfg.GeminiKey
		if userGeminiKey != "" { targetKey = userGeminiKey }
		
		modelID := "gemini-1.5-flash"
		if cached, ok := discoveredModelMap.Load(targetKey); ok { modelID = cached.(string) }
		modelID = strings.TrimPrefix(modelID, "models/")

		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, targetKey)
		geminiPayload := gemini.GeminiRequest{
			Contents: []gemini.Content{{Parts: []gemini.Part{{Text: userReq.Message}}}},
		}
		payloadBytes, _ = json.Marshal(geminiPayload)
		log.Printf("🛰️ [NATIVE] Routing Gemini -> %s", modelID)

	} else {
		targetURL = "https://api.openai.com/v1/chat/completions"
		targetKey = cfg.OpenAIKey
		if userOpenAIKey != "" { targetKey = userOpenAIKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] OpenAI Bridge")
	}

	// --- 5. EXECUTE ---
	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	if !isGemini { req.Header.Set("Authorization", "Bearer "+targetKey) }

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)

	if isGemini && (err != nil || (resp != nil && resp.StatusCode == 404)) {
		log.Printf("🔄 [HEALING] Triggering Re-Probe...")
		workingModel := DiscoverWorkingModel(targetKey)
		workingModel = strings.TrimPrefix(workingModel, "models/")
		discoveredModelMap.Store(targetKey, workingModel)
		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", workingModel, targetKey)
		
		geminiPayload := gemini.GeminiRequest{
			Contents: []gemini.Content{{Parts: []gemini.Part{{Text: userReq.Message}}}},
		}
		payloadBytes, _ = json.Marshal(geminiPayload)
		req, _ = http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
		resp, err = client.Do(req)
	}

	if err != nil || (resp != nil && resp.StatusCode != 200) {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [PROVIDER ERROR] Status: %d | Body: %s", resp.StatusCode, string(body))
		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(body), triggeredRule, "FAILED")
		fmt.Fprintf(w, "data: {\"error\": \"Inference failed\", \"details\": %s}\n\n", string(body))
		return
	}
	defer resp.Body.Close()
	log.Printf("📡 [STREAM] Provider Connected. Capturing...")

	// --- 6. UNIFIED STREAM PARSER ---
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
				var oRes struct { Choices []struct { Delta struct { Content string `json:"content"` } `json:"delta"` } `json:"choices"` }
				if err := json.Unmarshal([]byte(cleanLine), &oRes); err == nil && len(oRes.Choices) > 0 {
					extracted = oRes.Choices[0].Delta.Content
				}
			}

			if extracted != "" {
				fullResponseBuilder.WriteString(extracted)
				chunk := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": extracted}}}}
				jsonChunk, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				if f, ok := w.(http.Flusher); ok { f.Flush() }
				if isGemini { time.Sleep(10 * time.Millisecond) }
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
			if !usingOwnKey && redisClient != nil {
				redisClient.Incr(ctx, "stats:total_requests")
				redisClient.Incr(ctx, "stats:cache_misses")
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
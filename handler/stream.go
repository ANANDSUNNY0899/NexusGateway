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

var discoveredModelMap = sync.Map{}

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	cfg := config.LoadConfig()
	ctx := context.Background()
	userKey := getStreamAPIKey(r)
	redisClient := GetClient()

	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	userGroqKey := r.Header.Get("x-nexus-groq-key")
	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var userReq ChatRequest
	json.NewDecoder(r.Body).Decode(&userReq)
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
	gov := EvaluateConstitution(userReq.Message)
	if !gov.Allowed {
		log.Printf("🚫 [REFUSAL] Blocked by Governance")
		streamSimulatedResponse(w, gov.RefusalMsg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")
		return
	}

	// 🛡️ IRON DOME: PREMIUM GATE
	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
		log.Printf("🛡️ [IRON DOME] Blocking premium model")
		fmt.Fprintf(w, "data: {\"error\": \"Premium Model Locked. Use BYOK or Upgrade.\"}\n\n")
		return
	}

	userReq.Message = gov.ModifiedText
	govAction := "PERMITTED"
	if gov.RuleID != "NONE" { govAction = "REDACTED" }

	// 2. CACHE LAYER 0 (REDIS)
	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
	msgHash := GenerateHash(cleanMsg)
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 [HIT] Redis")
			streamSimulatedResponse(w, cached)
			go LogRequest(userKey, userReq.Model, 200, true, len(userReq.Message)/4, len(cached)/4, CalculateSavings(userReq.Model, len(userReq.Message)/4, len(cached)/4), int(time.Since(startTime).Milliseconds()), userReq.Message, cached, gov.RuleID, govAction)
			return
		}
	}

	log.Printf("🐢 [MISS] No Redis match found for: %s", msgHash[:8])

	// 3. UNIVERSAL ROUTER
	var targetURL string
	var payloadBytes []byte
	var targetKey string
	modelLower := strings.ToLower(userReq.Model)
	isGemini := strings.Contains(modelLower, "gemini")

	if isGemini {
		targetKey = cfg.GeminiKey
		if userGeminiKey != "" { targetKey = userGeminiKey }
		
		modelID := "gemini-1.5-flash"
		if val, ok := discoveredModelMap.Load(targetKey); ok {
			modelID = val.(string)
		} else {
			modelID = DiscoverWorkingModel(targetKey)
			discoveredModelMap.Store(targetKey, modelID)
		}

		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, targetKey)
		log.Printf("🛰️ [NATIVE] Adaptive Path -> %s", modelID)
		
		geminiPayload := map[string]any{"contents": []map[string]any{{"parts": []map[string]string{{"text": userReq.Message}}}}}
		payloadBytes, _ = json.Marshal(geminiPayload)

	} else if strings.Contains(modelLower, "llama") {
		targetURL = "https://api.groq.com/openai/v1/chat/completions"
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: "llama-3.3-70b-versatile", Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] Groq Bridge")
	} else {
		targetURL = "https://api.openai.com/v1/chat/completions"
		targetKey = cfg.OpenAIKey
		if userOpenAIKey != "" { targetKey = userOpenAIKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] OpenAI Bridge")
	}

	// 4. EXECUTE
	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	if !isGemini { req.Header.Set("Authorization", "Bearer "+targetKey) }

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)

	if isGemini && (err != nil || (resp != nil && resp.StatusCode == 404)) {
		log.Printf("🔄 [HEALING] Connection Failed. Re-probing...")
		modelID := DiscoverWorkingModel(targetKey)
		discoveredModelMap.Store(targetKey, modelID)
		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, targetKey)
		req, _ = http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
		resp, err = client.Do(req)
	}

	if err != nil || resp.StatusCode != 200 {
		errorBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [PROVIDER ERROR] %s", string(errorBody))
		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(errorBody), gov.RuleID, "FAILED")
		return
	}
	defer resp.Body.Close()

	// 5. PARSER
	reader := bufio.NewReader(resp.Body)
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)
		if strings.HasPrefix(lineStr, "data: ") {
			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
			if cleanLine == "" || cleanLine == "[DONE]" { 
				fmt.Fprintf(w, "data: [DONE]\n\n")
				break 
			}

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

	// 6. LEDGER
	finalText := fullResponseBuilder.String()
	if finalText != "" {
		if gov.Disclaimer != "" { finalText += gov.Disclaimer }
		go func() {
			if !usingOwnKey && redisClient != nil {
				redisClient.Incr(ctx, "stats:total_requests")
				redisClient.Incr(ctx, "stats:cache_misses")
			}
			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, finalText, 24*time.Hour) }
			LogRequest(userKey, userReq.Model, 200, false, len(userReq.Message)/4, len(finalText)/4, 0, int(time.Since(startTime).Milliseconds()), userReq.Message, finalText, gov.RuleID, govAction)
		}()
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
package handler

import (
	"NexusGateway/config"
	"NexusGateway/handler/gemini" // Now properly used
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

	// Set SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no") 
	w.Header().Set("Cache-Control", "no-cache")
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
			sav := CalculateSavings(userReq.Model, pT, rT)
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()), userReq.Message, cached, triggeredRule, govAction)
			return
		}
	}

	log.Printf("🐢 [MISS] No Redis match. Checking Semantic Cache...")

	// --- 3. VECTOR DB SEARCH (LAYER 1) ---
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)
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

	// --- 4. UNIVERSAL ROUTER (SELF-HEALING) ---
	var targetURL string
	var payloadBytes []byte
	modelLower := strings.ToLower(userReq.Model)
	targetKey := cfg.OpenAIKey
	
	// FIX: Declare isGemini here for global scope within function
	isGemini := strings.Contains(modelLower, "gemini")

	if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral") {
		targetURL = "https://api.groq.com/openai/v1/chat/completions"
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: "llama-3.3-70b-versatile", Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] Groq Bridge")

	} else if isGemini {
		keyToUse := cfg.GeminiKey
		if userGeminiKey != "" { keyToUse = userGeminiKey }
		
		modelID := "gemini-1.5-flash"
		if cached, ok := discoveredModelMap.Load(keyToUse); ok {
			modelID = cached.(string)
			log.Printf("📌 [PINNED] Using: %s", modelID)
		} else {
			modelID = discoverWorkingGemini(keyToUse)
			discoveredModelMap.Store(keyToUse, modelID)
		}

		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, keyToUse)
		
		// Use the Gemini native type from our package
		geminiPayload := gemini.GeminiRequest{
			Contents: []gemini.Content{{Parts: []gemini.Part{{Text: userReq.Message}}}},
		}
		payloadBytes, _ = json.Marshal(geminiPayload)
		log.Printf("🛰️ [ADAPTIVE] Routing to Gemini: %s", modelID)

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
	if !isGemini { 
		req.Header.Set("Authorization", "Bearer "+targetKey) 
	}

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)

	if isGemini && (err != nil || resp.StatusCode != 200) {
		log.Printf("🔄 [HEALING] Re-probing Google API...")
		workingModel := discoverWorkingGemini(targetKey)
		discoveredModelMap.Store(targetKey, workingModel)
		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", workingModel, targetKey)
		req, _ = http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
	}

	if err != nil || resp.StatusCode != 200 {
		errorBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [PROVIDER ERROR] %s", string(errorBody))
		go LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, string(errorBody), triggeredRule, "FAILED")
		return
	}
	defer resp.Body.Close()

	// --- 6. PARSER ---
	reader := bufio.NewReader(resp.Body)
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)
		if strings.HasPrefix(lineStr, "data: ") {
			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
			if cleanLine == "" || cleanLine == "[DONE]" { break }

			var extracted string
			if isGemini {
				var gRes gemini.GeminiResponse // Properly using the import
				if err := json.Unmarshal([]byte(cleanLine), &gRes); err == nil && len(gRes.Candidates) > 0 {
					extracted = gRes.Candidates[0].Content.Parts[0].Text
				}
			} else {
				var oRes struct { Choices []struct { Delta struct { Content string `json:"content"` } `json:"delta"` } `json:"choices"` }
				if err := json.Unmarshal([]byte(cleanLine), &oRes); err == nil && len(oRes.Choices) > 0 {
					extracted = oRes.Choices[0].Delta.Content
				}
			}

			if extracted != "" {
				fullResponseBuilder.WriteString(extracted)
				formatted := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": extracted}}}}
				jsonChunk, _ := json.Marshal(formatted)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				w.(http.Flusher).Flush()
				if isGemini { time.Sleep(10 * time.Millisecond) }
			}
		}
	}

	// --- 7. FINAL LEDGER ---
	finalText := fullResponseBuilder.String()
	if gov.Disclaimer != "" { finalText += gov.Disclaimer }
	
	if finalText != "" {
		go func() {
			if !usingOwnKey && redisClient != nil {
				redisClient.Incr(ctx, "stats:total_requests")
				redisClient.Incr(ctx, "stats:cache_misses")
			}
			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, finalText, 24*time.Hour) }
			if vector != nil && cfg.PineconeKey != "" { SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, finalText) }
			LogRequest(userKey, userReq.Model, 200, false, EstimateTokens(userReq.Message), EstimateTokens(finalText), 0, int(time.Since(startTime).Milliseconds()), userReq.Message, finalText, gov.RuleID, govAction)
		}()
	}
}

// --- ALL HELPERS (Synced) ---

func discoverWorkingGemini(key string) string {
	versions := []string{"v1", "v1beta"}
	for _, ver := range versions {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models?key=%s", ver, key)
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != 200 { continue }
		defer resp.Body.Close()
		var list struct { Models []struct { Name string `json:"name"`; SupportedGenerationMethods []string `json:"supportedGenerationMethods"` } `json:"models"` }
		json.NewDecoder(resp.Body).Decode(&list)
		for _, m := range list.Models {
			for _, method := range m.SupportedGenerationMethods {
				if method == "generateContent" {
					name := strings.TrimPrefix(m.Name, "models/")
					if strings.Contains(name, "1.5-flash") { return name }
				}
			}
		}
	}
	return "gemini-1.5-flash"
}

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
		w.(http.Flusher).Flush()
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
}
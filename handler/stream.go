package handler

import (
	"NexusGateway/config"
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

// Global Discovery Cache for Gemini
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
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

	// Set Professional SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no") 
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0)
		return
	}
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// --- 📥 LOG: REQUEST START ---
	log.Printf("📥 [REQUEST] User: %s... | Model: %s | BYOK: %v", userKey[:6], userReq.Model, usingOwnKey)

	// 2. HYBRID CACHE LAYER 0 (REDIS EXACT)
	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
	msgHash := GenerateHash(cleanMsg)
	
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 [HIT] Redis match serving instantly.")
			streamSimulatedResponse(w, cached)
			
			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cached)
			sav := CalculateSavings(userReq.Model, pT, rT)
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()))
			return
		}
	}

	// --- 🐢 LOG: CACHE MISS ---
	log.Printf("🐢 [MISS] No local match found. Routing to provider...")

	// 3. VECTOR DB SEARCH (LAYER 1)
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)
	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		if err == nil && score > 0.75 {
			log.Printf("⚡ [PINECONE HIT] Score: %.2f", score)
			streamSimulatedResponse(w, cachedAnswer)
			
			pT, rT := EstimateTokens(userReq.Message), EstimateTokens(cachedAnswer)
			sav := CalculateSavings(userReq.Model, pT, rT)
			go LogRequest(userKey, userReq.Model, 200, true, pT, rT, sav, int(time.Since(startTime).Milliseconds()))
			return
		}
	}

	// 4. UNIVERSAL ROUTER (SELF-HEALING)
	var targetURL string
	var payloadBytes []byte
	modelLower := strings.ToLower(userReq.Model)
	targetKey := cfg.OpenAIKey
	isGemini := strings.Contains(modelLower, "gemini")

	if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral") {
		// GROQ ROUTE - Normalized for Browser Success
		targetURL = "https://api.groq.com/openai/v1/chat/completions"
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }

		// Use modern stable model ID
		groqModel := "llama-3.3-70b-versatile"
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: groqModel, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] Groq Bridge -> %s", groqModel)

	} else if isGemini {
		// ADAPTIVE GEMINI ROUTE
		keyToUse := cfg.GeminiKey
		if userGeminiKey != "" { keyToUse = userGeminiKey }
		
		modelID := "gemini-1.5-flash" 
		if cached, ok := discoveredModelMap.Load(keyToUse); ok {
			modelID = cached.(string)
			log.Printf("📌 [PINNED] Using discovered model: %s", modelID)
		}

		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, keyToUse)
		geminiPayload := map[string]any{"contents": []map[string]any{{"parts": []map[string]string{{"text": userReq.Message}}}}}
		payloadBytes, _ = json.Marshal(geminiPayload)
		log.Printf("🛰️ [ROUTING] Gemini Native Bridge")

	} else {
		// OPENAI ROUTE
		targetURL = "https://api.openai.com/v1/chat/completions"
		targetKey = cfg.OpenAIKey
		if userOpenAIKey != "" { targetKey = userOpenAIKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] OpenAI Bridge")
	}

	// 5. EXECUTE PROVIDER HANDSHAKE
	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	if !isGemini { req.Header.Set("Authorization", "Bearer "+targetKey) }

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)

	// AUTO-RECOVERY FOR GEMINI 404
	if isGemini && (err != nil || resp.StatusCode == 404) {
		keyToUse := cfg.GeminiKey
		if userGeminiKey != "" { keyToUse = userGeminiKey }
		workingModel := discoverWorkingGemini(keyToUse)
		discoveredModelMap.Store(keyToUse, workingModel)
		
		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", workingModel, keyToUse)
		req, _ = http.NewRequest("POST", targetURL, bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
	}

	if err != nil || resp.StatusCode != 200 {
		errorBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [PROVIDER ERROR] Status: %d | Body: %s", resp.StatusCode, string(errorBody))
		fmt.Fprintf(w, "data: {\"error\": \"Inference failed\", \"details\": %s}\n\n", string(errorBody))
		return
	}
	defer resp.Body.Close()
	log.Printf("📡 [STREAM] Handshake success. Piping data...")

	// --- 6. UNIFIED STREAM PARSER WITH PACING & HANG FIX ---
	reader := bufio.NewReader(resp.Body)
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)

		if strings.HasPrefix(lineStr, "data: ") {
			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
			
			// 🚀 THE HANG FIX: Break on [DONE] signal
			if cleanLine == "[DONE]" { 
				fmt.Fprintf(w, "data: [DONE]\n\n")
				w.(http.Flusher).Flush()
				break 
			}
			if cleanLine == "" { continue }

			var extractedText string
			if isGemini {
				var gRes struct {
					Candidates []struct { Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"` } `json:"candidates"`
				}
				if err := json.Unmarshal([]byte(cleanLine), &gRes); err == nil && len(gRes.Candidates) > 0 {
					extractedText = gRes.Candidates[0].Content.Parts[0].Text
				}
			} else {
				var oRes struct { Choices []struct { Delta struct { Content string `json:"content"` } `json:"delta"` } `json:"choices"` }
				if err := json.Unmarshal([]byte(cleanLine), &oRes); err == nil && len(oRes.Choices) > 0 {
					extractedText = oRes.Choices[0].Delta.Content
				}
			}

			// --- 🚀 THE SMOOTH-STREAM PACING ---
			if extractedText != "" {
				fullResponseBuilder.WriteString(extractedText)

				words := strings.Split(extractedText, " ")
				for i, word := range words {
					content := word
					if i < len(words)-1 { content += " " }

					formatted := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": content}}}}
					jsonChunk, _ := json.Marshal(formatted)
					fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
					w.(http.Flusher).Flush()

					// Pacing delay: 5ms for large dumps (Gemini), 0ms for word-by-word (OpenAI)
					if len(words) > 1 {
						time.Sleep(5 * time.Millisecond)
					}
				}
			}
		}
	}

	// 7. TELEMETRY LEDGER
	finalText := fullResponseBuilder.String()
	log.Printf("✅ [SUCCESS] Stream finished. Captured: %d chars.", len(finalText))

	if finalText != "" {
		go func() {
			pT, cT := len(userReq.Message)/4, len(finalText)/4
			latency := int(time.Since(startTime).Milliseconds())
			
			log.Printf("📊 [LEDGER] Telemetry: %d tokens | %dms latency", pT+cT, latency)

			if !usingOwnKey && redisClient != nil {
				redisClient.Incr(ctx, "stats:total_requests")
				redisClient.Incr(ctx, "stats:cache_misses")
			}
			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, finalText, 24*time.Hour) }
			if vector != nil && cfg.PineconeKey != "" { SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, finalText) }
			
			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, latency)
		}()
	}
}

// --- HELPERS (Discovery, Keys, Simulated Response) ---

func discoverWorkingGemini(key string) string {
	log.Printf("🔍 [DISCOVERY] Mapping available models for this key...")
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", key)
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	var list struct { Models []struct { Name string `json:"name"` } `json:"models"` }
	json.NewDecoder(resp.Body).Decode(&list)
	
	for _, m := range list.Models {
		name := strings.TrimPrefix(m.Name, "models/")
		if name == "gemini-1.5-flash" { return name }
	}
	for _, m := range list.Models {
		name := strings.TrimPrefix(m.Name, "models/")
		if strings.Contains(name, "flash") { return name }
	}
	return "gemini-pro"
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
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
}
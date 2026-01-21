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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var userReq ChatRequest
	json.NewDecoder(r.Body).Decode(&userReq)
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// --- 📥 LOG: REQUEST START ---
	log.Printf("📥 [REQUEST] User: %s... | Model: %s | BYOK: %v", userKey[:6], userReq.Model, usingOwnKey)

	// 2. HYBRID CACHE LAYER 0
	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
	msgHash := GenerateHash(cleanMsg)
	
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 [CACHE HIT] Redis match found. Serving instantly.")
			streamSimulatedResponse(w, cached)
			return
		}
	}

	// --- 🐢 LOG: CACHE MISS ---
	log.Printf("🐢 [CACHE MISS] No local match for: %s", msgHash[:8])

	// 3. PREPARE VECTOR & PINECONE (Layer 1)
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)
	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		if err == nil && score > 0.75 {
			log.Printf("⚡ [PINECONE HIT] Score: %.2f", score)
			streamSimulatedResponse(w, cachedAnswer)
			return
		}
	}

	// 4. UNIVERSAL ROUTER
	var targetURL string
	var payloadBytes []byte
	modelLower := strings.ToLower(userReq.Model)
	targetKey := cfg.OpenAIKey
	isGemini := strings.Contains(modelLower, "gemini")

	if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral") {
		// GROQ
		targetURL = "https://api.groq.com/openai/v1/chat/completions"
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] Forwarding to Groq Infrastructure...")

	} else if isGemini {
		// ADAPTIVE GEMINI
		keyToUse := cfg.GeminiKey
		if userGeminiKey != "" { keyToUse = userGeminiKey }
		
		modelID := "gemini-1.5-flash"
		if cached, ok := discoveredModelMap.Load(keyToUse); ok {
			modelID = cached.(string)
		}

		targetURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, keyToUse)
		geminiPayload := map[string]any{"contents": []map[string]any{{"parts": []map[string]string{{"text": userReq.Message}}}}}
		payloadBytes, _ = json.Marshal(geminiPayload)
		log.Printf("🛰️ [ROUTING] Forwarding to Gemini Native: %s", modelID)

	} else {
		// OPENAI
		targetURL = "https://api.openai.com/v1/chat/completions"
		targetKey = cfg.OpenAIKey
		if userOpenAIKey != "" { targetKey = userOpenAIKey }
		payloadBytes, _ = json.Marshal(StreamRequestPayload{Model: userReq.Model, Messages: []Message{{Role: "user", Content: userReq.Message}}, Stream: true})
		log.Printf("🔀 [ROUTING] Forwarding to OpenAI Infrastructure...")
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
		return
	}
	defer resp.Body.Close()
	log.Printf("📡 [STREAM] Handshake success. Piping data...")

	// 6. STREAM PARSER
	reader := bufio.NewReader(resp.Body)
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)
		if strings.HasPrefix(lineStr, "data: ") {
			cleanLine := strings.TrimSpace(strings.TrimPrefix(lineStr, "data: "))
			if cleanLine == "" || cleanLine == "[DONE]" { continue }

			var text string
			if isGemini {
				var gRes struct {
					Candidates []struct { Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"` } `json:"candidates"`
				}
				if err := json.Unmarshal([]byte(cleanLine), &gRes); err == nil && len(gRes.Candidates) > 0 {
					text = gRes.Candidates[0].Content.Parts[0].Text
				}
			} else {
				var oRes struct { Choices []struct { Delta struct { Content string `json:"delta"` } `json:"delta"` } `json:"choices"` }
				if err := json.Unmarshal([]byte(cleanLine), &oRes); err == nil && len(oRes.Choices) > 0 {
					text = oRes.Choices[0].Delta.Content
				}
			}

			if text != "" {
				fullResponseBuilder.WriteString(text)
				formatted := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": text}}}}
				jsonChunk, _ := json.Marshal(formatted)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				w.(http.Flusher).Flush()
			}
		}
	}

	// 7. TELEMETRY LEDGER
	finalText := fullResponseBuilder.String()
	log.Printf("✅ [SUCCESS] Stream finished. Captured %d characters.", len(finalText))

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

// --- HELPERS (Discover, Key, Simulated) ---

func discoverWorkingGemini(key string) string {
	log.Printf("🔍 [DISCOVERY] Mapping available models for this API key...")
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", key)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 { return "gemini-1.5-flash" }
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
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
}
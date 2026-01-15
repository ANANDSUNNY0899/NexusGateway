// package handler

// import (
// 	"NexusGateway/config"
// 	"bufio"
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"strings"
// 	"time"
// )

// type StreamRequestPayload struct {
// 	Model    string    `json:"model"`
// 	Messages []Message `json:"messages"`
// 	Stream   bool      `json:"stream"`
// }

// // Helper needed inside this file too
// func getStreamAPIKey(r *http.Request) string {
// 	authHeader := r.Header.Get("Authorization")
// 	if authHeader == "" { return "" }
// 	parts := strings.Split(authHeader, " ")
// 	if len(parts) == 2 { return strings.TrimSpace(parts[1]) }
// 	return ""
// }

// func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
// 	cfg := config.LoadConfig()
// 	ctx := context.Background()
// 	userKey := getStreamAPIKey(r) // Capture Key

// 	// Headers
// 	w.Header().Set("Content-Type", "text/event-stream")
// 	w.Header().Set("Cache-Control", "no-cache")
// 	w.Header().Set("Connection", "keep-alive")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	var userReq ChatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
// 		LogRequest(userKey, "unknown", 400, false) // Log Error
// 		return
// 	}
// 	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

// 	// 🛡️ FIREWALL ACTIVATION
//     original := userReq.Message
//     userReq.Message = RedactPII(original)
//     if original != userReq.Message {
//         log.Println("🛡️ Stream Firewall: PII Redacted")
//     }

// 	redisClient := GetClient()
// 	if redisClient != nil {
// 		redisClient.Incr(ctx, "stats:total_requests")
// 	}

// 	// CHECK CACHE
// 	log.Println("🧠 Stream: Checking Cache...")
// 	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)

// 	if vector != nil && cfg.PineconeKey != "" {
// 		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
// 		if err == nil && score > 0.75 {
// 			log.Println("⚡ STREAM HIT")
//             if redisClient != nil { redisClient.Incr(ctx, "stats:cache_hits") }
			
// 			// Log HIT to DB
// 			LogRequest(userKey, userReq.Model, 200, true)

// 			words := strings.Split(cachedAnswer, " ")
// 			for _, word := range words {
// 				chunk := map[string]any{
// 					"choices": []map[string]any{
// 						{ "delta": map[string]string{ "content": word + " " } },
// 					},
// 				}
// 				jsonChunk, _ := json.Marshal(chunk)
// 				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
// 				w.(http.Flusher).Flush()
// 				time.Sleep(50 * time.Millisecond)
// 			}
// 			fmt.Fprintf(w, "data: [DONE]\n\n")
// 			return
// 		}
// 	}

// 	// CACHE MISS
// 	log.Println("🐢 STREAM MISS")
//     if redisClient != nil { redisClient.Incr(ctx, "stats:cache_misses") }
	
// 	payload := StreamRequestPayload{
// 		Model: userReq.Model,
// 		Messages: []Message{ {Role: "user", Content: userReq.Message} },
// 		Stream: true,
// 	}
// 	jsonBody, _ := json.Marshal(payload)

// 	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

// 	httpClient := &http.Client{}
// 	resp, err := httpClient.Do(req)
// 	if err != nil {
// 		LogRequest(userKey, userReq.Model, 500, false) // Log Error
// 		fmt.Fprintf(w, "data: Error connecting to OpenAI\n\n")
// 		return
// 	}
// 	defer resp.Body.Close()

// 	reader := bufio.NewReader(resp.Body)
// 	fullResponse := "" 

// 	for {
// 		line, err := reader.ReadBytes('\n')
// 		if err != nil { break }
		
// 		lineStr := string(line)
// 		if strings.HasPrefix(lineStr, "data: ") {
// 			w.Write(line)
// 			w.(http.Flusher).Flush()

// 			jsonStr := strings.TrimPrefix(strings.TrimSpace(lineStr), "data: ")
// 			if jsonStr != "[DONE]" {
// 				var chunkData struct {
// 					Choices []struct {
// 						Delta struct { Content string `json:"content"` } `json:"delta"`
// 					} `json:"choices"`
// 				}
// 				if json.Unmarshal([]byte(jsonStr), &chunkData) == nil {
// 					if len(chunkData.Choices) > 0 {
// 						fullResponse += chunkData.Choices[0].Delta.Content
// 					}
// 				}
// 			}
// 		}
// 	}

// 	// Save to Cache
// 	if vector != nil && cfg.PineconeKey != "" && fullResponse != "" {
// 		id := GenerateHash(userReq.Message)
// 		SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, fullResponse)
// 	}

// 	// Log MISS to DB
// 	LogRequest(userKey, userReq.Model, 200, false)
// }







package handler

import (
	"NexusGateway/config"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamRequestPayload struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

func getStreamAPIKey(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" { return "" }
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 { return strings.TrimSpace(parts[1]) }
	return ""
}

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	ctx := context.Background()
	userKey := getStreamAPIKey(r)

	// --- LAYER 2: BYOK HEADERS ---
	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	usingOwnKey := (userOpenAIKey != "")

	// Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		LogRequest(userKey, "unknown", 400, false)
		return
	}
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// --- LAYER 1: IRON DOME ---
	if !usingOwnKey {
		if userReq.Model == "gpt-4" || userReq.Model == "gpt-4o" {
			fmt.Fprintf(w, "data: Error: GPT-4 requires x-nexus-openai-key header\n\n")
			return
		}
	}

	redisClient := GetClient()
	if !usingOwnKey && redisClient != nil {
		redisClient.Incr(ctx, "stats:total_requests")
	}

	// FIREWALL (Security)
	original := userReq.Message
	userReq.Message = RedactPII(original)
	if original != userReq.Message { log.Println("🛡️ Stream Firewall: PII Redacted") }

	// CACHE CHECK
	log.Println("🧠 Stream: Checking Cache...")
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)

	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
		if err == nil && score > 0.75 {
			log.Println("⚡ STREAM HIT")
            if !usingOwnKey && redisClient != nil { redisClient.Incr(ctx, "stats:cache_hits") }
			LogRequest(userKey, userReq.Model, 200, true)

			words := strings.Split(cachedAnswer, " ")
			for _, word := range words {
				chunk := map[string]any{ "choices": []map[string]any{ { "delta": map[string]string{ "content": word + " " } } } }
				jsonChunk, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				w.(http.Flusher).Flush()
				time.Sleep(50 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			return
		}
	}

	// CACHE MISS
	log.Println("🐢 STREAM MISS")
    if !usingOwnKey && redisClient != nil { redisClient.Incr(ctx, "stats:cache_misses") }
	
	// Determine Key
	finalKey := cfg.OpenAIKey
	if userOpenAIKey != "" { finalKey = userOpenAIKey }

	payload := StreamRequestPayload{
		Model: userReq.Model,
		Messages: []Message{ {Role: "user", Content: userReq.Message} },
		Stream: true,
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+finalKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		LogRequest(userKey, userReq.Model, 500, false)
		fmt.Fprintf(w, "data: Error connecting to OpenAI\n\n")
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	fullResponse := "" 

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)
		if strings.HasPrefix(lineStr, "data: ") {
			w.Write(line)
			w.(http.Flusher).Flush()
			jsonStr := strings.TrimPrefix(strings.TrimSpace(lineStr), "data: ")
			if jsonStr != "[DONE]" {
				var chunkData struct {
					Choices []struct {
						Delta struct { Content string `json:"content"` } `json:"delta"`
					} `json:"choices"`
				}
				if json.Unmarshal([]byte(jsonStr), &chunkData) == nil {
					if len(chunkData.Choices) > 0 {
						fullResponse += chunkData.Choices[0].Delta.Content
					}
				}
			}
		}
	}

	if vector != nil && cfg.PineconeKey != "" && fullResponse != "" {
		id := GenerateHash(userReq.Message)
		SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, fullResponse)
	}
	LogRequest(userKey, userReq.Model, 200, false)
}
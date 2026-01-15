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

// type Message struct {
// 	Role    string `json:"role"`
// 	Content string `json:"content"`
// }

// type StreamRequestPayload struct {
// 	Model    string    `json:"model"`
// 	Messages []Message `json:"messages"`
// 	Stream   bool      `json:"stream"`
// }

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
// 	userKey := getStreamAPIKey(r)

// 	// --- LAYER 2: BYOK HEADERS ---
// 	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
// 	usingOwnKey := (userOpenAIKey != "")

// 	w.Header().Set("Content-Type", "text/event-stream")
// 	w.Header().Set("Cache-Control", "no-cache")
// 	w.Header().Set("Connection", "keep-alive")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	var userReq ChatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
// 		LogRequest(userKey, "unknown", 400, false)
// 		return
// 	}
// 	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

// 	// --- LAYER 1: IRON DOME ---
// 	if !usingOwnKey {
// 		if userReq.Model == "gpt-4" || userReq.Model == "gpt-4o" {
// 			fmt.Fprintf(w, "data: Error: GPT-4 requires x-nexus-openai-key header\n\n")
// 			return
// 		}
// 	}

// 	redisClient := GetClient()
// 	if !usingOwnKey && redisClient != nil {
// 		redisClient.Incr(ctx, "stats:total_requests")
// 	}

// 	// FIREWALL
// 	original := userReq.Message
// 	userReq.Message = RedactPII(original)
// 	if original != userReq.Message { log.Println("🛡️ Stream Firewall: PII Redacted") }

// 	// CACHE CHECK
// 	log.Println("🧠 Stream: Checking Cache...")
// 	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)

// 	if vector != nil && cfg.PineconeKey != "" {
// 		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
// 		// Lowered threshold to 0.70 to catch "Hii" vs "Hi"
// 		if err == nil && score > 0.70 {
// 			log.Println("⚡ STREAM HIT: Serving from Pinecone")
//             if !usingOwnKey && redisClient != nil { redisClient.Incr(ctx, "stats:cache_hits") }
// 			LogRequest(userKey, userReq.Model, 200, true)

// 			words := strings.Split(cachedAnswer, " ")
// 			for _, word := range words {
// 				chunk := map[string]any{ "choices": []map[string]any{ { "delta": map[string]string{ "content": word + " " } } } }
// 				jsonChunk, _ := json.Marshal(chunk)
// 				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
// 				w.(http.Flusher).Flush()
// 				time.Sleep(10 * time.Millisecond)
// 			}
// 			fmt.Fprintf(w, "data: [DONE]\n\n")
// 			return
// 		}
// 	}

// 	// CACHE MISS
// 	log.Println("🐢 STREAM MISS: Calling OpenAI...")
//     if !usingOwnKey && redisClient != nil { redisClient.Incr(ctx, "stats:cache_misses") }
	
// 	finalKey := cfg.OpenAIKey
// 	if userOpenAIKey != "" { finalKey = userOpenAIKey }

// 	payload := StreamRequestPayload{
// 		Model: userReq.Model,
// 		Messages: []Message{ {Role: "user", Content: userReq.Message} },
// 		Stream: true,
// 	}
// 	jsonBody, _ := json.Marshal(payload)

// 	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+finalKey)

// 	httpClient := &http.Client{}
// 	resp, err := httpClient.Do(req)
// 	if err != nil {
// 		LogRequest(userKey, userReq.Model, 500, false)
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
// 				// Added error printing here to see if parsing fails
// 				if err := json.Unmarshal([]byte(jsonStr), &chunkData); err == nil {
// 					if len(chunkData.Choices) > 0 {
// 						content := chunkData.Choices[0].Delta.Content
// 						fullResponse += content
// 					}
// 				}
// 			}
// 		}
// 	}

// 	// DEBUG LOG: See what we actually captured
// 	log.Printf("DEBUG: Captured Response Length: %d", len(fullResponse))

// 	// 6. SAVE TO CACHE
// 	if vector != nil && cfg.PineconeKey != "" && fullResponse != "" {
// 		id := GenerateHash(userReq.Message)
// 		// Check for error when saving
// 		err := SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, fullResponse)
// 		if err != nil {
// 			log.Printf("❌ Failed to Save to Pinecone: %v", err)
// 		} else {
// 			log.Println("💾 Stream Saved to Pinecone Successfully")
// 		}
// 	} else {
// 		log.Println("⚠️ Skipping Save: Vector/Key/Response is empty")
// 	}

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

	// FIREWALL
	original := userReq.Message
	userReq.Message = RedactPII(original)
	if original != userReq.Message { log.Println("🛡️ Stream Firewall: PII Redacted") }

	// CACHE CHECK
	log.Println("🧠 Stream: Checking Cache...")
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)

	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
		if err == nil && score > 0.70 {
			log.Println("⚡ STREAM HIT: Serving from Pinecone")
            if !usingOwnKey && redisClient != nil { redisClient.Incr(ctx, "stats:cache_hits") }
			LogRequest(userKey, userReq.Model, 200, true)

			words := strings.Split(cachedAnswer, " ")
			for _, word := range words {
				chunk := map[string]any{ "choices": []map[string]any{ { "delta": map[string]string{ "content": word + " " } } } }
				jsonChunk, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				w.(http.Flusher).Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			return
		}
	}

	// CACHE MISS
	log.Println("🐢 STREAM MISS: Calling OpenAI...")
    if !usingOwnKey && redisClient != nil { redisClient.Incr(ctx, "stats:cache_misses") }
	
	finalKey := cfg.OpenAIKey
	if userOpenAIKey != "" { finalKey = userOpenAIKey }

	// ROUTER LOGIC FOR STREAMING (Only OpenAI format supported for now)
	targetURL := "https://api.openai.com/v1/chat/completions"
	targetKey := finalKey

	if strings.Contains(userReq.Model, "llama") {
		targetURL = "https://api.groq.com/openai/v1/chat/completions"
		targetKey = cfg.GroqKey
	}

	payload := StreamRequestPayload{
		Model: userReq.Model,
		Messages: []Message{ {Role: "user", Content: userReq.Message} },
		Stream: true,
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+targetKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		LogRequest(userKey, userReq.Model, 500, false)
		fmt.Fprintf(w, "data: Error connecting to Provider\n\n")
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	fullResponse := "" 

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil { break }
		lineStr := string(line)
		
		// Pass-through to user
		if strings.HasPrefix(lineStr, "data: ") {
			w.Write(line)
			w.(http.Flusher).Flush()

			// ROBUST PARSING LOGIC
			cleanLine := strings.TrimSpace(lineStr)
			cleanLine = strings.TrimPrefix(cleanLine, "data: ")
			if cleanLine == "[DONE]" { continue }

			var chunkData struct {
				Choices []struct {
					Delta struct { Content string `json:"content"` } `json:"delta"`
				} `json:"choices"`
			}
			
			if err := json.Unmarshal([]byte(cleanLine), &chunkData); err == nil {
				if len(chunkData.Choices) > 0 {
					fullResponse += chunkData.Choices[0].Delta.Content
				}
			}
		}
	}

	// DEBUG LOG
	log.Printf("DEBUG: Captured Length: %d", len(fullResponse))

	// SAVE TO CACHE
	if vector != nil && cfg.PineconeKey != "" && fullResponse != "" {
		id := GenerateHash(userReq.Message)
		SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, fullResponse)
		log.Println("💾 Stream Saved to Pinecone")
	}

	LogRequest(userKey, userReq.Model, 200, false)
}
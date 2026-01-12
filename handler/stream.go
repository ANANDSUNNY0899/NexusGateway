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

type StreamRequestPayload struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	ctx := context.Background()

	// 1. Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 2. Parse Request
	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		return
	}
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

    // 3. LOGGING (Increment Counter)
	redisClient := GetClient()
	if redisClient != nil {
		redisClient.Incr(ctx, "stats:total_requests")
	}

	// 4. CHECK CACHE (Pinecone)
	log.Println("🧠 Stream: Checking Cache...")
	vector, _ := GetEmbedding(userReq.Message, cfg.OpenAIKey)

	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
		// CACHE HIT!
		if err == nil && score > 0.85 {
			log.Println("⚡ STREAM HIT: Serving from Pinecone")
            if redisClient != nil { redisClient.Incr(ctx, "stats:cache_hits") }
			
			// Fake Stream the Cached Answer
			words := strings.Split(cachedAnswer, " ")
			for _, word := range words {
				chunk := map[string]any{
					"choices": []map[string]any{
						{ "delta": map[string]string{ "content": word + " " } },
					},
				}
				jsonChunk, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
				w.(http.Flusher).Flush()
				time.Sleep(50 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			return
		}
	}

	// 5. CACHE MISS (Call OpenAI)
	log.Println("🐢 STREAM MISS: Calling OpenAI...")
    if redisClient != nil { redisClient.Incr(ctx, "stats:cache_misses") }
	
	payload := StreamRequestPayload{
		Model: userReq.Model,
		Messages: []Message{ {Role: "user", Content: userReq.Message} },
		Stream: true,
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
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

	// 6. SAVE TO CACHE
	if vector != nil && cfg.PineconeKey != "" && fullResponse != "" {
		id := GenerateHash(userReq.Message)
		SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, fullResponse)
		log.Println("💾 Stream Saved to Pinecone")
	}
}
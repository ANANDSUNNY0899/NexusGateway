package handler

import (
	"NexusGateway/config"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateHash creates a unique ID for the prompt
func GenerateHash(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

func HandleChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	ctx := context.Background()

	// 1. Parse User Request
	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. CACHE CHECK: Generate Hash
	promptHash := GenerateHash(userReq.Message)
	
	// Try to get from Redis
	client := GetClient()
	if client != nil {
		val, err := client.Get(ctx, promptHash).Result()
		if err == nil {
			// CACHE HIT! Return the stored JSON directly
			log.Println("⚡ CACHE HIT: Serving from Redis")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT") // We add a custom header so you can see it in Postman
			w.Write([]byte(val))
			return
		} else if err != redis.Nil {
			log.Printf("Redis Error: %v", err)
		}
	}

	// 3. CACHE MISS: Call OpenAI
	log.Println("🐢 CACHE MISS: Calling OpenAI...")
	
	openAIPayload := OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "user", Content: userReq.Message},
		},
	}

	payloadBytes, err := json.Marshal(openAIPayload)
	if err != nil {
		http.Error(w, "Error processing payload", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to contact OpenAI", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 4. Save to Redis (TTL: 24 Hours)
	if client != nil && resp.StatusCode == 200 {
		err := client.Set(ctx, promptHash, body, 24*time.Hour).Err()
		if err != nil {
			log.Printf("Failed to save to Redis: %v", err)
		} else {
			log.Println("💾 Saved response to Redis")
		}
	}

	// 5. Return Response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(body)
}
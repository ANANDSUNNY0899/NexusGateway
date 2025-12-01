package handler

import (
	"NexusGateway/config"
	"bytes"
	"context"
	"crypto/sha256" // <--- Added
	"encoding/hex"    // <--- Added
	"encoding/json"
	"io"
	"log"
	"net/http"
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

// <--- RESTORED THIS FUNCTION
func GenerateHash(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

func HandleChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	ctx := context.Background()

	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Generate Embedding
	log.Println("🧠 Generating Embedding...")
	vector, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
	if err != nil {
		log.Printf("Embedding Failed: %v", err)
		http.Error(w, "Failed to generate embedding", http.StatusInternalServerError)
		return
	}

	// 2. SEMANTIC SEARCH (Pinecone)
	if cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		if err == nil {
			log.Printf("🔍 Similarity Score: %.2f", score)
			
			// Threshold: 0.85 means "85% Similar"
			if score > 0.70 {
				log.Println("⚡ SEMANTIC HIT: Serving from Pinecone")
				
				client := GetClient()
				if client != nil {
					client.Incr(ctx, "stats:cache_hits")
				}

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT-SEMANTIC")
				w.Write([]byte(cachedAnswer))
				return
			}
		} else {
			log.Printf("Pinecone Search Error: %v", err)
		}
	}

	// 3. CACHE MISS: Call OpenAI
	log.Println("🐢 CACHE MISS: Calling OpenAI...")
	
	client := GetClient()
	if client != nil {
		client.Incr(ctx, "stats:cache_misses")
	}

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

	// 4. Save Vector + Answer to Pinecone
	if resp.StatusCode == 200 && cfg.PineconeKey != "" {
		id := GenerateHash(userReq.Message)
		err := SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, string(body))
		if err != nil {
			log.Printf("Failed to save to Pinecone: %v", err)
		} else {
			log.Println("💾 Saved Vector to Pinecone")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(body)
}
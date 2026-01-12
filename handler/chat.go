// package handler

// import (
// 	"NexusGateway/config"
// 	"context"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"strings" // Added strings package
// )

// // Request Structure
// type ChatRequest struct {
// 	Message string `json:"message"`
// 	Model   string `json:"model"`
// }

// // Helper: Extract Key from Header
// // func getAPIKey(r *http.Request) string {
// // 	authHeader := r.Header.Get("Authorization")
// // 	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
// // }


// // Helper: Extract Key from Header safely
// func getAPIKey(r *http.Request) string {
// 	authHeader := r.Header.Get("Authorization")
// 	// If empty, return empty string (don't crash)
// 	if authHeader == "" {
// 		return ""
// 	}
// 	// Split by space ("Bearer", "KEY") and take the second part
// 	parts := strings.Split(authHeader, " ")
// 	if len(parts) == 2 {
// 		return strings.TrimSpace(parts[1])
// 	}
// 	return ""
// }



// func GenerateHash(input string) string {
// 	hash := sha256.New()
// 	hash.Write([]byte(input))
// 	return hex.EncodeToString(hash.Sum(nil))
// }

// func HandleChat(w http.ResponseWriter, r *http.Request) {
// 	cfg := config.LoadConfig()
// 	ctx := context.Background()
// 	userKey := getAPIKey(r) // Get the key for logging

// 	// 1. Parse Request
// 	var userReq ChatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if userReq.Model == "" {
// 		userReq.Model = "gpt-3.5-turbo"
// 	}

// 	// 2. Generate Embedding
// 	log.Println("🧠 Generating Embedding...")
// 	vector, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
// 	if err != nil {
// 		log.Printf("Embedding Warning: %v", err)
// 	}

// 	// 3. SEMANTIC SEARCH (Cache Hit)
// 	if vector != nil && cfg.PineconeKey != "" {
// 		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
// 		if err == nil {
// 			log.Printf("🔍 Similarity Score: %.2f", score)
			
// 			if score > 0.85 {
// 				log.Println("⚡ SEMANTIC HIT: Serving from Pinecone")
				
// 				client := GetClient()
// 				if client != nil { client.Incr(ctx, "stats:cache_hits") }

// 				// --- LOGGING (HIT) ---
// 				LogRequest(userKey, userReq.Model, 200, true)
// 				// ---------------------

// 				w.Header().Set("Content-Type", "application/json")
// 				json.NewEncoder(w).Encode(map[string]any{
// 					"choices": []map[string]any{
// 						{ "message": map[string]string{ "content": cachedAnswer } },
// 					},
// 				})
// 				return
// 			}
// 		}
// 	}

// 	// 4. ROUTER (Cache Miss)
// 	log.Printf("🐢 CACHE MISS: Routing request to %s...", userReq.Model)
	
// 	client := GetClient()
// 	if client != nil { client.Incr(ctx, "stats:cache_misses") }

// 	provider, err := GetProvider(userReq.Model, cfg.OpenAIKey, cfg.AnthropicKey)
// 	if err != nil {
// 		http.Error(w, "Invalid Model", http.StatusBadRequest)
// 		return
// 	}

// 	responseText, err := provider.Send(userReq.Message)
// 	if err != nil {
// 		log.Printf("Provider Error: %v", err)
		
// 		// --- LOGGING (ERROR) ---
// 		LogRequest(userKey, userReq.Model, 500, false)
// 		// -----------------------

// 		http.Error(w, "AI Provider Error: "+err.Error(), http.StatusBadGateway)
// 		return
// 	}

// 	// 5. Save to Pinecone
// 	if vector != nil && cfg.PineconeKey != "" {
// 		id := GenerateHash(userReq.Message)
// 		SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, responseText)
// 	}

// 	// --- LOGGING (MISS / SUCCESS) ---
// 	LogRequest(userKey, userReq.Model, 200, false)
// 	// --------------------------------

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]any{
// 		"choices": []map[string]any{
// 			{ "message": map[string]string{ "content": responseText } },
// 		},
// 	})
// }





package handler

import (
	"NexusGateway/config"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type ChatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model"`
}

// Helper: Extract Key from Header safely
func getAPIKey(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func GenerateHash(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

func HandleChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	ctx := context.Background()
	userKey := getAPIKey(r) 

	// 1. Increment Global Counter (Redis)
	// This makes the "Total Requests" number go up immediately
	client := GetClient()
	if client != nil {
		client.Incr(ctx, "stats:total_requests")
	}

	// 2. Parse User Request
	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		// Log Bad Request
		LogRequest(userKey, "unknown", 400, false)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if userReq.Model == "" {
		userReq.Model = "gpt-3.5-turbo"
	}

	// 3. Generate Embedding (For Cache)
	log.Println("🧠 Generating Embedding...")
	vector, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
	if err != nil {
		log.Printf("Embedding Warning: %v", err)
	}

	// 4. CHECK CACHE (Pinecone)
	if vector != nil && cfg.PineconeKey != "" {
		cachedAnswer, score, err := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
		// If found (> 85% match)
		if err == nil && score > 0.85 {
			log.Println("⚡ SEMANTIC HIT")
			
			// Update Redis Hit Counter
			if client != nil { client.Incr(ctx, "stats:cache_hits") }

			// !!! CRITICAL: Log to Database !!!
			LogRequest(userKey, userReq.Model, 200, true)

			// Return Answer
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{ "message": map[string]string{ "content": cachedAnswer } },
				},
			})
			return // STOP HERE
		}
	}

	// 5. CACHE MISS (Call AI Provider)
	log.Printf("🐢 CACHE MISS: Routing to %s", userReq.Model)
	
	// Update Redis Miss Counter
	if client != nil { client.Incr(ctx, "stats:cache_misses") }

	provider, err := GetProvider(userReq.Model, cfg.OpenAIKey, cfg.AnthropicKey)
	if err != nil {
		// Log Internal Error
		LogRequest(userKey, userReq.Model, 400, false)
		http.Error(w, "Invalid Model Config", http.StatusBadRequest)
		return
	}

	responseText, err := provider.Send(userReq.Message)
	if err != nil {
		log.Printf("Provider Error: %v", err)
		
		// !!! CRITICAL: Log the Error !!!
		// This ensures your graph shows the failed attempt
		LogRequest(userKey, userReq.Model, 500, false)

		http.Error(w, "AI Provider Error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// 6. Save new answer to Pinecone
	if vector != nil && cfg.PineconeKey != "" {
		id := GenerateHash(userReq.Message)
		SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, id, vector, responseText)
	}

	// !!! CRITICAL: Log the Success !!!
	LogRequest(userKey, userReq.Model, 200, false)

	// Return Answer
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{ "message": map[string]string{ "content": responseText } },
		},
	})
}
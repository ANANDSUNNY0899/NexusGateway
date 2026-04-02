// package handler

// import (
// 	"NexusGateway/config"
// 	"context"
// 	"encoding/json"
// 	"io"
// 	"log"
// 	"net/http"
// 	"strings"
// 	"time"
// )

// // --- MAIN HANDLER (Non-Streaming) ---

// func HandleChat(w http.ResponseWriter, r *http.Request) {
// 	startTime := time.Now()
// 	cfg := config.LoadConfig()
// 	ctx := context.Background()
// 	userKey := getAPIKey(r)
// 	redisClient := GetClient()

// 	// 1. CAPTURE BYOK HEADERS
// 	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
// 	userGroqKey := r.Header.Get("x-nexus-groq-key")
// 	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
// 	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

// 	// 2. PARSE REQUEST
// 	var userReq ChatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
// 		LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0, "", "", "NONE", "FAILED")
// 		respondWithError(w, "Invalid JSON payload", http.StatusBadRequest)
// 		return
// 	}

// 	if userReq.Model == "" {
// 		userReq.Model = "llama-3.3-70b-versatile"
// 	}

// 	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
// 	gov := EvaluateConstitution(userReq.Message)

// 	if !gov.Allowed {
// 		log.Printf("🚫 Constitution Blocked: %s", gov.RuleID)
// 		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, gov.RefusalMsg, gov.RuleID, "BLOCKED")

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(map[string]any{
// 			"choices": []map[string]any{
// 				{"message": map[string]string{"content": gov.RefusalMsg}},
// 			},
// 		})
// 		return
// 	}

// 	// Apply Redaction
// 	originalMsg := userReq.Message
// 	userReq.Message = gov.ModifiedText
// 	govAction := "PERMITTED"
// 	triggeredRule := "NONE"
// 	if originalMsg != gov.ModifiedText {
// 		govAction = "REDACTED"
// 		triggeredRule = gov.RuleID
// 	}

// 	// 3. IRON DOME (MODEL GATE)
// 	if !usingOwnKey {
// 		if strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3") {
// 			LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, userReq.Message, "Premium Blocked", "NONE", "FAILED")
// 			respondWithError(w, "Premium Model Access Denied", http.StatusForbidden)
// 			return
// 		}
// 	}

// 	// 4. HYBRID CACHE (Layer 0 - Exact Match)
// 	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
// 	msgHash := GenerateHash(cleanMsg)
// 	if redisClient != nil {
// 		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
// 			log.Printf("🚀 Redis Exact Match")
// 			responseText := cached
// 			if gov.Disclaimer != "" { responseText += gov.Disclaimer }

// 			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(responseText)
// 			sav := CalculateSavings(userReq.Model, pT, cT)
// 			lat := int(time.Since(startTime).Milliseconds())

// 			go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat, userReq.Message, responseText, triggeredRule, govAction)

// 			w.Header().Set("Content-Type", "application/json")
// 			json.NewEncoder(w).Encode(map[string]any{
// 				"choices": []map[string]any{
// 					{"message": map[string]string{"content": responseText}},
// 				},
// 			})
// 			return
// 		}
// 	}

// 	// --- 🚀 LAYER 1 (INTENT CACHE) ---
// 	intentKey := GenerateIntentSignature(userReq.Message, cfg.GroqKey)
// 	if redisClient != nil {
// 		if intentCached, _ := redisClient.Get(ctx, "intent:"+intentKey).Result(); intentCached != "" {
// 			log.Printf("🧠 Intent Cache Match!")
// 			responseText := intentCached
// 			if gov.Disclaimer != "" { responseText += gov.Disclaimer }

// 			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(responseText)
// 			sav := CalculateSavings(userReq.Model, pT, cT)
// 			lat := int(time.Since(startTime).Milliseconds())

// 			go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat, userReq.Message, responseText, triggeredRule, govAction)

// 			w.Header().Set("Content-Type", "application/json")
// 			json.NewEncoder(w).Encode(map[string]any{
// 				"choices": []map[string]any{
// 					{"message": map[string]string{"content": responseText}},
// 				},
// 			})
// 			return
// 		}
// 	}



// 	// --- 🌌 LAYER 2 (SEMANTIC PINECONE CACHE) ---
// vector, err := GetEmbedding(userReq.Message, cfg.OpenAIKey)
// if err == nil {
//     avgSim := 0.65
//     // FIXED: Added userReq.Message as the 1st argument to match function signature
//     dynamicThresh := CalculateDynamicThreshold(userReq.Message, 0.70, avgSim)

//     answer, score, searchErr := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
    
//     // Safety check for short messages in logs
//     displayMsg := userReq.Message
//     if len(displayMsg) > 15 { displayMsg = displayMsg[:15] }

//     log.Printf("🔍 [DEBUG] Topic: %s | Score: %.4f | Threshold: %.2f", displayMsg, score, dynamicThresh)
    
//     if searchErr == nil && score >= dynamicThresh {
//         log.Printf("🌌 Pinecone Semantic Match (Score: %.2f)", score)
//         responseText := answer
//         if gov.Disclaimer != "" { responseText += gov.Disclaimer }

//         pT, cT := EstimateTokens(userReq.Message), EstimateTokens(responseText)
//         sav := CalculateSavings(userReq.Model, pT, cT)
//         lat := int(time.Since(startTime).Milliseconds())

//         go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat, userReq.Message, responseText, triggeredRule, govAction)

//         w.Header().Set("Content-Type", "application/json")
//         json.NewEncoder(w).Encode(map[string]any{
//             "choices": []map[string]any{
//                 {"message": map[string]string{"content": responseText}},
//             },
//         })
//         return
//     }
// }

// 	// 5. UNIVERSAL ROUTER
// 	provider := GetProvider(userReq.Model)
// 	targetKey := cfg.OpenAIKey
// 	if strings.Contains(userReq.Model, "gemini") {
// 		targetKey = cfg.GeminiKey
// 		if userGeminiKey != "" { targetKey = userGeminiKey }
// 	} else if strings.Contains(userReq.Model, "llama") {
// 		targetKey = cfg.GroqKey
// 		if userGroqKey != "" { targetKey = userGroqKey }
// 	} else if userOpenAIKey != "" {
// 		targetKey = userOpenAIKey
// 	}

// 	// 6. EXECUTE CALL
// 	req, _ := provider.PrepareRequest(userReq.Message, userReq.Model, targetKey, "")
// 	client := &http.Client{Timeout: 60 * time.Second}
// 	resp, err := client.Do(req)

// 	if err != nil || resp == nil || resp.StatusCode != 200 {
// 		LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, userReq.Message, "Provider Fail", "NONE", "FAILED")
// 		respondWithError(w, "AI Provider failure", http.StatusBadGateway)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	// 7. PARSE RESPONSE
// 	body, _ := io.ReadAll(resp.Body)
// 	var responseText string

// 	if strings.Contains(userReq.Model, "gemini") {
// 		var gRes struct {
// 			Candidates []struct {
// 				Content struct {
// 					Parts []struct { Text string `json:"text"` } `json:"parts"`
// 				} `json:"content"`
// 			} `json:"candidates"`
// 		}
// 		json.Unmarshal(body, &gRes)
// 		if len(gRes.Candidates) > 0 && len(gRes.Candidates[0].Content.Parts) > 0 {
// 			responseText = gRes.Candidates[0].Content.Parts[0].Text
// 		}
// 	} else {
// 		var oRes struct {
// 			Choices []struct {
// 				Message struct { Content string `json:"content"` } `json:"message"`
// 			} `json:"choices"`
// 		}
// 		json.Unmarshal(body, &oRes)
// 		if len(oRes.Choices) > 0 {
// 			responseText = oRes.Choices[0].Message.Content
// 		}
// 	}

// 	// 8. TELEMETRY & CACHE POPULATION
// 	if responseText != "" {
// 		if gov.Disclaimer != "" { responseText += gov.Disclaimer }

// 		go func() {
// 			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(responseText)
// 			lat := int(time.Since(startTime).Milliseconds())
			
// 			if redisClient != nil {
// 				redisClient.Set(ctx, "exact:"+msgHash, responseText, 24*time.Hour)
// 				redisClient.Set(ctx, "intent:"+intentKey, responseText, 24*time.Hour)
// 			}
// 			// if vector != nil {
// 			// 	SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, responseText)
// 			// }
// 			// chat.go logic
// 			if vector != nil {
// 				// This matches the 5-argument wrapper in vector.go
// 				SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, responseText)
// 			}
// 			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, lat, userReq.Message, responseText, triggeredRule, govAction)
// 		}()
// 	}

// 	// 9. RETURN JSON
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]any{
// 		"choices": []map[string]any{
// 			{"message": map[string]string{"content": responseText}},
// 		},
// 	})
// }

// func getAPIKey(r *http.Request) string {
// 	auth := r.Header.Get("Authorization")
// 	parts := strings.Split(auth, " ")
// 	if len(parts) == 2 { return parts[1] }
// 	return ""
// }




package handler

import (
	"NexusGateway/config"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- MAIN HANDLER (Non-Streaming) ---

func HandleChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	cfg := config.LoadConfig()
	ctx := context.Background()
	userKey := getStreamAPIKey(r)
	redisClient := GetClient()

	// 1. CAPTURE BYOK HEADERS
	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	userGroqKey := r.Header.Get("x-nexus-groq-key")
	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

	// 2. PARSE MULTI-TURN REQUEST
	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0, "", "", "NONE", "FAILED")
		respondWithError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if userReq.Model == "" {
		userReq.Model = "llama-3.3-70b-versatile"
	}

	// 🧠 MEMORY LOGIC: Extract the latest message for processing
	if len(userReq.Messages) == 0 {
		respondWithError(w, "No messages provided", http.StatusBadRequest)
		return
	}
	latestIdx := len(userReq.Messages) - 1
	currentPrompt := userReq.Messages[latestIdx].Content

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
	gov := EvaluateConstitution(currentPrompt)

	if !gov.Allowed {
		log.Printf("🚫 Constitution Blocked: %s", gov.RuleID)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, currentPrompt, gov.RefusalMsg, gov.RuleID, "BLOCKED")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": gov.RefusalMsg}},
			},
		})
		return
	}

	// Apply Redaction to the thread
	userReq.Messages[latestIdx].Content = gov.ModifiedText
	currentPrompt = gov.ModifiedText // Update local variable for caching
	govAction := "PERMITTED"
	triggeredRule := "NONE"
	if gov.RuleID != "NONE" {
		govAction = "REDACTED"
		triggeredRule = gov.RuleID
	}

	// 3. IRON DOME (MODEL GATE)
	if !usingOwnKey {
		if strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3") {
			LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, currentPrompt, "Premium Blocked", "NONE", "FAILED")
			respondWithError(w, "Premium Model Access Denied", http.StatusForbidden)
			return
		}
	}

	// 4. HYBRID CACHE (Exact Match on latest query)
	cleanMsg := strings.ToLower(strings.TrimSpace(currentPrompt))
	msgHash := GenerateHash(cleanMsg)
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 Redis Exact Match")
			responseText := cached
			if gov.Disclaimer != "" { responseText += gov.Disclaimer }

			pT, cT := EstimateTokens(currentPrompt), EstimateTokens(responseText)
			sav := CalculateSavings(userReq.Model, pT, cT)
			lat := int(time.Since(startTime).Milliseconds())

			go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat, currentPrompt, responseText, triggeredRule, govAction)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"content": responseText}},
				},
			})
			return
		}
	}

	// --- 🚀 LAYER 1 (INTENT CACHE) ---
	intentKey := GenerateIntentSignature(currentPrompt, cfg.GroqKey)
	if redisClient != nil {
		if intentCached, _ := redisClient.Get(ctx, "intent:"+intentKey).Result(); intentCached != "" {
			log.Printf("🧠 Intent Cache Match!")
			responseText := intentCached
			if gov.Disclaimer != "" { responseText += gov.Disclaimer }

			pT, cT := EstimateTokens(currentPrompt), EstimateTokens(responseText)
			sav := CalculateSavings(userReq.Model, pT, cT)
			lat := int(time.Since(startTime).Milliseconds())

			go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat, currentPrompt, responseText, triggeredRule, govAction)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"content": responseText}},
				},
			})
			return
		}
	}

	// --- 🌌 LAYER 2 (SEMANTIC PINECONE CACHE) ---
	vector, err := GetEmbedding(currentPrompt, cfg.OpenAIKey)
	if err == nil {
		avgSim := 0.65
		dynamicThresh := CalculateDynamicThreshold(currentPrompt, 0.70, avgSim)

		answer, score, searchErr := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		
		if searchErr == nil && score >= dynamicThresh {
			log.Printf("🌌 Pinecone Semantic Match (Score: %.2f)", score)
			responseText := answer
			if gov.Disclaimer != "" { responseText += gov.Disclaimer }

			pT, cT := EstimateTokens(currentPrompt), EstimateTokens(responseText)
			sav := CalculateSavings(userReq.Model, pT, cT)
			lat := int(time.Since(startTime).Milliseconds())

			go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat, currentPrompt, responseText, triggeredRule, govAction)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"content": responseText}},
				},
			})
			return
		}
	}

	// 5. UNIVERSAL ROUTER
	provider := GetProvider(userReq.Model)
	targetKey := cfg.OpenAIKey
	if strings.Contains(userReq.Model, "gemini") {
		targetKey = cfg.GeminiKey
		if userGeminiKey != "" { targetKey = userGeminiKey }
	} else if strings.Contains(userReq.Model, "llama") {
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }
	} else if userOpenAIKey != "" {
		targetKey = userOpenAIKey
	}

	// 6. EXECUTE CALL (Now passing the full Messages slice)
	// req, _ := provider.PrepareRequest(userReq.Messages, userReq.Model, targetKey, "")

	// Correct call for non-streaming HandleChat
    req, _ := provider.PrepareRequest(userReq.Messages, userReq.Model, targetKey, "")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)

	if err != nil || resp == nil || resp.StatusCode != 200 {
		LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, currentPrompt, "Provider Fail", "NONE", "FAILED")
		respondWithError(w, "AI Provider failure", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 7. PARSE RESPONSE
	body, _ := io.ReadAll(resp.Body)
	var responseText string

	// (Unified Parsing Logic - keeping Gemini support)
	if strings.Contains(userReq.Model, "gemini") {
		var gRes struct {
			Candidates []struct {
				Content struct {
					Parts []struct { Text string `json:"text"` } `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		json.Unmarshal(body, &gRes)
		if len(gRes.Candidates) > 0 && len(gRes.Candidates[0].Content.Parts) > 0 {
			responseText = gRes.Candidates[0].Content.Parts[0].Text
		}
	} else {
		var oRes struct {
			Choices []struct {
				Message struct { Content string `json:"content"` } `json:"message"`
			} `json:"choices"`
		}
		json.Unmarshal(body, &oRes)
		if len(oRes.Choices) > 0 {
			responseText = oRes.Choices[0].Message.Content
		}
	}

	// 8. TELEMETRY & CACHE POPULATION
	if responseText != "" {
		if gov.Disclaimer != "" { responseText += gov.Disclaimer }

		go func() {
			pT, cT := EstimateTokens(currentPrompt), EstimateTokens(responseText)
			lat := int(time.Since(startTime).Milliseconds())
			
			if redisClient != nil {
				redisClient.Set(ctx, "exact:"+msgHash, responseText, 24*time.Hour)
				redisClient.Set(ctx, "intent:"+intentKey, responseText, 24*time.Hour)
			}
			if vector != nil {
				SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, responseText)
			}
			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, lat, currentPrompt, responseText, triggeredRule, govAction)
		}()
	}

	// 9. RETURN JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": responseText}},
		},
	})
}
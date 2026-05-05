
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

// --- MAIN HANDLER (Sovereign Edition v3.2) ---

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

	// 🧠 MEMORY LOGIC: Extract the latest message
	if len(userReq.Messages) == 0 {
		respondWithError(w, "No messages provided", http.StatusBadRequest)
		return
	}
	latestIdx := len(userReq.Messages) - 1
	currentPrompt := userReq.Messages[latestIdx].Content

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE & TIERING ---
	
	// A. CONSTITUTIONAL CHECK
	gov := EvaluateConstitution(currentPrompt)
	if !gov.Allowed {
		log.Printf("🚫 Constitution Blocked: %s", gov.RuleID)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, currentPrompt, gov.RefusalMsg, gov.RuleID, "BLOCKED")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": gov.RefusalMsg}}},
		})
		return
	}

	// B. COMPLEXITY SCORER (Autonomous Cost Optimization)
	// If the user didn't specify a model, or if we want to force-optimize
	originalModel := userReq.Model
	if userReq.Model == "" || !usingOwnKey {
		userReq.Model = DetermineModelTier(currentPrompt) 
		if originalModel != "" && userReq.Model != originalModel {
			log.Printf("🔄 Tiering: Optimized %s -> %s", originalModel, userReq.Model)
		}
	}

	// Apply Redaction/Modification from Constitution
	userReq.Messages[latestIdx].Content = gov.ModifiedText
	currentPrompt = gov.ModifiedText 

	// 3. IRON DOME (MODEL GATE)
	if !usingOwnKey {
		if strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3") {
			LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, currentPrompt, "Premium Blocked", "NONE", "FAILED")
			respondWithError(w, "Premium Model Access Denied", http.StatusForbidden)
			return
		}
	}

	// 4. THE MONEY VAULT (HYBRID CACHE LAYERS)
	cleanMsg := strings.ToLower(strings.TrimSpace(currentPrompt))
	msgHash := GenerateHash(cleanMsg)
	
	if redisClient != nil {
		// Layer 0: Exact Match
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			serveCachedResponse(w, userKey, userReq.Model, currentPrompt, cached, gov.Disclaimer, startTime, "REDIS_EXACT")
			return
		}

		// Layer 1: Intent Match
		intentKey := GenerateIntentSignature(currentPrompt, cfg.GroqKey)
		if intentCached, _ := redisClient.Get(ctx, "intent:"+intentKey).Result(); intentCached != "" {
			serveCachedResponse(w, userKey, userReq.Model, currentPrompt, intentCached, gov.Disclaimer, startTime, "REDIS_INTENT")
			return
		}
	}

	// Layer 2: Semantic Pinecone Match
	vector, err := GetEmbedding(currentPrompt, cfg.OpenAIKey)
	if err == nil {
		answer, score, searchErr := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		dynamicThresh := CalculateDynamicThreshold(currentPrompt, 0.70, 0.65)
		if searchErr == nil && score >= dynamicThresh {
			serveCachedResponse(w, userKey, userReq.Model, currentPrompt, answer, gov.Disclaimer, startTime, "PINECONE_SEMANTIC")
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

	// 6. EXECUTE PROVIDER CALL
	req, _ := provider.PrepareRequest(userReq.Messages, userReq.Model, targetKey, "")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)

	if err != nil || resp == nil || resp.StatusCode != 200 {
		LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0, currentPrompt, "Provider Fail", "NONE", "FAILED")
		respondWithError(w, "AI Provider failure", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 7. PARSE & UNIFY RESPONSE
	body, _ := io.ReadAll(resp.Body)
	log.Printf("DEBUG: Raw Provider Response: %s", string(body))
	responseText := parseProviderResponse(userReq.Model, body)

	// 8. TELEMETRY & CACHE POPULATION
	if responseText != "" {
		if gov.Disclaimer != "" { responseText += gov.Disclaimer }
		go func() {
			pT, cT := EstimateTokens(currentPrompt), EstimateTokens(responseText)
			lat := int(time.Since(startTime).Milliseconds())
			
			// Populate the Money Vault for the next user
			if redisClient != nil {
				redisClient.Set(ctx, "exact:"+msgHash, responseText, 24*time.Hour)
				intentKey := GenerateIntentSignature(currentPrompt, cfg.GroqKey)
				redisClient.Set(ctx, "intent:"+intentKey, responseText, 24*time.Hour)
			}
			if vector != nil {
				SaveToPinecone(cfg.PineconeHost, cfg.PineconeKey, msgHash, vector, responseText)
			}
			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, lat, currentPrompt, responseText, "NONE", "PERMITTED")
		}()
	}

	// 9. RETURN JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"content": responseText}}},
	})
}

// --- HELPER FUNCTIONS FOR CLEANER CODE ---

func serveCachedResponse(w http.ResponseWriter, userKey, model, prompt, cached, disclaimer string, start time.Time, source string) {
	log.Printf("💰 Sovereign Cache Hit: %s", source)
	res := cached + disclaimer
	pT, cT := EstimateTokens(prompt), EstimateTokens(res)
	sav := CalculateSavings(model, pT, cT)
	lat := int(time.Since(start).Milliseconds())

	go LogRequest(userKey, model, 200, true, pT, cT, sav, lat, prompt, res, "NONE", "CACHE_HIT")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Nexus-Source", source)
	json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"content": res}}},
	})
}



func parseProviderResponse(model string, body []byte) string {
	bodyStr := string(body)

	// 🛡️ 1. STREAM INTERCEPTOR (Glues chunked data together)
	if strings.Contains(bodyStr, "data: ") {
		var fullContent strings.Builder
		lines := strings.Split(bodyStr, "\n")
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				line = strings.TrimPrefix(line, "data: ")
				if line == "[DONE]" {
					continue
				}

				// Catch the "delta" pieces from the stream
				var chunk struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				
				if err := json.Unmarshal([]byte(line), &chunk); err == nil && len(chunk.Choices) > 0 {
					fullContent.WriteString(chunk.Choices[0].Delta.Content)
				}
			}
		}
		
		if fullContent.Len() > 0 {
			return fullContent.String()
		}
	}

	// 🛡️ 2. Handle standard Gemini Structure (Block)
	if strings.Contains(model, "gemini") {
		var gRes struct {
			Candidates []struct {
				Content struct {
					Parts []struct { Text string `json:"text"` } `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		json.Unmarshal(body, &gRes)
		if len(gRes.Candidates) > 0 && len(gRes.Candidates[0].Content.Parts) > 0 {
			return gRes.Candidates[0].Content.Parts[0].Text
		}
	}

	// 🛡️ 3. Handle standard OpenAI/Groq/DeepSeek Structure (Block)
	var oRes struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}

	json.Unmarshal(body, &oRes)
	if len(oRes.Choices) > 0 {
		content := oRes.Choices[0].Message.Content
		if content == "" && oRes.Choices[0].Message.ReasoningContent != "" {
			return oRes.Choices[0].Message.ReasoningContent
		}
		return content
	}

	return ""
}
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
	userKey := getAPIKey(r)
	redisClient := GetClient()

	// 1. CAPTURE BYOK HEADERS
	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	userGroqKey := r.Header.Get("x-nexus-groq-key")
	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "")

	// 2. PARSE REQUEST
	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0)
		respondWithError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// 3. IRON DOME (MODEL GATE)
	if !usingOwnKey {
		if strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3") {
			LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0)
			respondWithError(w, "Premium Model Access Denied. Use BYOK or Upgrade.", http.StatusForbidden)
			return
		}
	}

	// 4. HYBRID CACHE (Layer 0)
	cleanMsg := strings.ToLower(strings.TrimSpace(userReq.Message))
	msgHash := GenerateHash(cleanMsg)
	
	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 CHAT CACHE HIT: %s", cleanMsg)
			
			pT := EstimateTokens(userReq.Message)
			cT := EstimateTokens(cached)
			sav := CalculateSavings(userReq.Model, pT, cT)
			lat := int(time.Since(startTime).Milliseconds())

			go LogRequest(userKey, userReq.Model, 200, true, pT, cT, sav, lat)
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]string{"content": cached}}},
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
	} else if strings.Contains(userReq.Model, "llama") || strings.Contains(userReq.Model, "mixtral") {
		targetKey = cfg.GroqKey
		if userGroqKey != "" { targetKey = userGroqKey }
	} else {
		if userOpenAIKey != "" { targetKey = userOpenAIKey }
	}

	// 6. EXECUTE CALL
	req, _ := provider.PrepareRequest(userReq.Message, userReq.Model, targetKey)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	
	if err != nil || resp.StatusCode != 200 {
		LogRequest(userKey, userReq.Model, 500, false, 0, 0, 0, 0)
		respondWithError(w, "AI Provider failure", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 7. PARSE FULL RESPONSE
	body, _ := io.ReadAll(resp.Body)
	var responseText string
	
	if strings.Contains(userReq.Model, "gemini") {
		var gRes struct { Candidates []struct { Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"` } `json:"candidates"` }
		json.Unmarshal(body, &gRes)
		if len(gRes.Candidates) > 0 { responseText = gRes.Candidates[0].Content.Parts[0].Text }
	} else {
		var oRes struct { Choices []struct { Message struct { Content string `json:"content"` } `json:"message"` } `json:"choices"` }
		json.Unmarshal(body, &oRes)
		if len(oRes.Choices) > 0 { responseText = oRes.Choices[0].Message.Content }
	}

	// 8. TELEMETRY & BACKGROUND PERSISTENCE
	if responseText != "" {
		go func() {
			pT, cT := EstimateTokens(userReq.Message), EstimateTokens(responseText)
			lat := int(time.Since(startTime).Milliseconds())
			
			if redisClient != nil { redisClient.Set(ctx, "exact:"+msgHash, responseText, 24*time.Hour) }
			LogRequest(userKey, userReq.Model, 200, false, pT, cT, 0, lat)
		}()
	}

	// 9. RETURN UNIFIED JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"content": responseText}}},
	})
}

// --- HELPER ---
func getAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	parts := strings.Split(auth, " ")
	if len(parts) == 2 { return parts[1] }
	return ""
}
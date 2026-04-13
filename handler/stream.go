
package handler

import (
	"NexusGateway/config"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Global Discovery Cache (Declared only once)
var discoveredModelMap = sync.Map{}

/* // UNCOMMENT ONLY IF YOU DELETED THESE FROM types.go
type Message struct {
	Role    string `json:"role"`    
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
*/

// --- MAIN HANDLER ---

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    cfg := config.LoadConfig()
    ctx := context.Background()
    userKey := getStreamAPIKey(r)
    redisClient := GetClient() // Only call this once

    // 1. SET HEADERS (The Foundation)
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("X-Accel-Buffering", "no")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // 2. DECODE REQUEST (Crucial: Do this before priming the pipe)
    var userReq ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
        LogRequest(userKey, "unknown", 400, false, 0, 0, 0, 0, "", "", "NONE", "FAILED")
        w.WriteHeader(http.StatusBadRequest) // Send 400 if JSON is trash
        return
    }

    // 3. PRIME THE PIPE (The Handshake)
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
        return
    }

    // Now we officially start the 200 stream
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, ": nexus-handshake-active\n\n")
    flusher.Flush() 

    // 4. MEMORY LOGIC & PROMPT EXTRACTION
    if len(userReq.Messages) == 0 { return }
    latestIdx := len(userReq.Messages) - 1
    currentPrompt := userReq.Messages[latestIdx].Content

    if userReq.Model == "" {
        userReq.Model = "llama-3.3-70b-versatile"
    }

	// 1. CAPTURE BYOK & AUTH HEADERS
	userOpenAIKey := r.Header.Get("x-nexus-openai-key")
	userGroqKey := r.Header.Get("x-nexus-groq-key")
	userGeminiKey := r.Header.Get("x-nexus-gemini-key")
	userAnthropicKey := r.Header.Get("x-nexus-anthropic-key")
	userDeepSeekKey := r.Header.Get("x-nexus-deepseek-key")
	usingOwnKey := (userOpenAIKey != "" || userGroqKey != "" || userGeminiKey != "" || userAnthropicKey != "" || userDeepSeekKey != "")

	// --- 🏛️ PHASE 1: SOVEREIGN GOVERNANCE ---
	gov := EvaluateConstitution(currentPrompt)
	if !gov.Allowed {
		log.Printf("🚫 [REFUSAL] Constitution Blocked: %s", gov.RuleID)
		streamSimulatedResponse(w, gov.RefusalMsg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, currentPrompt, gov.RefusalMsg, gov.RuleID, "BLOCKED")
		return
	}

	// Apply redaction to the message slice for the AI to see
	userReq.Messages[latestIdx].Content = gov.ModifiedText
	currentPrompt = gov.ModifiedText 
	govAction := "PERMITTED"
	triggeredRule := "NONE"
	if gov.RuleID != "NONE" {
		govAction = "REDACTED"
		triggeredRule = gov.RuleID
	}

	// 🛡️ IRON DOME: PREMIUM MODEL GATE
	if !usingOwnKey && (strings.Contains(userReq.Model, "gpt-4") || strings.Contains(userReq.Model, "claude-3")) {
		msg := "Premium Model Locked. Use BYOK or Upgrade."
		streamSimulatedResponse(w, msg)
		go LogRequest(userKey, userReq.Model, 403, false, 0, 0, 0, 0, currentPrompt, msg, "IRON_DOME", "BLOCKED")
		return
	}

	log.Printf("📥 [REQUEST] User: %s... | Model: %s | Messages: %d", userKey[:6], userReq.Model, len(userReq.Messages))

	// --- 2. HYBRID CACHE LAYER ---
	cleanMsg := strings.ToLower(strings.TrimSpace(currentPrompt))
	msgHash := GenerateHash(cleanMsg)
	intentKey := GenerateIntentSignature(currentPrompt, cfg.GroqKey)

	if redisClient != nil {
		if cached, _ := redisClient.Get(ctx, "exact:"+msgHash).Result(); cached != "" {
			log.Printf("🚀 [HIT] Redis serving instantly.")
			streamSimulatedResponse(w, cached)
			return
		}
	}

	var vector []float32
	var err error
	vector, err = GetEmbedding(currentPrompt, cfg.OpenAIKey)
	if err == nil {
		dynamicThresh := CalculateDynamicThreshold(currentPrompt, 0.70, 0.65)
		answer, score, searchErr := SearchPinecone(cfg.PineconeHost, cfg.PineconeKey, vector)
		if searchErr == nil && score >= dynamicThresh && answer != "" {
			log.Printf("🌌 Pinecone Semantic Match (Score: %.2f)", score)
			streamSimulatedResponse(w, answer)
			return
		}
	}

	// --- 🚀 3. SOVEREIGN ROUTER ---
	provider := GetProvider(userReq.Model)
	var activeKey string
	modelLower := strings.ToLower(userReq.Model)

	switch {
	case strings.Contains(modelLower, "deepseek"):
		activeKey = cfg.DeepSeekKey
		if userDeepSeekKey != "" { activeKey = userDeepSeekKey }
	case strings.Contains(modelLower, "gemini"):
		activeKey = cfg.GeminiKey
		if userGeminiKey != "" { activeKey = userGeminiKey }
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral"):
		activeKey = cfg.GroqKey
		if userGroqKey != "" { activeKey = userGroqKey }
	default:
		activeKey = cfg.OpenAIKey
		if userOpenAIKey != "" { activeKey = userOpenAIKey }
	}

	// PASSING FULL HISTORY TO ADAPTER
	req, err := provider.PrepareRequest(userReq.Messages, userReq.Model, activeKey, "v1")
	if err != nil {
		log.Printf("❌ [ROUTER ERROR] %v", err)
		return
	}

	// --- 🚀 4. EXECUTION & SELF-HEALING ---
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)

	if strings.Contains(modelLower, "gemini") && (err != nil || (resp != nil && resp.StatusCode == 404)) {
		workingModel := DiscoverWorkingModel(activeKey)
		discoveredModelMap.Store(activeKey, workingModel)
		req, _ = provider.PrepareRequest(userReq.Messages, workingModel, activeKey, "v1beta")
		resp, err = client.Do(req)
	}

	if err != nil || (resp != nil && resp.StatusCode != 200) {
		fmt.Fprintf(w, "data: {\"error\": \"Provider unavailable\"}\n\n")
		return
	}
	defer resp.Body.Close()

	// --- 🚀 5. UNIFIED STREAM PARSER ---
    // Use a custom scanner with a larger buffer (important for long R1 reasoning tokens)
    scanner := bufio.NewScanner(resp.Body)
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, 1024*1024) 

    var fullResponseBuilder strings.Builder
    flusher, isFlusher := w.(http.Flusher)

    // --- 🚀 5. UNIFIED STREAM PARSER ---
    // 1. Initialize Scanner & Buffer
    scanner := bufio.NewScanner(resp.Body)
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, 1024*1024) 

    var fullResponseBuilder strings.Builder
    flusher, _ := w.(http.Flusher) // Use '_' to avoid "isFlusher declared and not used"

    // 2. Start the Scan Loop
    for scanner.Scan() {
        line := scanner.Text() // Re-using 'line' is fine here as it's scoped to the loop
        if line == "" { continue }

        // Use the provider adapter to extract the text
        extracted, isDone := provider.ParseStreamChunk(line)
        
        if isDone {
            fmt.Fprintf(w, "data: [DONE]\n\n")
            if flusher != nil { flusher.Flush() }
            break
        }

        if extracted != "" {
            fullResponseBuilder.WriteString(extracted)

            formatted := map[string]any{
                "choices": []map[string]any{
                    {
                        "delta": map[string]any{
                            "content": extracted,
                        },
                    },
                },
            }

            jsonChunk, _ := json.Marshal(formatted)
            fmt.Fprintf(w, "data: %s\n\n", jsonChunk)
            
            if flusher != nil { flusher.Flush() }
        }
    }

    if err := scanner.Err(); err != nil {
        log.Printf("🚨 Scanner Error: %v", err)
    }

	// --- 💾 6. LEDGER & TELEMETRY ---
	finalText := fullResponseBuilder.String()
	if finalText != "" {
		IncrementUsage(userKey)
		go func(reply string, v []float32) {
			pT, rT, cost := provider.GetPricing(currentPrompt, reply, userReq.Model)
			latency := int(time.Since(startTime).Milliseconds())

			LogRequest(userKey, userReq.Model, 200, false, pT, rT, cost, latency, currentPrompt, reply, triggeredRule, govAction)
			if redisClient != nil {
				redisClient.Set(ctx, "exact:"+msgHash, reply, 24*time.Hour)
			}
			if v != nil {
				sparseIndices, sparseValues := GenerateSparseVector(currentPrompt)
				SaveToPineconeHybrid(cfg.PineconeHost, cfg.PineconeKey, intentKey, v, sparseIndices, sparseValues, reply)
			}
		}(finalText, vector)
	}
}

// --- 🛠️ SOVEREIGN HELPERS ---

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
		if f, ok := w.(http.Flusher); ok { f.Flush() }
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
}
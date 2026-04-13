

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"NexusGateway/handler/gemini"
)

type GeminiAdapter struct{}

// Ensure this matches the 4-argument signature in the interface

func (g *GeminiAdapter) PrepareRequest(messages []Message, model, key, version string) (*http.Request, error) {
    modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
    if version == "" { version = "v1" }
    
    url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)
    
    // Map Nexus Messages to Gemini Contents
    var geminiContents []gemini.Content
    for _, m := range messages {
        role := m.Role
        if role == "assistant" { role = "model" } // Gemini uses "model" instead of "assistant"
        
        geminiContents = append(geminiContents, gemini.Content{
            Role:  role,
            Parts: []gemini.Part{{Text: m.Content}},
        })
    }

    payload := gemini.GeminiRequest{
        Contents: geminiContents,
    }
    
    body, _ := json.Marshal(payload)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil { return nil, err }
    req.Header.Set("Content-Type", "application/json")
    return req, nil
}

func (a *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
    // 1. Clean the line
    line = strings.TrimSpace(line)
    if line == "" || line == "[" || line == "]" || line == "," {
        return "", false
    }

    // 2. Define Gemini's specific JSON Schema
    var chunk struct {
        Candidates []struct {
            Content struct {
                Parts []struct {
                    Text string `json:"text"`
                } `json:"parts"`
            } `json:"content"`
        } `json:"candidates"`
    }

    // 3. Parse the raw JSON
    if err := json.Unmarshal([]byte(line), &chunk); err != nil {
        return "", false
    }

    // 4. Extract the nested text
    if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
        return chunk.Candidates[0].Content.Parts[0].Text, false
    }

    return "", false
}

// 🚀 FIXED: Added missing GetPricing method
func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
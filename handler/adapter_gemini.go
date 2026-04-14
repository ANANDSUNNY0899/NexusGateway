

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
    
    // 🔥 FIX 1: Use v1beta for better streaming support if not specified
    if version == "" { version = "v1beta" }
    
    url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)
    
    var geminiContents []gemini.Content
    var systemInstruction *gemini.Content

    for _, m := range messages {
        // 🔥 FIX 2: Gemini handles System Prompts in a separate top-level field
        if m.Role == "system" {
            systemInstruction = &gemini.Content{
                Parts: []gemini.Part{{Text: m.Content}},
            }
            continue
        }

        role := m.Role
        if role == "assistant" { role = "model" }
        
        geminiContents = append(geminiContents, gemini.Content{
            Role:  role,
            Parts: []gemini.Part{{Text: m.Content}},
        })
    }

    payload := gemini.GeminiRequest{
        Contents:          geminiContents,
        SystemInstruction: systemInstruction, // Add this to your GeminiRequest struct
    }
    
    body, _ := json.Marshal(payload)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil { return nil, err }
    req.Header.Set("Content-Type", "application/json")
    return req, nil
}

func (a *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
    // 1. Aggressive Cleaning
    line = strings.TrimSpace(line)
    
    line = strings.TrimPrefix(line, "[")
    line = strings.TrimPrefix(line, ",")
    line = strings.TrimSuffix(line, "]")
    line = strings.TrimSpace(line)

    if line == "" {
        return "", false
    }

    // 2. Define Schema (Correct as per your types.go)
    var chunk struct {
        Candidates []struct {
            Content struct {
                Parts []struct {
                    Text string `json:"text"`
                } `json:"parts"`
            } `json:"content"`
        } `json:"candidates"`
    }

    // 3. Parse with error logging (Check Railway logs for this!)
    if err := json.Unmarshal([]byte(line), &chunk); err != nil {
        // log.Printf("Gemini Parse Error: %v | Raw: %s", err, line)
        return "", false
    }

    // 4. Extraction
    if len(chunk.Candidates) > 0 && 
       len(chunk.Candidates[0].Content.Parts) > 0 {
        return chunk.Candidates[0].Content.Parts[0].Text, false
    }

    return "", false
}

// 🚀 FIXED: Added missing GetPricing method
func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
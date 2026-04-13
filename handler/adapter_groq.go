package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// GroqAdapter implements the AIProvider interface
type GroqAdapter struct{}

func (a *GroqAdapter) PrepareRequest(messages []Message, model, key, version string) (*http.Request, error) {
	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *GroqAdapter) ParseStreamChunk(line string) (string, bool) {
    if strings.Contains(line, "[DONE]") {
        return "", true
    }

    // 1. Clean the SSE prefix
    raw := strings.TrimPrefix(line, "data: ")
    raw = strings.TrimSpace(raw)
    if raw == "" { return "", false }

    // 2. Exact Groq/OpenAI Schema
    var chunk struct {
        Choices []struct {
            Delta struct {
                Content string `json:"content"` // 🔥 MUST BE LOWERCASE
            } `json:"delta"`
        } `json:"choices"`
    }

    if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
        return "", false 
    }

    if len(chunk.Choices) > 0 {
        return chunk.Choices[0].Delta.Content, false
    }

    return "", false
}

func (a *GroqAdapter) GetPricing(p, r, m string) (int, int, float64) {
	promptTokens := len(p) / 4
	resTokens := len(r) / 4
	// Groq Llama 3.3 70b pricing
	cost := (float64(promptTokens+resTokens) / 1000000.0) * 0.59
	return promptTokens, resTokens, cost
}
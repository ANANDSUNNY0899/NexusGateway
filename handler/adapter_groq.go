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
    // 1. Handle the End of Stream
    if strings.Contains(line, "[DONE]") {
        return "", true
    }

    // 2. Clean the SSE prefix
    rawJSON := strings.TrimPrefix(line, "data: ")
    rawJSON = strings.TrimSpace(rawJSON)

    if rawJSON == "" {
        return "", false
    }

    // 3. Define the exact Groq Schema
    var chunk struct {
        Choices []struct {
            Delta struct {
                Content string `json:"content"`
            } `json:"delta"`
        } `json:"choices"`
    }

    // 4. Parse
    if err := json.Unmarshal([]byte(rawJSON), &chunk); err != nil {
        // If it's not JSON, it's a heartbeat or error, skip it
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
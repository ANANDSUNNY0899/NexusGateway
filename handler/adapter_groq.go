package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type GroqAdapter struct{}

// 1. PrepareRequest: Logic to build the Groq API call
func (a *GroqAdapter) PrepareRequest(messages []Message, model, key, version string) (*http.Request, error) {
	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// 2. ParseStreamChunk: Your existing logic (Keep this as is)
func (a *GroqAdapter) ParseStreamChunk(line string) (string, bool) {
	if line == "data: [DONE]" {
		return "", true
	}
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}

	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(line[6:]), &chunk); err != nil {
		return "", false
	}

	if len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content, false
	}
	return "", false
}

// 3. GetPricing: Logic for telemetry/logging
func (a *GroqAdapter) GetPricing(p, r, m string) (int, int, float64) {
	promptTokens := len(p) / 4 // Rough estimation
	resTokens := len(r) / 4
	// Groq Llama 3.3 70b is extremely cheap, usually ~$0.59/1M tokens
	cost := (float64(promptTokens+resTokens) / 1000000.0) * 0.59
	return promptTokens, resTokens, cost
}
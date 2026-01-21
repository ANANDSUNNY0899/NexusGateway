package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type OpenAIAdapter struct{}

func (o *OpenAIAdapter) PrepareRequest(p, m, k string) (*http.Request, error) {
	url := "https://api.openai.com/v1/chat/completions"
	modelID := m

	// 🚀 GROQ AUTO-MAPPER: Purane models ko naye models par map karna
	if strings.Contains(strings.ToLower(m), "llama") || strings.Contains(strings.ToLower(m), "mixtral") {
		url = "https://api.groq.com/openai/v1/chat/completions"
		if strings.Contains(strings.ToLower(m), "llama") {
			modelID = "llama-3.3-70b-versatile" // Groq ka current stable model
		}
	}

	payload := StreamRequestPayload{
		Model:    modelID,
		Messages: []Message{{Role: "user", Content: p}},
		Stream:   true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+k)
	return req, nil
}

// Baki functions (ParseStreamChunk, GetPricing) same rahenge
func (o *OpenAIAdapter) ParseStreamChunk(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") { return "", false }
	cleanLine := strings.TrimPrefix(line, "data: ")
	if strings.TrimSpace(cleanLine) == "[DONE]" { return "", true } // End the stream
	var chunk struct {
		Choices []struct { Delta struct { Content string `json:"content"` } `json:"delta"` } `json:"choices"`
	}
	if err := json.Unmarshal([]byte(cleanLine), &chunk); err == nil && len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content, false
	}
	return "", false
}

func (o *OpenAIAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
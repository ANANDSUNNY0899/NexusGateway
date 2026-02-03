package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings" // 🚀 FIXED: Added missing import
)

type AnthropicAdapter struct{}

func (a *AnthropicAdapter) PrepareRequest(p, m, k, v string) (*http.Request, error) {
	url := "https://api.anthropic.com/v1/messages"
	payload := map[string]any{
		"model":      m,
		"max_tokens": 1024,
		"stream":     true,
		"messages":   []map[string]string{{"role": "user", "content": p}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", k)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *AnthropicAdapter) ParseStreamChunk(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	cleanLine := strings.TrimPrefix(line, "data: ")
	var chunk struct {
		Type  string `json:"type"`
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
	}
	if err := json.Unmarshal([]byte(cleanLine), &chunk); err == nil {
		if chunk.Type == "content_block_delta" {
			return chunk.Delta.Text, false
		}
		if chunk.Type == "message_stop" {
			return "", true
		}
	}
	return "", false
}

func (a *AnthropicAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
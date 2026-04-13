
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings" // 🚀 FIXED: Added missing import
)

type AnthropicAdapter struct{}

func (a *AnthropicAdapter) PrepareRequest(messages []Message, model, apiKey, version string) (*http.Request, error) {
    url := "https://api.anthropic.com/v1/messages"

    // Use 'model' instead of 'm' and 'messages' instead of 'p'
    payload := map[string]any{
        "model":      model,
        "messages":   messages,
        "max_tokens": 1024,
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))

    req.Header.Set("x-api-key", apiKey) // Use 'apiKey' instead of 'k'
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("content-type", "application/json")

    return req, nil
}

func (a *AnthropicAdapter) ParseStreamChunk(line string) (string, bool) {
	// Anthropic SSE lines start with "data: "
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}

	cleanLine := strings.TrimPrefix(line, "data: ")
	
	// Anthropic termination signal is often a message_stop type
	if strings.Contains(cleanLine, "message_stop") {
		return "", true
	}

	// Define the Anthropic-specific delta structure
	var chunk struct {
		Type  string `json:"type"`
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
	}

	if err := json.Unmarshal([]byte(cleanLine), &chunk); err == nil {
		// Anthropic sends text updates in 'content_block_delta' types
		if chunk.Type == "content_block_delta" {
			return chunk.Delta.Text, false
		}
	}

	return "", false
}

func (a *AnthropicAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
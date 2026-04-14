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

func (g *GeminiAdapter) PrepareRequest(messages []Message, model, key, version string) (*http.Request, error) {
	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")

	// 🔥 FIX 1: Use v1beta for advanced features like system instructions
	if version == "" {
		version = "v1beta"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)

	var geminiContents []gemini.Content
	var systemInstruction *gemini.Content

	for _, m := range messages {
		// Intercept the System Prompt
		if m.Role == "system" {
			systemInstruction = &gemini.Content{
				Parts: []gemini.Part{{Text: m.Content}},
			}
			continue
		}

		role := m.Role
		if role == "assistant" {
			role = "model" // Gemini uses "model", not "assistant"
		}

		geminiContents = append(geminiContents, gemini.Content{
			Role:  role,
			Parts: []gemini.Part{{Text: m.Content}},
		})
	}

	// 🛡️ THE SOVEREIGN WORKAROUND: Use a dynamic map to bypass strict struct definitions.
	// This ensures the build passes even if gemini.GeminiRequest lacks the SystemInstruction field.
	payload := map[string]interface{}{
		"contents": geminiContents,
	}

	if systemInstruction != nil {
		payload["system_instruction"] = systemInstruction
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
	// 1. Detect Standard End of Stream
	if strings.Contains(line, "[DONE]") {
		return "", true
	}

	// 2. Aggressive Cleaning
	line = strings.TrimSpace(line)

	// 🔥 CRITICAL SSE FIX: Because you used alt=sse, Gemini sends "data: "
	line = strings.TrimPrefix(line, "data: ")
	line = strings.TrimSpace(line)

	// Clean legacy array artifacts just in case alt=sse fails
	line = strings.TrimPrefix(line, "[")
	line = strings.TrimPrefix(line, ",")
	line = strings.TrimSuffix(line, "]")
	line = strings.TrimSpace(line)

	if line == "" {
		return "", false
	}

	// 3. Define the Extraction Schema
	var chunk struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	// 4. Parse the clean JSON
	if err := json.Unmarshal([]byte(line), &chunk); err != nil {
		return "", false
	}

	// 5. Extract the nested text safely
	if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
		return chunk.Candidates[0].Content.Parts[0].Text, false
	}

	return "", false
}

func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
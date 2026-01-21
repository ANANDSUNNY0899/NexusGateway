package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type GeminiAdapter struct{}

func (g *GeminiAdapter) PrepareRequest(prompt, model, key string) (*http.Request, error) {
	// 1. STABLE MODEL ID (Google V1 standard)
	modelID := "gemini-1.5-flash" 
	if strings.Contains(strings.ToLower(model), "pro") {
		modelID = "gemini-1.5-pro"
	}

	// 2. STABLE V1 ENDPOINT (Removing v1beta, adding alt=sse)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, key)
	
	// 3. NATIVE GOOGLE PAYLOAD (The only one V1 accepts reliably)
	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature": 0.7,
			"topP":        0.95,
			"topK":        40,
		},
	}
	body, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }
	
	req.Header.Set("Content-Type", "application/json")
	// IMPORTANT: No Authorization header set here.
	return req, nil
}

func (g *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
	cleanLine := strings.TrimSpace(line)
	if !strings.HasPrefix(cleanLine, "data: ") { return "", false }
	
	cleanLine = strings.TrimPrefix(cleanLine, "data: ")
	if cleanLine == "" { return "", false }

	// Google Native SSE Parsing logic
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct { Text string `json:"text"` } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	
	if err := json.Unmarshal([]byte(cleanLine), &resp); err == nil && len(resp.Candidates) > 0 {
		return resp.Candidates[0].Content.Parts[0].Text, false
	}
	return "", false
}

func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
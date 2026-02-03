



// package handler

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"strings"
// 	"NexusGateway/handler/gemini"
// )

// type GeminiAdapter struct{}

// func (g *GeminiAdapter) PrepareRequest(prompt, model, key, version string) (*http.Request, error) {
// 	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	
// 	// Default version to v1beta for widest compatibility with AI Studio keys
// 	if version == "" { version = "v1beta" }
	
// 	// 🚀 THE FIX: Native URL format for streaming
// 	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)
	
// 	payload := gemini.GeminiRequest{
// 		Contents: []gemini.Content{{Parts: []gemini.Part{{Text: prompt}}}},
// 	}
// 	body, _ := json.Marshal(payload)
// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
// 	if err != nil { return nil, err }
	
// 	req.Header.Set("Content-Type", "application/json")
// 	// Note: Authorization header is NOT set here for Gemini to avoid 404
// 	return req, nil
// }

// // ... (ParseStreamChunk and GetPricing remain same)

// func (g *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
// 	cleanLine := strings.TrimSpace(line)
// 	if !strings.HasPrefix(cleanLine, "data: ") { return "", false }
// 	cleanLine = strings.TrimPrefix(cleanLine, "data: ")
	
// 	var resp gemini.GeminiResponse
// 	if err := json.Unmarshal([]byte(cleanLine), &resp); err == nil && len(resp.Candidates) > 0 {
// 		if len(resp.Candidates[0].Content.Parts) > 0 {
// 			return resp.Candidates[0].Content.Parts[0].Text, false
// 		}
// 	}
// 	return "", false
// }

// func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
// 	pT, rT := EstimateTokens(p), EstimateTokens(r)
// 	return pT, rT, CalculateSavings(m, pT, rT)
// }



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
func (g *GeminiAdapter) PrepareRequest(prompt, model, key, version string) (*http.Request, error) {
	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	if version == "" { version = "v1" }
	
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)
	
	payload := gemini.GeminiRequest{
		Contents: []gemini.Content{{Parts: []gemini.Part{{Text: prompt}}}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (g *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
	cleanLine := strings.TrimSpace(line)
	if !strings.HasPrefix(cleanLine, "data: ") { return "", false }
	cleanLine = strings.TrimPrefix(cleanLine, "data: ")
	
	var resp gemini.GeminiResponse
	if err := json.Unmarshal([]byte(cleanLine), &resp); err == nil && len(resp.Candidates) > 0 {
		if len(resp.Candidates[0].Content.Parts) > 0 {
			return resp.Candidates[0].Content.Parts[0].Text, false
		}
	}
	return "", false
}

// 🚀 FIXED: Added missing GetPricing method
func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
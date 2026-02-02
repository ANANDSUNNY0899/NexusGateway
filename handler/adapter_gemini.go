// package handler

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"strings"
// )

// type GeminiAdapter struct{}

// func (g *GeminiAdapter) PrepareRequest(prompt, model, key string) (*http.Request, error) {
// 	// 🚀 THE FIX: Model Mapping to -latest version
// 	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	
// 	// Google AI Studio (AIza...) keys often REQUIRE the -latest suffix for 1.5 models
// 	if modelID == "gemini-1.5-flash" {
// 		modelID = "gemini-1.5-flash-latest"
// 	} else if modelID == "gemini-1.5-pro" {
// 		modelID = "gemini-1.5-pro-latest"
// 	}

// 	// URL: Switch back to v1beta with the -latest model anchor
// 	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, key)
	
// 	payload := map[string]any{
// 		"contents": []map[string]any{
// 			{
// 				"parts": []map[string]string{
// 					{"text": prompt},
// 				},
// 			},
// 		},
// 		"generationConfig": map[string]any{
// 			"temperature": 0.7,
// 			"topP":        0.95,
// 			"topK":        40,
// 		},
// 	}
// 	body, _ := json.Marshal(payload)
	
// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
// 	if err != nil { return nil, err }
	
// 	req.Header.Set("Content-Type", "application/json")
// 	return req, nil
// }

// func (g *GeminiAdapter) ParseStreamChunk(line string) (string, bool) {
// 	cleanLine := strings.TrimSpace(line)
// 	if !strings.HasPrefix(cleanLine, "data: ") { return "", false }
// 	cleanLine = strings.TrimPrefix(cleanLine, "data: ")
	
// 	if cleanLine == "" || cleanLine == "[DONE]" { return "", false }

// 	var resp map[string]interface{}
// 	if err := json.Unmarshal([]byte(cleanLine), &resp); err == nil {
// 		if candidates, ok := resp["candidates"].([]interface{}); ok && len(candidates) > 0 {
// 			if first, ok := candidates[0].(map[string]interface{}); ok {
// 				if content, ok := first["content"].(map[string]interface{}); ok {
// 					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
// 						if part, ok := parts[0].(map[string]interface{}); ok {
// 							if text, ok := part["text"].(string); ok {
// 								return text, false
// 							}
// 						}
// 					}
// 				}
// 			}
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

func (g *GeminiAdapter) PrepareRequest(prompt, model, key, version string) (*http.Request, error) {
	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	
	// Default version to v1beta for widest compatibility with AI Studio keys
	if version == "" { version = "v1beta" }
	
	// 🚀 THE FIX: Native URL format for streaming
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)
	
	payload := gemini.GeminiRequest{
		Contents: []gemini.Content{{Parts: []gemini.Part{{Text: prompt}}}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }
	
	req.Header.Set("Content-Type", "application/json")
	// Note: Authorization header is NOT set here for Gemini to avoid 404
	return req, nil
}

// ... (ParseStreamChunk and GetPricing remain same)

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

func (g *GeminiAdapter) GetPricing(p, r, m string) (int, int, float64) {
	pT, rT := EstimateTokens(p), EstimateTokens(r)
	return pT, rT, CalculateSavings(m, pT, rT)
}
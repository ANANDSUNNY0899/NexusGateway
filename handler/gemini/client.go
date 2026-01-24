package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// PrepareNativeRequest handles the connection to Google's STABLE V1 API
func PrepareNativeRequest(prompt, model, key string) (*http.Request, error) {
	// 🚀 THE FIX: Model Mapping for 2025/2026 Stability
	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	
	// 'gemini-pro' is legacy. We force it to the modern stable version.
	if modelID == "gemini-pro" || modelID == "gemini-1.0-pro" {
		modelID = "gemini-1.5-pro" 
	}
	if modelID == "gemini-flash" {
		modelID = "gemini-1.5-flash"
	}

	// 🚀 THE ANCHOR: Using /v1/ instead of /v1beta/
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:streamGenerateContent?alt=sse&key=%s", modelID, key)

	geminiPayload := GeminiRequest{
		Contents: []Content{
			{Parts: []Part{{Text: prompt}}},
		},
	}
	body, _ := json.Marshal(geminiPayload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }

	req.Header.Set("Content-Type", "application/json")
	// Clean Auth: No Bearer header for V1 AI Studio keys
	return req, nil
}
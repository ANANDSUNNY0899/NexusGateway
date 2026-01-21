package gemini

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// PrepareNativeRequest handles the Strict OpenAI Shim for Google AI Studio
func PrepareNativeRequest(prompt, model, key string) (*http.Request, error) {
	// 1. THE STABLE SHIM URL (Clean, no query params)
	url := "https://generativelanguage.googleapis.com/v1beta/chat/completions"

	// 2. PAYLOAD NORMALIZATION
	// Google's OpenAI shim REQUIRED prefix 'models/' to avoid 404 v1main
	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	fullModelName := "models/" + modelID
	
	payload := map[string]interface{}{
		"model": fullModelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": true,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }

	req.Header.Set("Content-Type", "application/json")
	
	// 3. THE AUTHORIZATION PROTOCOL
	// AI Studio keys work perfectly in the Bearer header IF the path is clean
	req.Header.Set("Authorization", "Bearer " + key)
	
	return req, nil
}
package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// PrepareNativeRequest generates the specific native handshake for Google
func PrepareNativeRequest(prompt, model, key, version string) (*http.Request, error) {
	// 🚀 THE FIX: Model Mapping to -latest version for 100% recognition
	modelID := strings.TrimPrefix(strings.ToLower(model), "models/")
	if strings.Contains(modelID, "1.5-flash") {
		modelID = "gemini-1.5-flash-latest"
	}

	// Dynamic Versioning (v1 or v1beta)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:streamGenerateContent?alt=sse&key=%s", version, modelID, key)

	geminiPayload := GeminiRequest{
		Contents: []Content{
			{Parts: []Part{{Text: prompt}}},
		},
	}
	body, _ := json.Marshal(geminiPayload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil { return nil, err }

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
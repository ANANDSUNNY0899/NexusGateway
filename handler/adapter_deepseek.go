// package handler

// import (
// 	"bytes"
// 	"encoding/json"
// 	"net/http"
// 	"strings"
// )

// type DeepSeekAdapter struct {
// 	OpenAIAdapter // Inherits token math automatically
// }

// func (d *DeepSeekAdapter) PrepareRequest(p, m, k, v string) (*http.Request, error) {
// 	url := "https://api.deepseek.com/chat/completions"
	
// 	// Fixed: Added curly braces for Message slice and payload initialization
// 	payload := StreamRequestPayload{
// 		Model: m, 
// 		Messages: []Message{{Role: "user", Content: p}}, 
// 		Stream: true,
// 	}
	
// 	body, err := json.Marshal(payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
// 	if err != nil {
// 		return nil, err
// 	}

// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+k)
// 	return req, nil
// }

// // Override the stream parser to capture DeepSeek's reasoning tokens
// func (d *DeepSeekAdapter) ParseStreamChunk(line string) (string, bool) {
// 	if !strings.HasPrefix(line, "data: ") {
// 		return "", false
// 	}
// 	cleanLine := strings.TrimPrefix(line, "data: ")
// 	if strings.TrimSpace(cleanLine) == "[DONE]" { // Standard SSE termination
// 		return "", true
// 	}
// 	if strings.TrimSpace(cleanLine) == "" {
// 		return "", false
// 	}

// 	// Fixed: Added space between Choices and struct, and made Choices a slice []
// 	var chunk struct {
// 		Choices []struct {
// 			Delta struct {
// 				Content          string `json:"content"`
// 				ReasoningContent string `json:"reasoning_content"`
// 			} `json:"delta"`
// 		} `json:"choices"`
// 	}

// 	// Fixed: Changed byte(cleanLine) to []byte(cleanLine)
// 	if err := json.Unmarshal([]byte(cleanLine), &chunk); err == nil && len(chunk.Choices) > 0 {
// 		delta := chunk.Choices[0].Delta // Access the first element of the slice
		
// 		// Priority 1: Reasoning Content (DeepSeek's "thinking" phase)
// 		if delta.ReasoningContent != "" {
// 			return delta.ReasoningContent, false
// 		}
// 		// Priority 2: Final Answer Content
// 		if delta.Content != "" {
// 			return delta.Content, false
// 		}
// 	}
	
// 	return "", false
// }





package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type DeepSeekAdapter struct {
	OpenAIAdapter // Inherits token math automatically
}

func (d *DeepSeekAdapter) PrepareRequest(messages []Message, m, k, v string) (*http.Request, error) {
    url := "https://api.deepseek.com/chat/completions"
    
    // Pass the entire messages slice directly
    payload := map[string]any{
        "model":    m, 
        "messages": messages, 
        "stream":   true,
    }
    
    body, err := json.Marshal(payload)
    if err != nil { return nil, err }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil { return nil, err }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+k)
    return req, nil
}

// Override the stream parser to capture DeepSeek's reasoning tokens
func (d *DeepSeekAdapter) ParseStreamChunk(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	cleanLine := strings.TrimPrefix(line, "data: ")
	if strings.TrimSpace(cleanLine) == "[DONE]" { // Standard SSE termination
		return "", true
	}
	if strings.TrimSpace(cleanLine) == "" {
		return "", false
	}

	// Fixed: Added space between Choices and struct, and made Choices a slice []
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	// Fixed: Changed byte(cleanLine) to []byte(cleanLine)
	if err := json.Unmarshal([]byte(cleanLine), &chunk); err == nil && len(chunk.Choices) > 0 {
		delta := chunk.Choices[0].Delta // Access the first element of the slice
		
		// Priority 1: Reasoning Content (DeepSeek's "thinking" phase)
		if delta.ReasoningContent != "" {
			return delta.ReasoningContent, false
		}
		// Priority 2: Final Answer Content
		if delta.Content != "" {
			return delta.Content, false
		}
	}
	
	return "", false
}
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	
)

// THE CONTRACT
type AIProvider interface {
	Send(prompt string) (string, error)
}

// THE FACTORY (Smart Router)
func GetProvider(modelName string, cfgKeys map[string]string) (AIProvider, error) {
	switch {
	// Anthropic
	case strings.Contains(modelName, "claude"):
		return &AnthropicProvider{APIKey: cfgKeys["anthropic"], Model: modelName}, nil
	
	// Groq (Llama 3)
	case strings.Contains(modelName, "llama"):
		return &GroqProvider{APIKey: cfgKeys["groq"], Model: modelName}, nil
	
	// Gemini
	case strings.Contains(modelName, "gemini"):
		return &GeminiProvider{APIKey: cfgKeys["gemini"], Model: modelName}, nil

	// Default: OpenAI
	default:
		return &OpenAIProvider{APIKey: cfgKeys["openai"], Model: modelName}, nil
	}
}

// ---------------------------
// GROQ IMPLEMENTATION
// ---------------------------
type GroqProvider struct {
	APIKey string
	Model  string
}

func (p *GroqProvider) Send(prompt string) (string, error) {
	payload := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	return makeRequest(req)
}

// ---------------------------
// GEMINI IMPLEMENTATION
// ---------------------------
type GeminiProvider struct {
	APIKey string
	Model  string
}

func (p *GeminiProvider) Send(prompt string) (string, error) {
	// Gemini uses a different URL structure
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.Model, p.APIKey)
	
	payload := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}
	jsonBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Gemini response parsing is unique
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini Error: %s", string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct { Text string `json:"text"` } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("no response from Gemini")
}

// ---------------------------
// OPENAI & ANTHROPIC (Simplified Helper)
// ---------------------------
type OpenAIProvider struct {
	APIKey string
	Model  string
}

func (p *OpenAIProvider) Send(prompt string) (string, error) {
	payload := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}
	jsonBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return makeRequest(req)
}

type AnthropicProvider struct {
	APIKey string
	Model  string
}

func (p *AnthropicProvider) Send(prompt string) (string, error) {
	payload := map[string]any{
		"model": p.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}
	jsonBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	// Custom parsing for Anthropic
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic Error: %s", string(body))
	}
	var result struct { Content []struct { Text string `json:"text"` } `json:"content"` }
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Content) > 0 { return result.Content[0].Text, nil }
	return "", fmt.Errorf("no response")
}

// Helper for OpenAI/Groq (Same JSON structure)
func makeRequest(req *http.Request) (string, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API Error: %s", string(body))
	}

	var result struct {
		Choices []struct {
			Message struct { Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Choices) > 0 { return result.Choices[0].Message.Content, nil }
	return "", fmt.Errorf("no content")
}
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// 1. THE CONTRACT
type AIProvider interface {
	Send(prompt string) (string, error)
}

// 2. THE FACTORY
func GetProvider(modelName string, openAIKey string, anthropicKey string) (AIProvider, error) {
	switch modelName {
	case "claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307":
		return &AnthropicProvider{APIKey: anthropicKey, Model: modelName}, nil
	case "gpt-3.5-turbo", "gpt-4", "gpt-4o":
		return &OpenAIProvider{APIKey: openAIKey, Model: modelName}, nil
	default:
		// Default fallback
		return &OpenAIProvider{APIKey: openAIKey, Model: "gpt-3.5-turbo"}, nil
	}
}

// ---------------------------
// 3. OPENAI IMPLEMENTATION
// ---------------------------
type OpenAIProvider struct {
	APIKey string
	Model  string
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func (p *OpenAIProvider) Send(prompt string) (string, error) {
	payload := OpenAIRequest{
		Model: p.Model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API Error: %s", string(body))
	}

	var result OpenAIResponse
	json.NewDecoder(resp.Body).Decode(&result)
	
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response from OpenAI")
}

// ---------------------------
// 4. ANTHROPIC IMPLEMENTATION
// ---------------------------
type AnthropicProvider struct {
	APIKey string
	Model  string
}

// Anthropic has a slightly different JSON structure
type AnthropicRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (p *AnthropicProvider) Send(prompt string) (string, error) {
	payload := AnthropicRequest{
		Model: p.Model,
		MaxTokens: 1024,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01") // Required header
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API Error: %s", string(body))
	}

	var result AnthropicResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", fmt.Errorf("no response from Anthropic")
}
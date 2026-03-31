package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// W5H2Intent defines the structured intent format
type W5H2Intent struct {
	Who   string `json:"who"`
	What  string `json:"what"`
	When  string `json:"when"`
	Where string `json:"where"`
	Why   string `json:"why"`
	How   string `json:"how"`
}

// GroqResponse defines the internal structure to parse Groq API response safely
type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateIntentSignature uses Groq to quickly classify the prompt's intent into JSON
func GenerateIntentSignature(prompt string, groqKey string) string {
	// If no Groq key is configured, fallback to standard hashing
	if groqKey == "" {
		return GenerateHash(prompt)
	}

	url := "https://api.groq.com/openai/v1/chat/completions"

	// Strict system prompt to enforce the W5H2 JSON extraction
	systemPrompt := `You are a lightning-fast intent classifier. Extract the core intent of the user's prompt into a strict JSON object with exactly these keys: "who", "what", "when", "where", "why", "how". If a field is not applicable, use "none". Output ONLY valid JSON.`

	payload := map[string]any{
		"model": "llama3-8b-8192", // Low-latency model for extraction
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.1,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+groqKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return GenerateHash(prompt) // Fallback if network fails
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// --- FIXED: Use a struct instead of map[string]any to avoid 'len' errors ---
	var result GroqResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return GenerateHash(prompt)
	}

	var intentStr string
	if len(result.Choices) > 0 {
		// Strip whitespace and normalize string for hashing
		intentStr = strings.TrimSpace(result.Choices[0].Message.Content)
	}

	if intentStr == "" {
		return GenerateHash(prompt)
	}

	// --- FIXED: Use []byte(intentStr) for conversion ---
	hash := sha256.Sum256([]byte(intentStr))
	return hex.EncodeToString(hash[:])
}
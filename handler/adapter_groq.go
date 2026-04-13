package adapters

import (
	"encoding/json"
	"strings"
)

type GroqAdapter struct{}

func (a *GroqAdapter) ParseStreamChunk(line string) (string, bool) {
	if line == "data: [DONE]" {
		return "", true
	}
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}

	// Groq sends standard OpenAI-style delta chunks
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	// Remove "data: " prefix and parse
	if err := json.Unmarshal([]byte(line[6:]), &chunk); err != nil {
		return "", false
	}

	if len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content, false
	}

	return "", false
}

// Ensure your Router knows how to build the request for Groq too
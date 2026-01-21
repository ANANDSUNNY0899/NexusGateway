package handler

import "strings"

func GetProvider(model string) AIProvider {
	if strings.Contains(strings.ToLower(model), "gemini") {
		return &GeminiAdapter{}
	}
	return &OpenAIAdapter{}
}
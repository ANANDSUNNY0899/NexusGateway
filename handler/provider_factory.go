// package handler

// import "strings"

// func GetProvider(model string) AIProvider {
// 	if strings.Contains(strings.ToLower(model), "gemini") {
// 		return &GeminiAdapter{}
// 	}
// 	return &OpenAIAdapter{}
// }




package handler

import "strings"

func GetProvider(model string) AIProvider {
	modelLower := strings.ToLower(model)

	if strings.Contains(modelLower, "gemini") {
		return &GeminiAdapter{}
	}
	if strings.Contains(modelLower, "claude") {
		return &AnthropicAdapter{}
	}
	
	return &OpenAIAdapter{}
}
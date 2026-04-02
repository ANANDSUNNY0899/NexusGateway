

// package handler

// import "strings"

// func GetProvider(model string) AIProvider {
//     m := strings.ToLower(model)

//     // 🚀 ALIASING: The "Intelligent" Choice
//     if m == "nexus-smart" || m == "nexus-auto" {
//         return &DeepSeekAdapter{} // Highest intelligence per dollar
//     }
//     if m == "nexus-fast" {
//         return &OpenAIAdapter{} // Usually routes to Groq/Llama
//     }

//     switch {
//     case strings.Contains(m, "gemini"):
//         return &GeminiAdapter{}
//     case strings.Contains(m, "deepseek"):
//         return &DeepSeekAdapter{}
//     case strings.Contains(m, "claude") || strings.Contains(m, "anthropic"):
//         return &AnthropicAdapter{}
//     case strings.Contains(m, "gpt") || strings.Contains(m, "openai"):
//         return &OpenAIAdapter{}
//     default:
//         // Default to OpenAI adapter for generic Llama/Mistral models
//         return &OpenAIAdapter{}
//     }
// }



package handler

import (
	"net/http"
	"strings"
)

// --- 🏛️ THE SOVEREIGN CONTRACT ---
// This interface MUST match the function signatures in your adapters exactly.
type AIProvider interface {
	PrepareRequest(messages []Message, model, key, version string) (*http.Request, error)
	ParseStreamChunk(line string) (string, bool)
	GetPricing(p, r, m string) (int, int, float64)
}

func GetProvider(model string) AIProvider {
	m := strings.ToLower(model)

	// 🚀 ALIASING: The "Intelligent" Choice
	// These routes prioritize DeepSeek-R1 for complex reasoning
	if m == "nexus-smart" || m == "nexus-auto" {
		return &DeepSeekAdapter{} 
	}
	
	// Fast routes go to Groq/Llama via the OpenAI-compatible adapter
	if m == "nexus-fast" {
		return &OpenAIAdapter{} 
	}

	switch {
	case strings.Contains(m, "gemini"):
		return &GeminiAdapter{}
	case strings.Contains(m, "deepseek"):
		return &DeepSeekAdapter{}
	case strings.Contains(m, "claude") || strings.Contains(m, "anthropic"):
		return &AnthropicAdapter{}
	case strings.Contains(m, "gpt") || strings.Contains(m, "openai"):
		return &OpenAIAdapter{}
	default:
		// Default for Llama/Mistral models (Groq-compatible)
		return &OpenAIAdapter{}
	}
}
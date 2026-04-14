

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
    if m == "nexus-smart" || m == "nexus-auto" {
        return &DeepSeekAdapter{} 
    }
    
    // Fast routes go to Groq/Llama
    if m == "nexus-fast" {
        return &GroqAdapter{} 
    }

    switch {
    // 🚀 Add 'google' to the Gemini adapter router
    case strings.Contains(m, "gemini") || strings.Contains(m, "google"):
        return &GeminiAdapter{}
        
    // 🚨 THE FIX: Catch Groq's R1 Distill BEFORE the official DeepSeek adapter does
    case strings.Contains(m, "deepseek-r1-distill"):
        return &GroqAdapter{}

    // Now it is safe to route official DeepSeek models
    case strings.Contains(m, "deepseek"):
        return &DeepSeekAdapter{}
        
    case strings.Contains(m, "claude") || strings.Contains(m, "anthropic"):
        return &AnthropicAdapter{}
        
    case strings.Contains(m, "gpt") || strings.Contains(m, "openai"):
        return &OpenAIAdapter{}
        
    case strings.Contains(m, "llama") || strings.Contains(m, "mixtral"):
        return &GroqAdapter{} 
        
    default:
        // Defaulting to GroqAdapter for high-speed fallback
        return &GroqAdapter{}
    }
}
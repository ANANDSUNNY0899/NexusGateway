

// package handler

// import "strings"

// func GetProvider(model string) AIProvider {
//     modelLower := strings.ToLower(model)

//     if strings.Contains(modelLower, "gemini") {
//         return &GeminiAdapter{}
//     }
//     if strings.Contains(modelLower, "claude") {
//         return &AnthropicAdapter{}
//     }
//     if strings.Contains(modelLower, "deepseek") {
//         return &DeepSeekAdapter{}
//     }
//     if strings.Contains(modelLower, "mistral") {
//         return &MistralAdapter{}
//     }
    
//     return &OpenAIAdapter{}
// }



package handler

import "strings"

func GetProvider(model string) AIProvider {
    m := strings.ToLower(model)

    // 🚀 ALIASING: The "Intelligent" Choice
    if m == "nexus-smart" || m == "nexus-auto" {
        return &DeepSeekAdapter{} // Highest intelligence per dollar
    }
    if m == "nexus-fast" {
        return &OpenAIAdapter{} // Usually routes to Groq/Llama
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
        // Default to OpenAI adapter for generic Llama/Mistral models
        return &OpenAIAdapter{}
    }
}
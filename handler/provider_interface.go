



// package handler

// import "net/http"

// type AIProvider interface {
// 	// PrepareRequest ab 4 arguments lega (prompt, model, key, version)
// 	PrepareRequest(prompt string, model string, key string, version string) (*http.Request, error)
// 	ParseStreamChunk(line string) (string, bool)
// 	GetPricing(prompt string, response string, model string) (int, int, float64)
// }


package handler

import "net/http"

// AIProvider is the sovereign contract for all LLMs
type AIProvider interface {
    // PrepareRequest handles URL construction, Auth, and Payload Morphing
    PrepareRequest(prompt string, model string, key string, version string) (*http.Request, error)
    
    // ParseStreamChunk handles the different "dialects" of SSE (Server-Sent Events)
    ParseStreamChunk(line string) (string, bool)
    
    // GetPricing calculates the "Sovereign Savings" based on tokens and model tier
    GetPricing(prompt string, response string, model string) (int, int, float64)
}
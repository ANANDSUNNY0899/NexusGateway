

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

type AIProvider interface {
	// PrepareRequest ab 4 arguments lega (prompt, model, key, version)
	PrepareRequest(prompt string, model string, key string, version string) (*http.Request, error)
	ParseStreamChunk(line string) (string, bool)
	GetPricing(prompt string, response string, model string) (int, int, float64)
}
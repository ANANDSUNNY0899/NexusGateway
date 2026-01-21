package handler

import "net/http"

type AIProvider interface {
	PrepareRequest(prompt string, model string, key string) (*http.Request, error)
	ParseStreamChunk(line string) (string, bool)
	GetPricing(prompt string, response string, model string) (int, int, float64)
}
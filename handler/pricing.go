package handler

import "strings"

type ModelPricing struct {
	InputPrice  float64
	OutputPrice float64
}

var PricingMap = map[string]ModelPricing{
	"gpt-4o":                   {InputPrice: 5.00, OutputPrice: 15.00},
	"gpt-3.5-turbo":            {InputPrice: 0.50, OutputPrice: 1.50},
	"llama-3.3-70b-versatile":  {InputPrice: 0.59, OutputPrice: 0.79}, // Groq New
	"llama-3.1-8b-instant":     {InputPrice: 0.05, OutputPrice: 0.08}, // Groq New
	"gemini-1.5-flash-latest":  {InputPrice: 0.35, OutputPrice: 1.05},
}

func EstimateTokens(text string) int {
	return len(text) / 4
}

func CalculateSavings(model string, pTokens, cTokens int) float64 {
	price, exists := PricingMap[strings.ToLower(model)]
	if !exists { price = PricingMap["gpt-3.5-turbo"] }
	return (float64(pTokens)/1000000.0)*price.InputPrice + (float64(cTokens)/1000000.0)*price.OutputPrice
}
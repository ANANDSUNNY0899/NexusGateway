package handler

import (
	"strings"
)

// DetermineModelTier analyzes the prompt to save the developer money.
// This is the "Billion Dollar" feature: Autonomous Cost Optimization.
func DetermineModelTier(prompt string) string {
	prompt = strings.ToLower(prompt)
	
	// Keywords that require the "Big Brain" (Reasoning/Logic)
	highLogicKeywords := []string{"analyze", "solve", "complex", "debug", "architecture", "reasoning", "r1", "math"}
	
	for _, word := range highLogicKeywords {
		if strings.Contains(prompt, word) {
			// If it's hard, use the high-tier reasoning model
			return "deepseek-reasoner" 
		}
	}

	// Default to the "Fast Brain" (Low Cost/High Speed)
	// This handles 80% of daily tasks for 1/10th the price.
	return "llama-3.3-70b-versatile" 
}



// ConstitutionResult defines the response from our governance layer
type ConstitutionResult struct {
	Allowed      bool
	ModifiedText string
	RuleID       string
	RefusalMsg   string
	Disclaimer   string
}
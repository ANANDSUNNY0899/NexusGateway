package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// DiscoverWorkingModel scans Google's infrastructure for Chat-capable models ONLY.
func DiscoverWorkingModel(key string) string {
	log.Printf("🔍 [ADAPTIVE] Key: %s... - Scanning for active generation engines", key[:6])
	
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", key)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return "gemini-1.5-flash" 
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name                      string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	bestFallback := ""
	for _, m := range result.Models {
		canGenerate := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				canGenerate = true
				break
			}
		}

		if canGenerate {
			name := strings.TrimPrefix(m.Name, "models/")
			if name == "gemini-1.5-flash" {
				log.Printf("🎯 [ADAPTIVE] Verified Stable Flash: %s", name)
				return name
			}
			if strings.Contains(name, "1.5-pro") {
				bestFallback = name
			}
			if bestFallback == "" && strings.Contains(name, "gemini") {
				bestFallback = name
			}
		}
	}

	if bestFallback != "" {
		log.Printf("🎯 [ADAPTIVE] Found Alternative Engine: %s", bestFallback)
		return bestFallback
	}

	return "gemini-1.5-flash"
}
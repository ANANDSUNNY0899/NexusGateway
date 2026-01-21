package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// GenerateHash converts a string into a SHA-256 hex string.
func GenerateHash(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil)) // FIXED: Changed ToString to EncodeToString
}

// RedactPII removes sensitive info like emails from the prompt.
func RedactPII(text string) string {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	return emailRegex.ReplaceAllString(text, "[REDACTED_EMAIL]")
}

// GenerateAPIKey creates a secure random key like nk-a1b2c3...
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("nk-%s", hex.EncodeToString(bytes)), nil
}
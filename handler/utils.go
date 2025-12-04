package handler

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateAPIKey creates a random secure string like "nk-a1b2c3..."
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "nk-" + hex.EncodeToString(bytes), nil
}
package handler

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// We removed 'var ctx' and 'RedisClient' from here because they are handled by redis.go

func GenerateKey(prompt string, model string) string {
	hash := sha256.Sum256([]byte(prompt + model))
	return fmt.Sprintf("nexus:cache:%x", hash)
}

func GetCachedResponse(prompt string, model string) (string, bool) {
	rdb := GetClient() // Use your existing helper to get the client
	if rdb == nil {
		return "", false
	}
	key := GenerateKey(prompt, model)
	val, err := rdb.Get(ctx, key).Result() // ctx comes from redis.go
	if err != nil {
		return "", false
	}
	return val, true
}

func SetCache(prompt string, model string, response string) {
	rdb := GetClient() // Use your existing helper
	if rdb == nil {
		return
	}
	key := GenerateKey(prompt, model)
	rdb.Set(ctx, key, response, 24*time.Hour)
}
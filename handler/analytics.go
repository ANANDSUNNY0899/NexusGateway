package handler

import (
	"context"
	"log"
)

// LogRequest saves the event to Supabase in the background
func LogRequest(apiKey string, model string, status int, isCacheHit bool) {
	if db == nil {
		return
	}

	// 🚀 GO ROUTINE (The Magic)
	// The keyword "go" tells the server: 
	// "Do this in a separate thread. Don't make the user wait."
	go func() {
		query := `
			INSERT INTO request_logs (api_key, model, status, is_cache_hit)
			VALUES ($1, $2, $3, $4)
		`
		_, err := db.Exec(context.Background(), query, apiKey, model, status, isCacheHit)
		
		if err != nil {
			log.Printf("⚠️ Analytics Error: %v", err)
		}
	}()
}
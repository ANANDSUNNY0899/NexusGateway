package handler

import (
	"context"
	"log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func InitializeDB(connString string) {
	config, _ := pgxpool.ParseConfig(connString)
	config.MaxConns = 25
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	db, _ = pgxpool.NewWithConfig(context.Background(), config)
	log.Println("✅ DB Connected")
}

// Inside handler/db.go

func LogRequest(apiKey string, model string, status int, isHit bool, pTokens int, cTokens int, savings float64, latency int, prompt string, response string) {
	if db == nil { return }

	query := `
		INSERT INTO request_logs 
		(api_key, model, status, is_cache_hit, prompt_tokens, completion_tokens, cost_saved, provider_latency_ms, prompt_text, response_text) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	go func() {
		_, err := db.Exec(context.Background(), query, 
			apiKey, model, status, isHit, pTokens, cTokens, savings, latency, prompt, response,
		)
		if err != nil {
			log.Printf("🚨 Telemetry Logging Failed: %v", err)
		}
	}()
}

func ValidateAPIKey(apiKey string) bool {
	var exists bool
	db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE api_key=$1)", apiKey).Scan(&exists)
	return exists
}

func CheckUserLimit(apiKey string) (bool, error) {
	var used, limit int
	err := db.QueryRow(context.Background(), "SELECT requests_used, request_limit FROM users WHERE api_key=$1", apiKey).Scan(&used, &limit)
	return used < limit, err
}

func IncrementUsage(apiKey string) {
	go db.Exec(context.Background(), "UPDATE users SET requests_used = requests_used + 1 WHERE api_key=$1", apiKey)
}
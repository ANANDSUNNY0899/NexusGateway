package handler

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

var db *pgx.Conn

func InitializeDB(connString string) {
	// 1. Parse Config
	config, err := pgx.ParseConfig(connString)
	if err != nil {
		log.Fatalf("❌ Invalid DB URL: %v", err)
	}

	// 2. FORCE Simple Protocol (Crucial for Supabase Pooler)
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// 3. Connect
	db, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v", err)
	}
	log.Println("✅ Connected to Supabase (Simple Mode)")
}

func ValidateAPIKey(apiKey string) bool {
	if db == nil { return false }
	var exists bool
	err := db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE api_key=$1)", apiKey).Scan(&exists)
	if err != nil {
		log.Printf("DB Error: %v", err)
		return false
	}
	return exists
}

// <--- NEW FUNCTIONS START HERE --->

// CheckUserLimit returns true if usage < limit
func CheckUserLimit(apiKey string) (bool, error) {
	if db == nil { return false, nil }

	var used int
	var limit int

	// Get current usage and limit
	query := `SELECT requests_used, request_limit FROM users WHERE api_key=$1`
	err := db.QueryRow(context.Background(), query, apiKey).Scan(&used, &limit)
	if err != nil {
		return false, err
	}

	// If used >= limit, BLOCK THEM
	if used >= limit {
		return false, nil
	}

	return true, nil
}

// IncrementUsage adds +1 to the user's meter
func IncrementUsage(apiKey string) {
	if db == nil { return }

	// Run in background (goroutine) so we don't slow down the user
	go func() {
		_, err := db.Exec(context.Background(), "UPDATE users SET requests_used = requests_used + 1 WHERE api_key=$1", apiKey)
		if err != nil {
			log.Printf("Failed to update usage: %v", err)
		}
	}()
}
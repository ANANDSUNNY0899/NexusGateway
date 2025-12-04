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

	// 2. DISABLE ALL CACHING (The Fix)
	config.StatementCacheCapacity = 0
	config.DescriptionCacheCapacity = 0
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// 3. Connect
	db, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v", err)
	}
	log.Println("✅ Connected to Supabase (Simple Mode)")
}

func ValidateAPIKey(apiKey string) bool {
	if db == nil {
		return false
	}
	var exists bool
	// Use QueryRow which works better with Simple Protocol
	err := db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE api_key=$1)", apiKey).Scan(&exists)
	if err != nil {
		log.Printf("DB Error: %v", err)
		return false
	}
	return exists
}
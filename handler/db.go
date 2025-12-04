package handler

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

var db *pgx.Conn

// InitializeDB connects to Supabase
func InitializeDB(connString string) {
	// 1. Parse the URL into a Config object
	config, err := pgx.ParseConfig(connString)
	if err != nil {
		log.Fatalf("❌ Invalid DB URL: %v", err)
	}

	// 2. FORCE Simple Protocol (This fixes the 401/Prepared Statement error)
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// 3. Connect using this specific config
	db, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v", err)
	}
	log.Println("✅ Connected to Supabase (Postgres)")
}

// ValidateAPIKey checks if the key exists in the 'users' table
func ValidateAPIKey(apiKey string) bool {
	if db == nil {
		return false // Fail safe if DB is down
	}

	var exists bool
	// SQL Query: Is there a user with this api_key?
	err := db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE api_key=$1)", apiKey).Scan(&exists)
	
	if err != nil {
		log.Printf("DB Error: %v", err)
		return false
	}
	return exists
}
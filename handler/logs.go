package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type LogEntry struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Status    int       `json:"status"`
	IsCache   bool      `json:"is_cache_hit"`
	CreatedAt time.Time `json:"created_at"`
}

func HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	// 1. Auth Check (Only the user can see their logs)
	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if db == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	// 2. Query Postgres (Last 50 logs for this key)
	query := `
		SELECT id, model, status, is_cache_hit, created_at 
		FROM request_logs 
		WHERE api_key = $1 
		ORDER BY created_at DESC 
		LIMIT 50
	`
	
	rows, err := db.Query(context.Background(), query, apiKey)
	if err != nil {
		log.Printf("Log Query Error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []LogEntry{}
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.Model, &l.Status, &l.IsCache, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	// 3. Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
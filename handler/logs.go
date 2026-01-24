package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// LogEntry matches the 12-parameter ledger plus metadata
type LogEntry struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Status    int       `json:"status"`
	IsCache   bool      `json:"is_cache_hit"`
	Savings   float64   `json:"cost_saved"`
	Latency   int       `json:"provider_latency_ms"`
	Prompt    string    `json:"prompt_text"`
	Response  string    `json:"response_text"`
	RuleID    string    `json:"triggered_rule"`
	Action    string    `json:"governance_action"`
	CreatedAt time.Time `json:"created_at"`
}

func HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	// 1. IDENTITY PROTECTION
	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

	if apiKey == "" {
		respondWithError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if db == nil {
		respondWithError(w, "DB Offline", http.StatusServiceUnavailable)
		return
	}

	// 2. THE UNBREAKABLE QUERY (Using COALESCE to prevent NULL crashes)
	// Order must match rows.Scan exactly!
	query := `
		SELECT 
			id, 
			model, 
			status, 
			is_cache_hit, 
			COALESCE(cost_saved, 0)::FLOAT, 
			COALESCE(provider_latency_ms, 0), 
			COALESCE(prompt_text, 'Pre-Sovereign Data'), 
			COALESCE(response_text, 'No payload captured'), 
			COALESCE(triggered_rule, 'NONE'), 
			COALESCE(governance_action, 'PERMITTED'), 
			created_at 
		FROM request_logs 
		WHERE api_key = $1 
		ORDER BY created_at DESC 
		LIMIT 50
	`
	
	rows, err := db.Query(context.Background(), query, apiKey)
	if err != nil {
		log.Printf("🚨 LOG QUERY FAILED: %v", err)
		respondWithError(w, "Query Failure", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []LogEntry{}
	for rows.Next() {
		var l LogEntry
		// 🚀 THE SYNC: Exactly 11 variables matching the 11 columns above
		err := rows.Scan(
			&l.ID, 
			&l.Model, 
			&l.Status, 
			&l.IsCache, 
			&l.Savings, 
			&l.Latency, 
			&l.Prompt, 
			&l.Response, 
			&l.RuleID, 
			&l.Action, 
			&l.CreatedAt,
		)
		if err != nil {
			log.Printf("⚠️  Row Scan Error (Skipping): %v", err)
			continue
		}
		logs = append(logs, l)
	}

	// 3. RETURN DATA
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
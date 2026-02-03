package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// LogEntry represents the 12-parameter technical trace
type LogEntry struct {
	ID               string    `json:"id"`
	Model            string    `json:"model"`
	Status           int       `json:"status"`
	IsCacheHit       bool      `json:"is_cache_hit"`
	Savings          float64   `json:"cost_saved"`
	Latency          int       `json:"provider_latency_ms"`
	PromptText       string    `json:"prompt_text"`
	ResponseText     string    `json:"response_text"`
	TriggeredRule    string    `json:"triggered_rule"`
	GovernanceAction string    `json:"governance_action"`
	CreatedAt        time.Time `json:"created_at"`
}

func HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	// 1. Identity Verification
	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

	if apiKey == "" {
		respondWithError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if db == nil {
		respondWithError(w, "Infrastructure Offline", http.StatusServiceUnavailable)
		return
	}

	// 2. THE SYNCED QUERY (Fetching all 11 core columns)
	// We use COALESCE to prevent NULL pointer errors for old data
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
		log.Printf("🚨 X-RAY FETCH ERROR: %v", err)
		respondWithError(w, "Telemetry retrieval failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		// 🚀 THE SYNC: Exactly matching the SELECT order
		err := rows.Scan(
			&l.ID, 
			&l.Model, 
			&l.Status, 
			&l.IsCacheHit, 
			&l.Savings, 
			&l.Latency, 
			&l.PromptText, 
			&l.ResponseText, 
			&l.TriggeredRule, 
			&l.GovernanceAction, 
			&l.CreatedAt,
		)
		if err != nil {
			log.Printf("⚠️  Row Scan Mismatch: %v", err)
			continue
		}
		logs = append(logs, l)
	}

	// 3. RETURN DATA
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
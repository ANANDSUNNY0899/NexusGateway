package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// LogEntry represents the telemetry data for the Trace Inspector
type LogEntry struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Status    int       `json:"status"`
	IsCache   bool      `json:"is_cache_hit"`
	Savings   float64   `json:"cost_saved"`
	Latency   int       `json:"provider_latency_ms"`
	Prompt    string    `json:"prompt_text"` 
	Response  string    `json:"response_text"` 
	CreatedAt time.Time `json:"created_at"`
}

func HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	// 1. IDENTITY CAPTURE
	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

	if apiKey == "" {
		respondWithError(w, "Unauthorized: API Key missing", http.StatusUnauthorized)
		return
	}

	if db == nil {
		respondWithError(w, "Infrastructure Error: DB not connected", http.StatusServiceUnavailable)
		return
	}

	// 2. TELEMETRY QUERY (Last 50 traces)
	// Humne isme cost_saved aur provider_latency_ms add kiya hai
	query := `
		SELECT id, model, status, is_cache_hit, cost_saved, provider_latency_ms, created_at 
		FROM request_logs 
		WHERE api_key = $1 
		ORDER BY created_at DESC 
		LIMIT 50
	`
	
	rows, err := db.Query(context.Background(), query, apiKey)
	if err != nil {
		log.Printf("🚨 Trace Query Error: %v", err)
		respondWithError(w, "Analytics Retrieval Failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []LogEntry{}
	for rows.Next() {
		var l LogEntry
		err := rows.Scan(
			&l.ID, 
			&l.Model, 
			&l.Status, 
			&l.IsCache, 
			&l.Savings, 
			&l.Latency, 
			&l.CreatedAt,
		)
		if err != nil {
			log.Printf("⚠️  Row Scan Error: %v", err)
			continue
		}
		logs = append(logs, l)
	}

	// 3. RETURN ENCRYPTED TELEMETRY JSON
	w.Header().Set("Content-Type", "application/json")
	// Cache control taaki dashboard fast load ho but logs real-time rahein
	w.Header().Set("Cache-Control", "private, max-age=5") 
	json.NewEncoder(w).Encode(logs)
}
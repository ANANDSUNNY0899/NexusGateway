package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// GraphPoint represents a single data point on the UI chart
type GraphPoint struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

func HandleStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var total, hits, tokens int64
	var savings, latency float64
	graphData := []GraphPoint{}

	// 1. FAST PATH: Get Real-time Counters from Redis
	if redisClient != nil {
		redisClient.Get(ctx, "stats:total_requests").Scan(&total)
		redisClient.Get(ctx, "stats:cache_hits").Scan(&hits)
	}

	// 2. ANALYTICS PATH: Pull Ledger Data from Supabase
	if db != nil {
		// A. Aggregate Totals (Ignoring old 0-telemetry rows for accuracy)
		aggQuery := `
			SELECT 
				COALESCE(SUM(cost_saved), 0)::FLOAT, 
				COALESCE(AVG(provider_latency_ms) FILTER (WHERE provider_latency_ms > 0), 0)::FLOAT,
				COALESCE(SUM(prompt_tokens + completion_tokens), 0)::BIGINT
			FROM request_logs
		`
		err := db.QueryRow(ctx, aggQuery).Scan(&savings, &latency, &tokens)
		if err != nil {
			log.Printf("🚨 Aggregation Fail: %v", err)
		}

		// B. Graph Data: Hourly traffic for the last 24 hours
		graphQuery := `
			SELECT to_char(created_at, 'HH24:00') as label, COUNT(*) as value
			FROM request_logs
			WHERE created_at > NOW() - INTERVAL '24 hours'
			GROUP BY label
			ORDER BY label ASC;
		`
		rows, err := db.Query(ctx, graphQuery)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p GraphPoint
				if err := rows.Scan(&p.Time, &p.Count); err == nil {
					graphData = append(graphData, p)
				}
			}
		}
	}

	// 3. ASSEMBLE ENTERPRISE PAYLOAD
	// Hum ensure karenge ki khali array ke bajaye empty slice bhejien taaki frontend crash na ho
	if len(graphData) == 0 {
		graphData = []GraphPoint{} 
	}

	response := map[string]any{
		"total_requests": total,
		"cache_hits":     hits,
		"total_savings":  savings,
		"avg_latency":    int(latency),
		"total_tokens":   tokens,
		"graph_data":     graphData, // 🚀 THE FIX: Graph is now sent to UI
	}

	// 4. RETURN ENCRYPTED TELEMETRY JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=10") // Reduce DB load
	json.NewEncoder(w).Encode(response)
}

// To handle individual user usage
func HandleUserUsage(w http.ResponseWriter, r *http.Request) {
    token := getAPIKey(r)
    var used, limit int
    
    // Fetch from Supabase
    db.QueryRow(r.Context(), "SELECT requests_used, request_limit FROM users WHERE api_key=$1", token).Scan(&used, &limit)
    
    json.NewEncoder(w).Encode(map[string]any{
        "used": used,
        "limit": limit,
    })
}
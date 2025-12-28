package handler

import (
	"context"
	"encoding/json"
	"net/http"
	
)

type StatsResponse struct {
	TotalRequests int64       `json:"total_requests"`
	CacheHits     int64       `json:"cache_hits"`
	GraphData     []GraphPoint `json:"graph_data"` // <--- NEW
}

type GraphPoint struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

func HandleStats(w http.ResponseWriter, r *http.Request) {
	// 1. Get Counters from Redis (Fast)
	client := GetClient()
	var total, hits int64
	if client != nil {
		total, _ = client.Get(ctx, "stats:total_requests").Int64()
		hits, _ = client.Get(ctx, "stats:cache_hits").Int64()
	}

	// 2. Get Graph Data from Postgres (Slow but detailed)
	// We aggregate requests by hour for the last 24 hours
	graphData := []GraphPoint{}
	
	if db != nil {
		query := `
			SELECT to_char(created_at, 'HH24:00') as time, COUNT(*) as count
			FROM request_logs
			WHERE created_at > NOW() - INTERVAL '24 hours'
			GROUP BY time
			ORDER BY time ASC;
		`
		rows, err := db.Query(context.Background(), query)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p GraphPoint
				rows.Scan(&p.Time, &p.Count)
				graphData = append(graphData, p)
			}
		}
	}

	// 3. Return JSON
	resp := StatsResponse{
		TotalRequests: total,
		CacheHits:     hits,
		GraphData:     graphData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
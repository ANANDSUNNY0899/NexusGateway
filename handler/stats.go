package handler

import (
	"context"
	"encoding/json"
	"net/http"
)

func HandleStats(w http.ResponseWriter, r *http.Request) {
	var total, hits, tokens int64
	var savings, latency float64

	if redisClient != nil {
		redisClient.Get(context.Background(), "stats:total_requests").Scan(&total)
		redisClient.Get(context.Background(), "stats:cache_hits").Scan(&hits)
	}

	if db != nil {
		query := `SELECT COALESCE(SUM(cost_saved), 0), COALESCE(AVG(provider_latency_ms), 0), COALESCE(SUM(prompt_tokens + completion_tokens), 0) FROM request_logs`
		db.QueryRow(context.Background(), query).Scan(&savings, &latency, &tokens)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"total_requests": total,
		"cache_hits":     hits,
		"total_savings":  savings,
		"avg_latency":    int(latency),
		"total_tokens":   tokens,
	})
}
package handler

import (
	"encoding/json"
	"net/http"
)

type StatsResponse struct {
	TotalRequests string `json:"total_requests"`
	CacheHits     string `json:"cache_hits"`
	CacheMisses   string `json:"cache_misses"`
	MoneySaved    string `json:"money_saved_est"` // Estimated
}

func HandleStats(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed. Use GET.", http.StatusMethodNotAllowed)
		return
	}

	// ... rest of the code is the same ...
	client := GetClient()


	
	// client := GetClient()
	// if client == nil {
	// 	http.Error(w, "Redis not connected", http.StatusServiceUnavailable)
	// 	return
	// }

	// Fetch counters from Redis
	total, _ := client.Get(ctx, "stats:total_requests").Result()
	hits, _ := client.Get(ctx, "stats:cache_hits").Result()
	misses, _ := client.Get(ctx, "stats:cache_misses").Result()

	// Basic calculation: Assume each hit saves $0.001 (approx cost of a small GPT query)
	// You can make this math more complex later
	// Note: Redis returns strings, for now we just send them as is.
	
	resp := StatsResponse{
		TotalRequests: total,
		CacheHits:     hits,
		CacheMisses:   misses,
		MoneySaved:    "$0.00", // You can calculate this in frontend or convert string->float here
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
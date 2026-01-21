package handler

import (
	"encoding/json"
	"net/http"
)

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct { Email string `json:"email"` }
	json.NewDecoder(r.Body).Decode(&req)

	apiKey, _ := GenerateAPIKey()
	
	// Create user in Supabase
	query := `INSERT INTO users (email, api_key, requests_used, request_limit) VALUES ($1, $2, 0, 100)`
	_, err := db.Exec(r.Context(), query, req.Email, apiKey)
	if err != nil {
		respondWithError(w, "User already exists or DB error", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"api_key": apiKey})
}
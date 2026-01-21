package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// 1. 🔍 CHECK: Kya user pehle se exist karta hai?
	var existingKey string
	err := db.QueryRow(ctx, "SELECT api_key FROM users WHERE email = $1", req.Email).Scan(&existingKey)

	if err == nil {
		// User found! Purana key wapis bhej do.
		log.Printf("👤 [RETURNING USER] Email: %s", req.Email)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"api_key": existingKey})
		return
	}

	// 2. ✨ CREATE: Agar user nahi mila, toh naya banao
	apiKey, _ := GenerateAPIKey()
	query := `INSERT INTO users (email, api_key, requests_used, request_limit) VALUES ($1, $2, 0, 100)`
	
	_, err = db.Exec(ctx, query, req.Email, apiKey)
	if err != nil {
		log.Printf("❌ [DB ERROR] Registration failed: %v", err)
		respondWithError(w, "Could not create account", http.StatusInternalServerError)
		return
	}

	log.Printf("🆕 [NEW USER] Registered: %s", req.Email)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"api_key": apiKey})
}
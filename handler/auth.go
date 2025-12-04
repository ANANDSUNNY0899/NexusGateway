package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type RegisterRequest struct {
	Email string `json:"email"`
}

type RegisterResponse struct {
	Email  string `json:"email"`
	APIKey string `json:"api_key"`
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	// 1. Only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Parse Email
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// 3. Generate a new Secure Key
	newKey, err := GenerateAPIKey()
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	// 4. Save to Supabase
	// We use QueryRow to ensure we catch duplicate email errors
	var userID string
	query := `INSERT INTO users (email, api_key) VALUES ($1, $2) RETURNING id`
	
	err = db.QueryRow(context.Background(), query, req.Email, newKey).Scan(&userID)
	if err != nil {
		log.Printf("Registration Error: %v", err)
		http.Error(w, "User already exists or DB error", http.StatusConflict)
		return
	}

	log.Printf("👤 New User Registered: %s", req.Email)

	// 5. Return the Key to the User
	resp := RegisterResponse{
		Email:  req.Email,
		APIKey: newKey,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
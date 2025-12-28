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
	Status string `json:"status"` // "created" or "found"
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

	// Safety check
	if db == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	var apiKey string
	var status string

	// 3. CHECK IF USER EXISTS
	// We try to find their key first
	err := db.QueryRow(context.Background(), "SELECT api_key FROM users WHERE email=$1", req.Email).Scan(&apiKey)

	if err == nil {
		// --- SCENARIO A: USER EXISTS ---
		// We found a key! Return it.
		log.Printf("👤 Existing User Logged In: %s", req.Email)
		status = "found"

	} else {
		// --- SCENARIO B: NEW USER ---
		// No key found, so we create one.
		
		newKey, _ := GenerateAPIKey()
		
		// Insert into DB
		var userID string
		insertQuery := `INSERT INTO users (email, api_key) VALUES ($1, $2) RETURNING id`
		err = db.QueryRow(context.Background(), insertQuery, req.Email, newKey).Scan(&userID)
		
		if err != nil {
			log.Printf("Registration Error: %v", err)
			http.Error(w, "Database Error", http.StatusInternalServerError)
			return
		}

		apiKey = newKey
		status = "created"
		log.Printf("👤 New User Registered: %s", req.Email)
	}

	// 4. Return the Key (Whether new or old)
	resp := RegisterResponse{
		Email:  req.Email,
		APIKey: apiKey,
		Status: status,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
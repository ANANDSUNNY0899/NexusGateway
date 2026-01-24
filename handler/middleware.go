package handler

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Helper to send standardized JSON errors
func respondWithError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// 1. AUTH MIDDLEWARE: Security & Quota Gate
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" || !ValidateAPIKey(token) {
			respondWithError(w, "Unauthorized: Invalid Nexus API Key", http.StatusUnauthorized)
			return
		}

		// Exempt system routes
		if r.URL.Path == "/api/checkout" || r.URL.Path == "/api/stats" {
			next(w, r)
			return
		}

		// BYOK Bypass (OpenAI, Groq, Gemini)
		if r.Header.Get("x-nexus-openai-key") != "" || r.Header.Get("x-nexus-groq-key") != "" || r.Header.Get("x-nexus-gemini-key") != "" {
			next(w, r)
			return
		}

		// Standard Quota Check
		allowed, err := CheckUserLimit(token)
		if err != nil || !allowed {
			respondWithError(w, "402 Payment Required: Nexus Credits Depleted", http.StatusPaymentRequired)
			return
		}

		IncrementUsage(token)
		next(w, r)
	}
}

// 2. RATE LIMIT MIDDLEWARE: DDoS Protection (The Missing Function)
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		client := GetClient()
		if client == nil {
			next(w, r) // If Redis is down, allow but log
			return
		}

		key := "ratelimit:" + ip
		limit := 30 // Allow 30 requests per minute per IP

		count, err := client.Incr(r.Context(), key).Result()
		if err == nil && count == 1 {
			client.Expire(r.Context(), key, 1*time.Minute)
		}

		if count > int64(limit) {
			log.Printf("🚫 RATE LIMIT: Blocked IP %s", ip)
			respondWithError(w, "Too many requests. Slow down.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// 3. CORS MIDDLEWARE: Browser Security


func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		
		// 🚀 THE FIX: Add x-nexus-gemini-key and x-nexus-groq-key
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-nexus-openai-key, x-nexus-groq-key, x-nexus-gemini-key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
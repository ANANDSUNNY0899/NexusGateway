package handler

import (
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// 1. AUTH MIDDLEWARE
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A. Get Key
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		if token == "" {
			http.Error(w, "Missing API Key", http.StatusUnauthorized)
			return
		}

		// B. Validate Nexus API Key
		if !ValidateAPIKey(token) {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		// --- EXEMPTIONS ---
		// 1. If they are trying to pay, let them through!
		if r.URL.Path == "/api/checkout" {
			next(w, r)
			return
		}

		// 2. BYOK BYPASS (If they provide ANY provider key, they bypass our quota)
		userOwnOpenAI := r.Header.Get("x-nexus-openai-key")
		userOwnGroq := r.Header.Get("x-nexus-groq-key")
		userOwnGemini := r.Header.Get("x-nexus-gemini-key")

		if userOwnOpenAI != "" || userOwnGroq != "" || userOwnGemini != "" {
			// They are paying the provider directly. We only provide the Caching/Intelligence.
			next(w, r)
			return
		}

		// C. Check Quota (Only for users using Nexus Credits)
		allowed, err := CheckUserLimit(token)
		if err != nil {
			log.Printf("DB Error: %v", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		
		if !allowed {
			http.Error(w, "402 - Quota Exceeded. Upgrade your plan or provide a BYOK key.", http.StatusPaymentRequired)
			return
		}

		// D. Increment Usage (Only if NOT using BYOK)
		IncrementUsage(token)

		// E. Pass
		next(w, r)
	}
}

// 2. RATE LIMIT MIDDLEWARE
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		key := "rate:" + ip
		limit := 20 // Increased for better UX

		client := GetClient()
		if client != nil {
			// Use r.Context() for better request lifecycle management
			count, err := client.Incr(r.Context(), key).Result()
			if err != nil {
				next(w, r)
				return
			}

			if count == 1 {
				client.Expire(r.Context(), key, 1*time.Minute)
			}

			if count > int64(limit) {
				log.Printf("🚫 BLOCKED IP: %s", ip)
				http.Error(w, "429 - Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}
		next(w, r)
	}
}

// 3. CORS MIDDLEWARE (The Frontend Fix)
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		
		// CRITICAL FIX: Added x-nexus-openai-key, x-nexus-groq-key, x-nexus-gemini-key
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-nexus-openai-key, x-nexus-groq-key, x-nexus-gemini-key")

		// Handle Preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
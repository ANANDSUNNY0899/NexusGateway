package handler

import (
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// 1. AUTH MIDDLEWARE (The Bouncer)
// It checks if the user has a valid API Key from our Database
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the header: "Authorization: Bearer nk-test-..."
		authHeader := r.Header.Get("Authorization")
		
		// Clean up the string to get just the key
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		if token == "" {
			http.Error(w, "Missing API Key", http.StatusUnauthorized)
			return
		}

		// Ask the Database: "Is this key real?"
		isValid := ValidateAPIKey(token)
		if !isValid {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		// If valid, let them pass
		next(w, r)
	}
}

// 2. RATE LIMIT MIDDLEWARE (The Traffic Cop)
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		key := "rate:" + ip
		limit := 10 // Increased limit for testing

		client := GetClient()
		if client != nil {
			count, err := client.Incr(ctx, key).Result()
			if err != nil {
				next(w, r)
				return
			}

			if count == 1 {
				client.Expire(ctx, key, 1*time.Minute)
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
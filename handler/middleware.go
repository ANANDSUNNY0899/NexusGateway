package handler

import (
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// 1. AUTH MIDDLEWARE (The Bouncer + The Accountant)
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

		// B. Validate Key
		if !ValidateAPIKey(token) {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		// C. NEW: Check Quota (Do they have credits?)
		allowed, err := CheckUserLimit(token)
		if err != nil {
			log.Printf("DB Error: %v", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		
		if !allowed {
			// This is the money shot. When they see this, they pay.
			http.Error(w, "402 - Quota Exceeded. Upgrade your plan.", http.StatusPaymentRequired)
			return
		}

		// D. NEW: Increment Usage (Charge them 1 credit)
		IncrementUsage(token)

		// E. Pass
		next(w, r)
	}
}

// 2. RATE LIMIT MIDDLEWARE (Speed Control)
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		key := "rate:" + ip
		limit := 10 // Max 10 requests per minute

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
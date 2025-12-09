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

		// B. Validate Key
		if !ValidateAPIKey(token) {
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}

		// <--- NEW FIX: EXEMPT CHECKOUT FROM QUOTA CHECK --->
		// If they are trying to pay, let them through!
		if r.URL.Path == "/api/checkout" {
			next(w, r)
			return
		}
		// <--- END FIX --->

		// C. Check Quota (Do they have credits?)
		allowed, err := CheckUserLimit(token)
		if err != nil {
			log.Printf("DB Error: %v", err)
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		
		if !allowed {
			http.Error(w, "402 - Quota Exceeded. Upgrade your plan.", http.StatusPaymentRequired)
			return
		}

		// D. Increment Usage (Charge them 1 credit)
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
		limit := 10 

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




// CORSMiddleware allows other websites (like your Frontend) to talk to this API
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Allow any origin (You can restrict this to your domain later)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		// 2. Allow specific methods and headers
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// 3. Handle "Preflight" requests (Browsers ask "Can I?" before doing it)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
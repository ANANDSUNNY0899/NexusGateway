package handler

import (
	"log"
	"net"
	"net/http"
	"time"
)

// RateLimitMiddleware checks if the IP has exceeded its limit
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get User IP
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// If we can't detect IP (sometimes happens on localhost), use the whole string
			ip = r.RemoteAddr
		}

		// 2. Define key for Redis (e.g., "rate:127.0.0.1")
		key := "rate:" + ip
		limit := 5 // Max 5 requests per minute

		client := GetClient()
		
		// If Redis is down, we usually allow traffic (Fail Open), or block it. 
		// Here we skip logic if client is nil.
		if client != nil {
			// 3. Increment the counter for this IP
			// Incr does two things:
			// - If key doesn't exist, sets it to 1.
			// - If key exists, adds 1.
			count, err := client.Incr(ctx, key).Result()
			if err != nil {
				log.Printf("Rate limit error: %v", err)
				// Allow request to proceed if Redis fails
				next(w, r)
				return
			}

			// 4. If this is the first request, set an expiration time (1 Minute)
			if count == 1 {
				client.Expire(ctx, key, 1*time.Minute)
			}

			// 5. Check if they exceeded the limit
			if count > int64(limit) {
				log.Printf("🚫 BLOCKED IP: %s (Request #%d)", ip, count)
				http.Error(w, "429 - Too Many Requests. Slow down!", http.StatusTooManyRequests)
				return
			}
		}

		// 6. Allow the request to proceed to the Chat Handler
		next(w, r)
	}
}
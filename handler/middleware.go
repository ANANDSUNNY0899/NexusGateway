// package handler

// import (
// 	"encoding/json"
// 	"log"
// 	"net"
// 	"net/http"
// 	"strings"
// 	"time"
// )

// // Helper to send standardized JSON errors
// func respondWithError(w http.ResponseWriter, msg string, code int) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(code)
// 	json.NewEncoder(w).Encode(map[string]string{"error": msg})
// }

// // 1. AUTH MIDDLEWARE: The Security & Quota Gate
// func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// A. Extract Token
// 		authHeader := r.Header.Get("Authorization")
// 		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

// 		if token == "" || !ValidateAPIKey(token) {
// 			respondWithError(w, "Unauthorized: Invalid Nexus API Key", http.StatusUnauthorized)
// 			return
// 		}

// 		// B. EXEMPTIONS: In raste par quota check nahi hoga
// 		// --- 🚀 THE FIX: Allow Logs and Usage even if Quota is 0 ---
//         if r.URL.Path == "/api/checkout" || r.URL.Path == "/api/user/usage" || r.URL.Path == "/api/stats" || r.URL.Path == "/api/logs" {
// 			next(w, r)
// 			return
//         }

// 		// C. THE BYOK BYPASS
// 		userOwnOpenAI := r.Header.Get("x-nexus-openai-key")
// 		userOwnGroq := r.Header.Get("x-nexus-groq-key")
// 		userOwnGemini := r.Header.Get("x-nexus-gemini-key")

// 		if userOwnOpenAI != "" || userOwnGroq != "" || userOwnGemini != "" {
// 			// User is bringing their own money. Proceed without quota check.
// 			next(w, r)
// 			return
// 		}

// 		// D. QUOTA CHECK
// 		allowed, err := CheckUserLimit(token)
// 		if err != nil || !allowed {
// 			respondWithError(w, "402 Payment Required: Nexus Credits Depleted", http.StatusPaymentRequired)
// 			return
// 		}

// 		// 🚀 THE FIX: IncrementUsage(token) yahan se HATA diya gaya hai.
// 		// Ab hum sirf stream.go ke end mein charge karenge.

// 		next(w, r)
// 	}
// }

// // 2. RATE LIMIT MIDDLEWARE: DDoS Protection
// func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		ip, _, err := net.SplitHostPort(r.RemoteAddr)
// 		if err != nil { ip = r.RemoteAddr }

// 		client := GetClient()
// 		if client == nil {
// 			next(w, r)
// 			return
// 		}

// 		key := "ratelimit:" + ip
// 		limit := 30 

// 		count, err := client.Incr(r.Context(), key).Result()
// 		if err == nil && count == 1 {
// 			client.Expire(r.Context(), key, 1*time.Minute)
// 		}

// 		if count > int64(limit) {
// 			log.Printf("🚫 RATE LIMIT: Blocked IP %s", ip)
// 			respondWithError(w, "Too many requests. Slow down.", http.StatusTooManyRequests)
// 			return
// 		}

// 		next(w, r)
// 	}
// }

// // Inside handler/middleware.go

// func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         w.Header().Set("Access-Control-Allow-Origin", "*")
//         w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        
//         // 🔥 UPDATED: Added Cache-Control and Pragma to the allowed headers list
//         w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-nexus-openai-key, x-nexus-groq-key, x-nexus-gemini-key, x-nexus-anthropic-key, Cache-Control, Pragma")

//         if r.Method == "OPTIONS" {
//             w.WriteHeader(http.StatusOK)
//             return
//         }

//         next(w, r)
//     }
// }



package handler

import (
    "encoding/json"
    "net/http"
    "strings"
    
)

func respondWithError(w http.ResponseWriter, msg string, code int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

        if token == "" || !ValidateAPIKey(token) {
            respondWithError(w, "Unauthorized: Invalid Nexus API Key", http.StatusUnauthorized)
            return
        }

        // 🚀 EXEMPTIONS: Public paths that don't need quota checks (Stats/Usage/Logs)
        if r.URL.Path == "/api/checkout" || r.URL.Path == "/api/user/usage" || r.URL.Path == "/api/stats" || r.URL.Path == "/api/logs" {
            next(w, r)
            return
        }

        // 🔐 BYOK BYPASS: User is paying with their own key
        if r.Header.Get("x-nexus-openai-key") != "" || r.Header.Get("x-nexus-groq-key") != "" || r.Header.Get("x-nexus-gemini-key") != "" {
            next(w, r)
            return
        }

        // 📊 QUOTA GATE
        allowed, err := CheckUserLimit(token)
        if err != nil || !allowed {
            // 💰 Trigger the Cyberpunk Upgrade Screen on frontend
            respondWithError(w, "402 Payment Required: Nexus Credits Depleted", http.StatusPaymentRequired)
            return
        }

        next(w, r)
    }
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        
        // 🛡️ FULL HEADER ACCESS: Includes BYOK keys and Cache controls
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-nexus-openai-key, x-nexus-groq-key, x-nexus-gemini-key, x-nexus-anthropic-key, Cache-Control, Pragma")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next(w, r)
    }
}
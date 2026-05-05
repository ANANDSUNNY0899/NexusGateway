
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

// 2. RATE LIMIT MIDDLEWARE: Satisfies the compiler reference in main.go
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Currently a pass-through to ensure the build completes.
		// You can re-integrate Redis-based limiting after the demo.
		next(w, r)
	}
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
		//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		// 🛡️ CRITICAL: Handle the preflight check before anything else
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	}
}
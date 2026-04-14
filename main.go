package main

import (
	"NexusGateway/config"
	"NexusGateway/handler"
	"log"
	"net/http"
)

func main() {
	// 1. LOAD INFRASTRUCTURE CONFIG
	cfg := config.LoadConfig()

	// 2. INITIALIZE CORE SERVICES
	if cfg.RedisURL != "" {
		handler.InitializeRedis(cfg.RedisURL)
	}

	if cfg.DBUrl != "" {
		handler.InitializeDB(cfg.DBUrl)
	} else {
		log.Println("⚠️  CRITICAL ALERT: DB_URL is missing. Identity-based features will fail.")
	}

	// 3. MIDDLEWARE WRAPPERS
	// Full Stack: CORS -> Auth -> RateLimit
	// Used for inference endpoints to prevent abuse and ensure billing.
	inferenceChain := func(h http.HandlerFunc) http.HandlerFunc {
		return handler.CORSMiddleware(handler.AuthMiddleware(handler.RateLimitMiddleware(h)))
	}

	// Control Stack: CORS -> Auth
	// Used for account management, logs, and billing.
	controlChain := func(h http.HandlerFunc) http.HandlerFunc {
		return handler.CORSMiddleware(handler.AuthMiddleware(h))
	}

	// Public Stack: CORS Only
	// Used for registration and global statistics.
	publicChain := func(h http.HandlerFunc) http.HandlerFunc {
		return handler.CORSMiddleware(h)
	}

	// 4. REGISTER PROTOCOL ROUTES

	// --- A. PUBLIC SYSTEM ROUTES ---
	http.HandleFunc("/api/register", publicChain(handler.HandleRegister))
	http.HandleFunc("/api/stats", publicChain(handler.HandleStats))
	http.HandleFunc("/api/webhook", handler.HandleWebhook) // Direct Stripe access

	// --- B. ACCOUNT & CONTROL PLANE (Protected) ---
	http.HandleFunc("/api/user/usage", controlChain(handler.HandleUserUsage))
	http.HandleFunc("/api/logs", controlChain(handler.HandleGetLogs))
	http.HandleFunc("/api/checkout", controlChain(handler.HandleCheckout))

	// --- C. CORE INFERENCE ENGINE (Protected + Rate Limited) ---
	http.HandleFunc("/api/chat", inferenceChain(handler.HandleChat))
	http.HandleFunc("/api/chat/stream", inferenceChain(handler.HandleStreamChat))

	// --- D. STATIC ASSET SERVING ---
	http.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/success.html")
	})
	http.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/cancel.html")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	})
	// --- E. SYSTEM DIAGNOSTICS ---
    http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        // Simple heartbeat for monitoring services
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status": "operational", "version": "3.1.5", "shield": "active"}`))
    })

	// 5. START INFRASTRUCTURE ENGINE
	log.Printf("---------------------------------------------------------")
	log.Printf("🚀 NEXUS GATEWAY v3.1 [SOVEREIGN EDITION] BOOT SUCCESS")
	log.Printf("📡 PORT: %s | ENV: PRODUCTION", cfg.Port)
	log.Printf("🛡️  SOVEREIGN SHIELD: ACTIVE")
	log.Printf("---------------------------------------------------------")

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("🚨 ENGINE SHUTDOWN: %v", err)
	}
}
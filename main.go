package main

import (
	"NexusGateway/config"
	"NexusGateway/handler"
	"log"
	"net/http"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Initialize Redis
	if cfg.RedisURL != "" {
		handler.InitializeRedis(cfg.RedisURL)
	}

	// 2. NEW: Initialize Database
	if cfg.DBUrl != "" {
		handler.InitializeDB(cfg.DBUrl)
	} else {
		log.Println("⚠️ Skipping DB connection (DB_URL missing)")
	}

	// 3. SECURE ROUTES
	// Order: Request -> Auth -> RateLimit -> Chat Handler
	protectedChat := handler.AuthMiddleware(handler.RateLimitMiddleware(handler.HandleChat))
	
	http.HandleFunc("/api/chat", protectedChat)
	http.HandleFunc("/api/stats", handler.HandleStats)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	})

	log.Printf("🚀 Nexus Gateway running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}
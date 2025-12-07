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

	// 2. Initialize Database
	if cfg.DBUrl != "" {
		handler.InitializeDB(cfg.DBUrl)
	} else {
		log.Println("⚠️ Skipping DB connection (DB_URL missing)")
	}

	// 3. PUBLIC ROUTES (No Key Needed)
	// Anyone can register or see the homepage
	http.HandleFunc("/api/register", handler.HandleRegister)
	http.HandleFunc("/api/webhook", handler.HandleWebhook)
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	})

	// 4. PROTECTED ROUTES (Require API Key)
	// Order: Request -> Auth -> RateLimit -> Chat Handler
	protectedChat := handler.AuthMiddleware(handler.RateLimitMiddleware(handler.HandleChat))
	
	http.HandleFunc("/api/chat", protectedChat)
	http.HandleFunc("/api/stats", handler.HandleStats)

	protectedCheckout := handler.AuthMiddleware(handler.HandleCheckout) 
	http.HandleFunc("/api/checkout", protectedCheckout)

	// 5. Start Server
	log.Printf("🚀 Nexus Gateway V2 (Simple Mode) running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}
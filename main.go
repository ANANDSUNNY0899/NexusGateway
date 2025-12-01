package main

import (
	"NexusGateway/config"
	"NexusGateway/handler"
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Force Redis Initialization
	if cfg.RedisURL == "" {
		log.Println("⚠️ CRITICAL: REDIS_URL is missing. Please set it using 'set REDIS_URL=...'")
	} else {
		log.Println("🔄 Attempting to connect to Redis...")
		handler.InitializeRedis(cfg.RedisURL)
	}

	// 2. Wrap the Chat Handler with the Rate Limiter
	// The request goes: Request -> RateLimitMiddleware -> HandleChat
	http.HandleFunc("/api/chat", handler.RateLimitMiddleware(handler.HandleChat))


	//  Stats Route (Public - for your dashboard)
	http.HandleFunc("/api/stats", handler.HandleStats)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Nexus Gateway is Online! Send POST requests to /api/chat")
	})

	log.Printf("🚀 Nexus Gateway running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}
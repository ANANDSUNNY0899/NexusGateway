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
	// If the URL is empty, we print a loud error to help you debug
	if cfg.RedisURL == "" {
		log.Println("⚠️ CRITICAL: REDIS_URL is missing. Please set it using 'set REDIS_URL=...'")
	} else {
		log.Println("🔄 Attempting to connect to Redis...")
		handler.InitializeRedis(cfg.RedisURL)
	}

	http.HandleFunc("/api/chat", handler.HandleChat)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Nexus Gateway is Online! Send POST requests to /api/chat")
	})

	log.Printf("🚀 Nexus Gateway running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}
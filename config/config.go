package config

import (
	"log"
	"os"
)

type Config struct {
	OpenAIKey string
	RedisURL  string
	Port      string
}

func LoadConfig() *Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY is not set")
	}

	// We verify if the variable is being read
	redisURL := os.Getenv("REDIS_URL")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		OpenAIKey: apiKey,
		RedisURL:  redisURL,
		Port:      port,
	}
}
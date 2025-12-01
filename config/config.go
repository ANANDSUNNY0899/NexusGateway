package config

import (
	"log"
	"os"
)

type Config struct {
	OpenAIKey    string
	RedisURL     string
	PineconeKey  string // <--- You were missing this
	PineconeHost string // <--- You were missing this
	Port         string
}

func LoadConfig() *Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY is not set")
	}

	redisURL := os.Getenv("REDIS_URL")
	
	// Get Pinecone variables
	pineconeKey := os.Getenv("PINECONE_API_KEY")
	pineconeHost := os.Getenv("PINECONE_HOST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		OpenAIKey:    apiKey,
		RedisURL:     redisURL,
		PineconeKey:  pineconeKey,
		PineconeHost: pineconeHost,
		Port:         port,
	}
}
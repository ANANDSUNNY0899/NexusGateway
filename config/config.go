package config

import (
	"log"
	"os"
)

type Config struct {
	OpenAIKey    string
	RedisURL     string
	PineconeKey  string
	PineconeHost string
	DBUrl        string // <--- NEW
	Port         string
}

func LoadConfig() *Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY is not set")
	}

	redisURL := os.Getenv("REDIS_URL")
	pineconeKey := os.Getenv("PINECONE_API_KEY")
	pineconeHost := os.Getenv("PINECONE_HOST")

	// NEW: Get Database URL
	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Println("⚠️ Warning: DB_URL is not set. Auth will fail.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		OpenAIKey:    apiKey,
		RedisURL:     redisURL,
		PineconeKey:  pineconeKey,
		PineconeHost: pineconeHost,
		DBUrl:        dbUrl, // <--- NEW
		Port:         port,
	}
}
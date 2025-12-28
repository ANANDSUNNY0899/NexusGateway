// package config

// import (
// 	"log"
// 	"os"
// 	"strings"
// )

// type Config struct {
// 	OpenAIKey       string
// 	AnthropicKey    string
// 	RedisURL        string
// 	PineconeKey     string
// 	PineconeHost    string
// 	DBUrl           string
// 	StripeSecretKey string // <--- ADDED THIS
// 	Port            string
// 	StripeWebhookSecret string
// }

// func LoadConfig() *Config {
// 	apiKey := os.Getenv("OPENAI_API_KEY")
// 	if apiKey == "" {
// 		log.Fatal("Error: OPENAI_API_KEY is not set")
// 	}

// 	redisURL := os.Getenv("REDIS_URL")
// 	pineconeKey := os.Getenv("PINECONE_API_KEY")
// 	pineconeHost := os.Getenv("PINECONE_HOST")
// 	dbUrl := os.Getenv("DB_URL")
	
// 	// NEW: Get Stripe Key
// 	stripeKey := os.Getenv("STRIPE_SECRET_KEY")

// 	if dbUrl == "" {
// 		log.Println("⚠️ Warning: DB_URL is not set. Auth will fail.")
// 	}

// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "8080"
// 	}

// 	return &Config{
// 		OpenAIKey:       apiKey,
// 		RedisURL:        redisURL,
// 		PineconeKey:     pineconeKey,
// 		PineconeHost:    pineconeHost,
// 		DBUrl:           dbUrl,
// 		StripeSecretKey: stripeKey, // <--- ADDED THIS
// 		Port:            port,
// 		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
// 		AnthropicKey:        os.Getenv("ANTHROPIC_API_KEY"),
// 	}
// }




package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	OpenAIKey           string
	AnthropicKey        string
	RedisURL            string
	PineconeKey         string
	PineconeHost        string
	DBUrl               string
	StripeSecretKey     string
	StripeWebhookSecret string
	Port                string
}

func LoadConfig() *Config {
	// Helper function to clean environment variables (removes spaces/newlines)
	get := func(key string) string {
		return strings.TrimSpace(os.Getenv(key))
	}

	// 1. Get and Clean all keys
	apiKey := get("OPENAI_API_KEY")
	anthropicKey := get("ANTHROPIC_API_KEY")
	redisURL := get("REDIS_URL")
	pineconeKey := get("PINECONE_API_KEY")
	pineconeHost := get("PINECONE_HOST")
	dbUrl := get("DB_URL")
	stripeKey := get("STRIPE_SECRET_KEY")
	webhookSecret := get("STRIPE_WEBHOOK_SECRET")
	port := get("PORT")

	// 2. Validate Critical Keys
	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY is not set")
	}
	if dbUrl == "" {
		log.Println("⚠️ Warning: DB_URL is not set. Auth will fail.")
	}
	if port == "" {
		port = "8080"
	}

	// 3. Return the Clean Config
	return &Config{
		OpenAIKey:           apiKey,
		AnthropicKey:        anthropicKey,
		RedisURL:            redisURL,
		PineconeKey:         pineconeKey,
		PineconeHost:        pineconeHost,
		DBUrl:               dbUrl,
		StripeSecretKey:     stripeKey,
		StripeWebhookSecret: webhookSecret,
		Port:                port,
	}
}
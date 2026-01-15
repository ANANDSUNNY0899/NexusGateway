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
	GroqKey             string
	GeminiKey           string
	RedisURL            string
	PineconeKey         string
	PineconeHost        string
	DBUrl               string
	StripeSecretKey     string
	StripeWebhookSecret string
	Port                string
}

func LoadConfig() *Config {
	get := func(key string) string {
		return strings.TrimSpace(os.Getenv(key))
	}

	apiKey := get("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY is not set")
	}

	port := get("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		OpenAIKey:           apiKey,
		AnthropicKey:        get("ANTHROPIC_API_KEY"),
		GroqKey:             get("GROQ_API_KEY"),
		GeminiKey:           get("GEMINI_API_KEY"),
		RedisURL:            get("REDIS_URL"),
		PineconeKey:         get("PINECONE_API_KEY"),
		PineconeHost:        get("PINECONE_HOST"),
		DBUrl:               get("DB_URL"),
		StripeSecretKey:     get("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: get("STRIPE_WEBHOOK_SECRET"),
		Port:                port,
	}
}
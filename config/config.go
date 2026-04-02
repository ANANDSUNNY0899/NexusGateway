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
    DeepSeekKey         string 
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
        DeepSeekKey:         get("DEEPSEEK_API_KEY"),
        RedisURL:            get("REDIS_URL"),
        PineconeKey:         get("PINECONE_API_KEY"),
        PineconeHost:        get("PINECONE_HOST"),
        DBUrl:               get("DB_URL"),
        StripeSecretKey:     get("STRIPE_SECRET_KEY"),
        StripeWebhookSecret: get("STRIPE_WEBHOOK_SECRET"),
        Port:                port,
    }
}
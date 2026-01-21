package handler

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

var redisClient *redis.Client
var ctx = context.Background()

func InitializeRedis(url string) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("❌ Invalid Redis URL: %v", err)
	}
	redisClient = redis.NewClient(opts)
	log.Println("🚀 Connected to Redis successfully")
}

func GetClient() *redis.Client {
	return redisClient
}
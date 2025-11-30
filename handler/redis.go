package handler

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// Global variables
var redisClient *redis.Client
var ctx = context.Background()

// InitializeRedis connects to the remote database
func InitializeRedis(connString string) {
	opt, err := redis.ParseURL(connString)
	if err != nil {
		log.Fatalf("Invalid Redis URL: %v", err)
	}

	redisClient = redis.NewClient(opt)

	// Ping the database to check if it's actually alive
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf(" Failed to connect to Redis: %v", err)
	}
	
	log.Println(" Connected to Redis successfully")
}

// GetClient returns the client instance
func GetClient() *redis.Client {
	return redisClient
}
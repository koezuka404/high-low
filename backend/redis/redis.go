package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func NewRedis() (redis.UniversalClient, error) {
	_ = godotenv.Load()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewRedisClient initializes and pings a Redis client
func NewRedisClient(env *Env, logger *zap.Logger) *redis.Client {
	addr := fmt.Sprintf("%s:%s", env.RedisHost, env.RedisPort)
	if env.RedisHost == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: env.RedisPassword,
		DB:       env.RedisDB,
	})

	// Test connection with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Warn("Failed to connect to Redis. Continuing without caching.", zap.Error(err), zap.String("addr", addr))
	} else {
		logger.Info("Successfully connected to Redis", zap.String("addr", addr))
	}

	return rdb
}

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

// URLCacheKey is the Redis key caching the URL row for a short code.
func URLCacheKey(code string) string {
	return "url:code:" + code
}

// UserAuthCacheKey caches a user's current role ("" when inactive) so each
// authenticated request needn't hit the database.
func UserAuthCacheKey(userID string) string {
	return "auth:user:" + userID
}

// RevokedTokenKey marks a token ID (jti) as revoked until the token expires.
func RevokedTokenKey(jti string) string {
	return "auth:revoked:" + jti
}

package auth

import (
	"context"
	"time"

	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/redis/go-redis/v9"
)

// userAuthTTL bounds how stale a cached user status/role can be when it
// changes outside the admin API (the admin API invalidates it directly).
const userAuthTTL = time.Minute

// inactiveMarker is cached for users who are deactivated or deleted.
const inactiveMarker = "-"

// TokenStore holds token revocations and the per-user auth cache.
type TokenStore interface {
	Revoke(ctx context.Context, jti string, until time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
	// CachedRole returns the user's cached role, inactiveMarker for an
	// inactive user, or ok=false on a cache miss.
	CachedRole(ctx context.Context, userID string) (role string, ok bool)
	CacheRole(ctx context.Context, userID, role string)
}

type redisTokenStore struct {
	rdb *redis.Client
}

// NewRedisTokenStore returns a Redis-backed TokenStore. With a nil client it
// stores nothing: revocation is unavailable and every lookup is a miss.
func NewRedisTokenStore(rdb *redis.Client) TokenStore {
	return &redisTokenStore{rdb: rdb}
}

func (s *redisTokenStore) Revoke(ctx context.Context, jti string, until time.Time) error {
	ttl := time.Until(until)
	if s.rdb == nil || jti == "" || ttl <= 0 {
		return nil
	}
	return s.rdb.Set(ctx, infra.RevokedTokenKey(jti), "1", ttl).Err()
}

func (s *redisTokenStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if s.rdb == nil || jti == "" {
		return false, nil
	}
	n, err := s.rdb.Exists(ctx, infra.RevokedTokenKey(jti)).Result()
	return n > 0, err
}

func (s *redisTokenStore) CachedRole(ctx context.Context, userID string) (string, bool) {
	if s.rdb == nil {
		return "", false
	}
	role, err := s.rdb.Get(ctx, infra.UserAuthCacheKey(userID)).Result()
	if err != nil {
		return "", false
	}
	return role, true
}

func (s *redisTokenStore) CacheRole(ctx context.Context, userID, role string) {
	if s.rdb == nil {
		return
	}
	s.rdb.Set(ctx, infra.UserAuthCacheKey(userID), role, userAuthTTL)
}

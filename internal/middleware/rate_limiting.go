package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	redis         *redis.Client
	maxRequests   int
	windowSeconds int

	// Fallback in-memory limiters, swept of idle entries so the map cannot
	// grow without bound.
	mu        sync.Mutex
	limiters  map[string]*memLimiter
	lastSweep time.Time
}

type memLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new RateLimiter instance
func NewRateLimiter(redis *redis.Client, maxRequests int, windowSeconds int) *RateLimiter {
	return &RateLimiter{
		redis:         redis,
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
		limiters:      make(map[string]*memLimiter),
	}
}

func (rl *RateLimiter) getInMemoryLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window := time.Duration(rl.windowSeconds) * time.Second

	// A limiter idle for a full window has refilled its bucket, so dropping
	// it loses no state.
	if now.Sub(rl.lastSweep) > window {
		for key, entry := range rl.limiters {
			if now.Sub(entry.lastSeen) > window {
				delete(rl.limiters, key)
			}
		}
		rl.lastSweep = now
	}

	entry, exists := rl.limiters[ip]
	if !exists {
		// Convert maxRequests in windowSeconds to limit per second
		r := rate.Limit(float64(rl.maxRequests) / float64(rl.windowSeconds))
		entry = &memLimiter{limiter: rate.NewLimiter(r, rl.maxRequests)}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = now
	return entry.limiter
}

// Limit returns a Gin middleware for rate limiting
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := c.Request.Context()

		if rl.redis != nil {
			// Redis rate limiting (fixed window)
			now := time.Now().Unix()
			windowNum := now / int64(rl.windowSeconds)
			key := fmt.Sprintf("rate_limit:%s:%d", ip, windowNum)

			count, err := rl.redis.Incr(ctx, key).Result()
			if err == nil {
				if count == 1 {
					rl.redis.Expire(ctx, key, time.Duration(rl.windowSeconds)*time.Second)
				}

				if int(count) > rl.maxRequests {
					c.JSON(http.StatusTooManyRequests, gin.H{
						"status":  "error",
						"message": "Too many requests. Please try again later.",
					})
					c.Abort()
					return
				}
				c.Next()
				return
			}
			// If Redis fails, log and fall through to in-memory fallback
		}

		// Fallback In-Memory Rate Limiting
		limiter := rl.getInMemoryLimiter(ip)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

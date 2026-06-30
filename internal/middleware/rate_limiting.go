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

	// Fallback in-memory limiters
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

// NewRateLimiter creates a new RateLimiter instance
func NewRateLimiter(redis *redis.Client, maxRequests int, windowSeconds int) *RateLimiter {
	return &RateLimiter{
		redis:         redis,
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
		limiters:      make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) getInMemoryLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		// Convert maxRequests in windowSeconds to limit per second
		r := rate.Limit(float64(rl.maxRequests) / float64(rl.windowSeconds))
		limiter = rate.NewLimiter(r, rl.maxRequests)
		rl.limiters[ip] = limiter
	}
	return limiter
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

package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const rateLimitKeyPrefix = "ratelimit:"

// RateLimitConfig defines rate limiting parameters.
type RateLimitConfig struct {
	// MaxAttempts is the maximum number of requests allowed within the Window.
	MaxAttempts int
	// Window is the time period for the rate limit counter.
	Window time.Duration
	// KeyFunc extracts the rate limit key from the request.
	// If it returns an empty string, rate limiting is skipped.
	KeyFunc func(c *gin.Context) string
}

// RateLimiter returns a Gin middleware that enforces request rate limits using Redis.
// It uses a simple counter with TTL (fixed window) per key.
// If redisClient is nil, the middleware is a no-op (fail-open).
func RateLimiter(redisClient *redis.Client, cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil {
			c.Next()
			return
		}

		key := cfg.KeyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		fullKey := rateLimitKeyPrefix + key
		ctx := c.Request.Context()

		count, err := increment(ctx, redisClient, fullKey, cfg.Window)
		if err != nil {
			// If Redis is unavailable, allow the request (fail-open).
			c.Next()
			return
		}

		if count > int64(cfg.MaxAttempts) {
			apierr.TooManyRequests(c, "Too many requests, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// increment atomically increments a counter and sets TTL on first use.
func increment(ctx context.Context, client *redis.Client, key string, window time.Duration) (int64, error) {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set TTL on first increment only.
	if count == 1 {
		client.Expire(ctx, key, window)
	}
	return count, nil
}

// IPRateLimiter creates a rate limiter keyed by client IP + a scope prefix.
func IPRateLimiter(redisClient *redis.Client, scope string, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return RateLimiter(redisClient, RateLimitConfig{
		MaxAttempts: maxAttempts,
		Window:      window,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("%s:ip:%s", scope, c.ClientIP())
		},
	})
}

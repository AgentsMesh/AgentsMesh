package middleware

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const rateLimitKeyPrefix = "ratelimit:"

type RateLimitConfig struct {
	MaxAttempts int
	Window time.Duration
	KeyFunc func(c *gin.Context) string
}

// RateLimiter returns a Gin middleware that enforces request rate limits using Redis.
// It uses a simple counter with TTL (fixed window) per key.
// If redisClient is nil, the middleware is a no-op (fail-open).
//
// Set RATE_LIMIT_DISABLED=true to bypass all rate limiting at process start.
// Used by dev/CI: e2e suites on fast self-hosted runners (8.2 min for 240
// specs) churn through `/auth/login` faster than the prod 20-req/min cap;
// the slower hosted runner happens to stay under the cap by accident.
// Disabling globally avoids leaking environment knowledge into every
// per-route call site, and prod stays opted-in by default.
func RateLimiter(redisClient *redis.Client, cfg RateLimitConfig) gin.HandlerFunc {
	if os.Getenv("RATE_LIMIT_DISABLED") == "true" {
		return func(c *gin.Context) { c.Next() }
	}
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
	if count == 1 {
		client.Expire(ctx, key, window)
	}
	return count, nil
}

func IPRateLimiter(redisClient *redis.Client, scope string, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return RateLimiter(redisClient, RateLimitConfig{
		MaxAttempts: maxAttempts,
		Window:      window,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("%s:ip:%s", scope, c.ClientIP())
		},
	})
}

// UserRateLimiter creates a rate limiter keyed by authenticated user ID + a
// scope prefix. Must be used after AuthMiddleware so GetUserID resolves a
// non-zero ID; otherwise falls back to client IP. If neither is available
// (e.g. tests without RemoteAddr), the limiter returns an empty key and
// skips counting — failing open is preferable to a global shared bucket.
func UserRateLimiter(redisClient *redis.Client, scope string, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return RateLimiter(redisClient, RateLimitConfig{
		MaxAttempts: maxAttempts,
		Window:      window,
		KeyFunc: func(c *gin.Context) string {
			if uid := GetUserID(c); uid != 0 {
				return fmt.Sprintf("%s:user:%d", scope, uid)
			}
			ip := c.ClientIP()
			if ip == "" {
				return ""
			}
			return fmt.Sprintf("%s:ip:%s", scope, ip)
		},
	})
}

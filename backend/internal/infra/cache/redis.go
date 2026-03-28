package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// Cache provides caching functionality using Redis
type Cache struct {
	client *redis.Client
}

// New creates a new Cache instance
func New(cfg *Config) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// Client returns the underlying Redis client
func (c *Cache) Client() *redis.Client {
	return c.client
}

// Set stores a value with expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		slog.Error("redis SET failed", "key", key, "error", err)
		return err
	}
	return nil
}

// Get retrieves a value by key
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		slog.Error("redis GET failed", "key", key, "error", err)
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		slog.Error("redis DEL failed", "keys", keys, "error", err)
		return err
	}
	return nil
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetNX sets a value only if the key doesn't exist (for distributed locking)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}
	ok, err := c.client.SetNX(ctx, key, data, expiration).Result()
	if err != nil {
		slog.Error("redis SETNX failed", "key", key, "error", err)
		return false, err
	}
	return ok, nil
}

// Increment increments a counter
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets expiration on a key
func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the remaining time to live of a key
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Keys returns keys matching a pattern
func (c *Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// Hash operations

// HSet sets a field in a hash
func (c *Cache) HSet(ctx context.Context, key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	if err := c.client.HSet(ctx, key, field, data).Err(); err != nil {
		slog.Error("redis HSET failed", "key", key, "field", field, "error", err)
		return err
	}
	return nil
}

// HGet gets a field from a hash
func (c *Cache) HGet(ctx context.Context, key, field string, dest interface{}) error {
	data, err := c.client.HGet(ctx, key, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		slog.Error("redis HGET failed", "key", key, "field", field, "error", err)
		return err
	}
	return json.Unmarshal(data, dest)
}

// HGetAll gets all fields from a hash
func (c *Cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HDel deletes fields from a hash
func (c *Cache) HDel(ctx context.Context, key string, fields ...string) error {
	if err := c.client.HDel(ctx, key, fields...).Err(); err != nil {
		slog.Error("redis HDEL failed", "key", key, "fields", fields, "error", err)
		return err
	}
	return nil
}

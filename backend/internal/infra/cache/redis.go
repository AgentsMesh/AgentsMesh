package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

type Cache struct {
	client *redis.Client
}

func New(cfg *Config) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Client() *redis.Client {
	return c.client
}

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

func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		slog.Error("redis DEL failed", "keys", keys, "error", err)
		return err
	}
	return nil
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}
	ok, err := c.client.SetNX(ctx, key, data, expiration).Result() //nolint:staticcheck // migrating to Set+NX tracked separately
	if err != nil {
		slog.Error("redis SETNX failed", "key", key, "error", err)
		return false, err
	}
	return ok, nil
}

func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

func (c *Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

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

func (c *Cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

func (c *Cache) HDel(ctx context.Context, key string, fields ...string) error {
	if err := c.client.HDel(ctx, key, fields...).Err(); err != nil {
		slog.Error("redis HDEL failed", "key", key, "fields", fields, "error", err)
		return err
	}
	return nil
}

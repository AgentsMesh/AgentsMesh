package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// List operations

// LPush pushes values to the left of a list
func (c *Cache) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, key, values...).Err()
}

// RPush pushes values to the right of a list
func (c *Cache) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, key, values...).Err()
}

// LRange gets a range of elements from a list
func (c *Cache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

// LTrim trims a list to the specified range
func (c *Cache) LTrim(ctx context.Context, key string, start, stop int64) error {
	return c.client.LTrim(ctx, key, start, stop).Err()
}

// Pub/Sub operations

// Publish publishes a message to a channel
func (c *Cache) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	if err := c.client.Publish(ctx, channel, data).Err(); err != nil {
		slog.Error("redis PUBLISH failed", "channel", channel, "error", err)
		return err
	}
	return nil
}

// Subscribe subscribes to channels
func (c *Cache) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// PSubscribe subscribes to channels matching patterns
func (c *Cache) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	return c.client.PSubscribe(ctx, patterns...)
}

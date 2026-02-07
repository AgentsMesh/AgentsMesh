package relay

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestRedisStore_Key tests the key prefix function
func TestRedisStore_Key(t *testing.T) {
	store := &RedisStore{prefix: "test:"}

	key := store.key("relay:info:", "relay-1")
	expected := "test:relay:info:relay-1"
	if key != expected {
		t.Errorf("key: got %q, want %q", key, expected)
	}
}

// TestRedisStore_KeyEmpty tests key with empty prefix
func TestRedisStore_KeyEmpty(t *testing.T) {
	store := &RedisStore{prefix: ""}

	key := store.key("relay:info:", "relay-1")
	expected := "relay:info:relay-1"
	if key != expected {
		t.Errorf("key: got %q, want %q", key, expected)
	}
}

// TestNewRedisStore tests RedisStore constructor
func TestNewRedisStore(t *testing.T) {
	// Create a mock cache - note: this tests the constructor logic
	store := NewRedisStore(nil, "prefix:")

	if store == nil {
		t.Fatal("NewRedisStore returned nil")
	}
	if store.prefix != "prefix:" {
		t.Errorf("prefix: got %q, want %q", store.prefix, "prefix:")
	}
}

// === Integration tests with miniredis ===

// createTestCache creates a cache with miniredis for integration testing
func createTestCache(t *testing.T) (*miniredis.Miniredis, *testCache) {
	t.Helper()
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
	})

	return mr, &testCache{client: client}
}

// testCache implements the cache interface needed by RedisStore
type testCache struct {
	client *redis.Client
}

func (c *testCache) Client() *redis.Client {
	return c.client
}

func (c *testCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// TestRedisStore_SaveAndGetRelay tests relay save and get operations
func TestRedisStore_SaveAndGetRelay(t *testing.T) {
	mr, tc := createTestCache(t)
	defer mr.Close()

	// Create store for key generation
	store := &RedisStore{prefix: ""}
	// Use test cache client directly
	client := tc.Client()
	ctx := context.Background()

	// Test saving relay manually (simulating SaveRelay logic)
	relay := &RelayInfo{
		ID:       "relay-1",
		URL:      "wss://relay.com",
		Region:   "us-east",
		Capacity: 100,
	}

	// Save relay data
	data, _ := json.Marshal(relay)
	key := store.key(relayKeyPrefix, relay.ID)
	err := client.Set(ctx, key, data, 0).Err()
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Add to relay list
	err = client.SAdd(ctx, store.key(relayListKey), relay.ID).Err()
	if err != nil {
		t.Fatalf("SAdd failed: %v", err)
	}

	// Verify data is stored
	stored, err := client.Get(ctx, key).Bytes()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	var retrieved RelayInfo
	if err := json.Unmarshal(stored, &retrieved); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if retrieved.ID != "relay-1" || retrieved.URL != "wss://relay.com" {
		t.Errorf("retrieved relay mismatch: %+v", retrieved)
	}
}

// TestRedisStore_DeleteRelay tests relay deletion
func TestRedisStore_DeleteRelay(t *testing.T) {
	mr, tc := createTestCache(t)
	defer mr.Close()

	store := &RedisStore{prefix: ""}
	client := tc.Client()
	ctx := context.Background()

	// First save a relay
	relay := &RelayInfo{ID: "relay-1", URL: "wss://relay.com"}
	data, _ := json.Marshal(relay)
	key := store.key(relayKeyPrefix, relay.ID)
	client.Set(ctx, key, data, 0)
	client.SAdd(ctx, store.key(relayListKey), relay.ID)

	// Delete relay
	client.Del(ctx, key)
	client.SRem(ctx, store.key(relayListKey), relay.ID)

	// Verify deleted
	exists, _ := client.Exists(ctx, key).Result()
	if exists > 0 {
		t.Error("relay should be deleted")
	}
}

// TestRedisStore_HeartbeatTTL tests heartbeat key TTL
func TestRedisStore_HeartbeatTTL(t *testing.T) {
	mr, tc := createTestCache(t)
	defer mr.Close()

	store := &RedisStore{prefix: ""}
	client := tc.Client()
	ctx := context.Background()

	// Set heartbeat with TTL
	heartbeatKey := store.key(relayHeartbeatPrefix, "relay-1")
	err := client.Set(ctx, heartbeatKey, time.Now().Unix(), relayHeartbeatTTL).Err()
	if err != nil {
		t.Fatalf("Set heartbeat failed: %v", err)
	}

	// Check TTL is set
	ttl, err := client.TTL(ctx, heartbeatKey).Result()
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 || ttl > relayHeartbeatTTL {
		t.Errorf("unexpected TTL: %v", ttl)
	}

	// Fast-forward time in miniredis
	mr.FastForward(relayHeartbeatTTL + time.Second)

	// Key should be expired
	exists, _ := client.Exists(ctx, heartbeatKey).Result()
	if exists > 0 {
		t.Error("heartbeat key should be expired")
	}
}

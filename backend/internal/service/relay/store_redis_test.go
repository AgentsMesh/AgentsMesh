package relay

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/anthropics/agentsmesh/backend/internal/infra/cache"
	"github.com/redis/go-redis/v9"
)

// === Unit tests (no Redis needed) ===

func TestRedisStore_Key(t *testing.T) {
	store := &RedisStore{prefix: "test:"}
	key := store.key("relay:info:", "relay-1")
	if want := "test:relay:info:relay-1"; key != want {
		t.Errorf("key: got %q, want %q", key, want)
	}
}

func TestRedisStore_KeyEmpty(t *testing.T) {
	store := &RedisStore{prefix: ""}
	key := store.key("relay:info:", "relay-1")
	if want := "relay:info:relay-1"; key != want {
		t.Errorf("key: got %q, want %q", key, want)
	}
}

func TestNewRedisStore(t *testing.T) {
	store := NewRedisStore(nil, "prefix:")
	if store == nil {
		t.Fatal("NewRedisStore returned nil")
	}
	if store.prefix != "prefix:" {
		t.Errorf("prefix: got %q, want %q", store.prefix, "prefix:")
	}
}

// === Integration tests with miniredis (through real RedisStore methods) ===

// createTestStore creates a RedisStore backed by miniredis for integration testing.
func createTestStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)

	host, portStr, err := net.SplitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	port, _ := strconv.Atoi(portStr)

	c, err := cache.New(&cache.Config{Host: host, Port: port})
	if err != nil {
		t.Fatalf("create cache: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	return NewRedisStore(c, ""), mr
}

func TestRedisStore_SaveAndGetRelay_E2E(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	relay := &RelayInfo{
		ID:            "relay-1",
		URL:           "wss://relay.com:8443",
		Region:        "us-east",
		Capacity:      500,
		Latitude:      40.7128,
		Longitude:     -74.0060,
		LastHeartbeat: time.Now(),
	}

	// Save
	if err := store.SaveRelay(ctx, relay); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	// Get
	got, err := store.GetRelay(ctx, "relay-1")
	if err != nil {
		t.Fatalf("GetRelay: %v", err)
	}
	if got == nil {
		t.Fatal("GetRelay returned nil")
	}

	// Verify all fields survive JSON round-trip
	if got.ID != "relay-1" {
		t.Errorf("ID: got %q, want %q", got.ID, "relay-1")
	}
	if got.URL != "wss://relay.com:8443" {
		t.Errorf("URL: got %q, want %q", got.URL, "wss://relay.com:8443")
	}
	if got.Region != "us-east" {
		t.Errorf("Region: got %q, want %q", got.Region, "us-east")
	}
	if got.Capacity != 500 {
		t.Errorf("Capacity: got %d, want 500", got.Capacity)
	}
	if got.Latitude != 40.7128 {
		t.Errorf("Latitude: got %f, want 40.7128", got.Latitude)
	}
	if got.Longitude != -74.0060 {
		t.Errorf("Longitude: got %f, want -74.0060", got.Longitude)
	}

	// Heartbeat key is set by SaveRelay → relay should be healthy
	if !got.Healthy {
		t.Error("relay should be healthy (heartbeat key set by SaveRelay)")
	}
}

func TestRedisStore_GetRelay_NotFound(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	got, err := store.GetRelay(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetRelay: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent relay, got %+v", got)
	}
}

func TestRedisStore_GetRelay_HealthFromHeartbeat(t *testing.T) {
	store, mr := createTestStore(t)
	ctx := context.Background()

	relay := &RelayInfo{
		ID:            "relay-1",
		URL:           "wss://relay.com",
		LastHeartbeat: time.Now(),
	}
	if err := store.SaveRelay(ctx, relay); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	// With heartbeat key alive → healthy
	got, _ := store.GetRelay(ctx, "relay-1")
	if !got.Healthy {
		t.Error("relay should be healthy when heartbeat key exists")
	}

	// After heartbeat key expires → unhealthy
	mr.FastForward(relayHeartbeatTTL + time.Second)

	got, _ = store.GetRelay(ctx, "relay-1")
	if got.Healthy {
		t.Error("relay should be unhealthy when heartbeat key expired")
	}
}

func TestRedisStore_GetAllRelays_E2E(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()
	now := time.Now()

	// Save multiple relays
	for _, id := range []string{"relay-1", "relay-2", "relay-3"} {
		r := &RelayInfo{ID: id, URL: "wss://" + id + ".com", LastHeartbeat: now}
		if err := store.SaveRelay(ctx, r); err != nil {
			t.Fatalf("SaveRelay(%s): %v", id, err)
		}
	}

	relays, err := store.GetAllRelays(ctx)
	if err != nil {
		t.Fatalf("GetAllRelays: %v", err)
	}
	if len(relays) != 3 {
		t.Fatalf("expected 3 relays, got %d", len(relays))
	}

	// All should be healthy (heartbeat keys set by SaveRelay)
	for _, r := range relays {
		if !r.Healthy {
			t.Errorf("relay %s should be healthy", r.ID)
		}
	}
}

func TestRedisStore_GetAllRelays_Empty(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	relays, err := store.GetAllRelays(ctx)
	if err != nil {
		t.Fatalf("GetAllRelays: %v", err)
	}
	if relays != nil {
		t.Errorf("expected nil for empty store, got %d relays", len(relays))
	}
}

func TestRedisStore_GetAllRelays_OrphanCleanup(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()
	client := store.cache.Client()

	// Add orphan ID to relay:list without relay:info data
	client.SAdd(ctx, store.key(relayListKey), "orphan-1", "orphan-2")

	// Save one real relay
	r := &RelayInfo{ID: "real-1", URL: "wss://real.com", LastHeartbeat: time.Now()}
	if err := store.SaveRelay(ctx, r); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	relays, err := store.GetAllRelays(ctx)
	if err != nil {
		t.Fatalf("GetAllRelays: %v", err)
	}
	if len(relays) != 1 || relays[0].ID != "real-1" {
		t.Errorf("expected only real-1, got %d relays", len(relays))
	}

	// Verify orphans were cleaned from relay:list
	members, _ := client.SMembers(ctx, store.key(relayListKey)).Result()
	if len(members) != 1 {
		t.Errorf("orphans should be cleaned: got %d members: %v", len(members), members)
	}
}

func TestRedisStore_GetAllRelays_LastHeartbeatRefresh(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	// Save relay with LastHeartbeat set to 1 hour ago (simulates stale relay:info JSON)
	oldTime := time.Now().Add(-time.Hour)
	r := &RelayInfo{ID: "relay-1", URL: "wss://relay.com", LastHeartbeat: oldTime}
	if err := store.SaveRelay(ctx, r); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	// GetAllRelays should refresh LastHeartbeat to now since heartbeat key is alive
	relays, err := store.GetAllRelays(ctx)
	if err != nil {
		t.Fatalf("GetAllRelays: %v", err)
	}
	if len(relays) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(relays))
	}

	// LastHeartbeat should be recent (within last second), not 1 hour ago
	elapsed := time.Since(relays[0].LastHeartbeat)
	if elapsed > 5*time.Second {
		t.Errorf("LastHeartbeat should be refreshed to now, but is %v old", elapsed)
	}
}

func TestRedisStore_DeleteRelay_E2E(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	// Save
	r := &RelayInfo{ID: "relay-1", URL: "wss://relay.com", LastHeartbeat: time.Now()}
	if err := store.SaveRelay(ctx, r); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	// Verify exists
	got, _ := store.GetRelay(ctx, "relay-1")
	if got == nil {
		t.Fatal("relay should exist after save")
	}

	// Delete
	if err := store.DeleteRelay(ctx, "relay-1"); err != nil {
		t.Fatalf("DeleteRelay: %v", err)
	}

	// Verify gone
	got, _ = store.GetRelay(ctx, "relay-1")
	if got != nil {
		t.Error("relay should be nil after delete")
	}

	// Verify cleaned from relay:list
	client := store.cache.Client()
	isMember, _ := client.SIsMember(ctx, store.key(relayListKey), "relay-1").Result()
	if isMember {
		t.Error("relay should be removed from relay:list")
	}
}

func TestRedisStore_UpdateRelayHeartbeat_E2E(t *testing.T) {
	store, mr := createTestStore(t)
	ctx := context.Background()

	// Save relay
	r := &RelayInfo{ID: "relay-1", URL: "wss://relay.com", LastHeartbeat: time.Now()}
	if err := store.SaveRelay(ctx, r); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	// Fast-forward past initial heartbeat TTL
	mr.FastForward(relayHeartbeatTTL - 5*time.Second)

	// Update heartbeat (refreshes TTL)
	if err := store.UpdateRelayHeartbeat(ctx, "relay-1", time.Now()); err != nil {
		t.Fatalf("UpdateRelayHeartbeat: %v", err)
	}

	// Fast-forward another 10 seconds (total > original TTL, but within refreshed TTL)
	mr.FastForward(10 * time.Second)

	// Heartbeat key should still exist (TTL was refreshed)
	got, _ := store.GetRelay(ctx, "relay-1")
	if got == nil {
		t.Fatal("relay should still exist")
	}
	if !got.Healthy {
		t.Error("relay should be healthy after heartbeat refresh")
	}
}

func TestRedisStore_HeartbeatTTL_Expiry(t *testing.T) {
	store, mr := createTestStore(t)
	ctx := context.Background()

	// Save relay (sets heartbeat key with TTL)
	r := &RelayInfo{ID: "relay-1", URL: "wss://relay.com", LastHeartbeat: time.Now()}
	if err := store.SaveRelay(ctx, r); err != nil {
		t.Fatalf("SaveRelay: %v", err)
	}

	// Verify heartbeat key TTL
	client := store.cache.Client()
	ttl, _ := client.TTL(ctx, store.key(relayHeartbeatPrefix, "relay-1")).Result()
	if ttl <= 0 || ttl > relayHeartbeatTTL {
		t.Errorf("unexpected TTL: %v", ttl)
	}

	// Fast-forward past TTL
	mr.FastForward(relayHeartbeatTTL + time.Second)

	// Heartbeat key expired ��� relay unhealthy
	got, _ := store.GetRelay(ctx, "relay-1")
	if got == nil {
		t.Fatal("relay info should still exist (only heartbeat key expired)")
	}
	if got.Healthy {
		t.Error("relay should be unhealthy after heartbeat TTL expiry")
	}
}

// === Legacy helper (kept for backwards compatibility if needed) ===

// testCache wraps a redis.Client for tests that need raw client access.
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

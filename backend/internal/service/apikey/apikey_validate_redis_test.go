package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServiceWithRedis creates a Service backed by a miniredis instance.
func newTestServiceWithRedis(t *testing.T) (*Service, *miniredis.Miniredis) {
	t.Helper()
	db := setupTestDB(t)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		rdb.Close()
	})

	svc := NewService(infra.NewAPIKeyRepository(db), rdb)
	return svc, mr
}

// computeKeyHash computes the SHA-256 hash of a raw key, same as the service does.
func computeKeyHash(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

func TestValidateKeyWithRedisCache(t *testing.T) {
	ctx := context.Background()

	t.Run("first call populates cache, second call uses cache", func(t *testing.T) {
		svc, mr := newTestServiceWithRedis(t)
		resp, _ := createTestAPIKey(t, svc, 1, "cached-key", []string{"pods:read", "tickets:write"})

		// First call — cache miss, hits DB, then caches
		result, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)
		assert.Equal(t, "cached-key", result.KeyName)

		// Verify something was written to Redis
		keys := mr.Keys()
		assert.NotEmpty(t, keys, "expected cache key to be set in Redis")

		// Second call — cache hit
		result2, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)
		assert.Equal(t, result.APIKeyID, result2.APIKeyID)
		assert.Equal(t, result.KeyName, result2.KeyName)
	})

	t.Run("disabled key from cache returns ErrAPIKeyDisabled", func(t *testing.T) {
		db := setupTestDB(t)
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { rdb.Close() })
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		resp, key := createTestAPIKey(t, svc, 1, "cache-disabled", []string{"pods:read"})

		// First call — populate cache
		_, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)

		// Disable in DB
		db.Model(&apikeyDomain.APIKey{}).Where("id = ?", key.ID).Update("is_enabled", false)

		// Invalidate cache so DB is re-read
		svc.invalidateCache(ctx, key.KeyHash)

		// Validate — hits DB, sees disabled, caches with disabled state, returns error
		_, err = svc.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyDisabled)

		// Third call — should hit cache this time with disabled state
		_, err = svc.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyDisabled)
	})

	t.Run("expired key from cache returns ErrAPIKeyExpired and invalidates cache", func(t *testing.T) {
		db := setupTestDB(t)
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { rdb.Close() })
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		resp, key := createTestAPIKey(t, svc, 1, "cache-expired", []string{"pods:read"})

		// Populate cache
		_, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)

		// Set expires_at to past in DB
		past := time.Now().Add(-1 * time.Hour)
		db.Model(&apikeyDomain.APIKey{}).Where("id = ?", key.ID).Update("expires_at", past)

		// Invalidate cache so next call hits DB
		svc.invalidateCache(ctx, key.KeyHash)

		// Validate — hits DB, sees expired, caches with expired state, returns error
		_, err = svc.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyExpired)

		// Now validate again — should hit cache, detect expiry, invalidate, return error
		_, err = svc.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyExpired)
	})

	t.Run("update invalidates cache", func(t *testing.T) {
		db := setupTestDB(t)
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { rdb.Close() })
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		resp, key := createTestAPIKey(t, svc, 1, "update-cache", []string{"pods:read"})

		// Populate cache
		_, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)
		assert.NotEmpty(t, mr.Keys())

		// Update key — should invalidate cache
		_, err = svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr("updated-cache-name"),
		})
		require.NoError(t, err)

		// Validate again — should hit DB with updated data
		result, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)
		assert.Equal(t, "updated-cache-name", result.KeyName)
	})

	t.Run("revoke invalidates cache", func(t *testing.T) {
		db := setupTestDB(t)
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { rdb.Close() })
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		resp, key := createTestAPIKey(t, svc, 1, "revoke-cache", []string{"pods:read"})

		// Populate cache
		_, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)

		// Revoke — should invalidate cache
		err = svc.RevokeAPIKey(ctx, key.ID, 1)
		require.NoError(t, err)

		// Validate — should hit DB and see disabled
		_, err = svc.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyDisabled)
	})

	t.Run("delete invalidates cache", func(t *testing.T) {
		db := setupTestDB(t)
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { rdb.Close() })
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		resp, key := createTestAPIKey(t, svc, 1, "delete-cache", []string{"pods:read"})

		// Populate cache
		_, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)

		// Delete — should invalidate cache
		err = svc.DeleteAPIKey(ctx, key.ID, 1)
		require.NoError(t, err)

		// Validate — should hit DB and see not found
		_, err = svc.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})
}

func TestGetFromCache(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for nil redis", func(t *testing.T) {
		svc, _ := newTestService(t) // nil redis
		cached, err := svc.getFromCache(ctx, "somehash")
		assert.Error(t, err)
		assert.Nil(t, cached)
	})

	t.Run("returns error for cache miss", func(t *testing.T) {
		svc, _ := newTestServiceWithRedis(t)
		cached, err := svc.getFromCache(ctx, "nonexistenthash")
		assert.Error(t, err) // redis.Nil
		assert.Nil(t, cached)
	})

	t.Run("returns cached data for cache hit", func(t *testing.T) {
		svc, _ := newTestServiceWithRedis(t)
		resp, _ := createTestAPIKey(t, svc, 1, "cache-test", []string{"pods:read"})

		// Populate cache via ValidateKey
		_, err := svc.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)

		// Now getFromCache should return the data
		keyHash := computeKeyHash(resp.RawKey)
		cached, err := svc.getFromCache(ctx, keyHash)
		require.NoError(t, err)
		require.NotNil(t, cached)
		assert.Equal(t, "cache-test", cached.KeyName)
		assert.True(t, cached.IsEnabled)
	})
}

func TestSetCache(t *testing.T) {
	ctx := context.Background()

	t.Run("does nothing for nil redis", func(t *testing.T) {
		svc, _ := newTestService(t)
		// Should not panic
		svc.setCache(ctx, "hash", &cachedKeyData{APIKeyID: 1})
	})

	t.Run("stores data in redis", func(t *testing.T) {
		svc, mr := newTestServiceWithRedis(t)
		svc.setCache(ctx, "testhash", &cachedKeyData{
			APIKeyID: 99,
			KeyName:  "cached",
		})
		assert.True(t, mr.Exists(cachePrefix+"testhash"))
	})

	t.Run("logs warning on redis Set error", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		db := setupTestDB(t)
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		// Close miniredis to cause Set to fail
		mr.Close()

		// Should not panic, just logs warning
		svc.setCache(ctx, "failhash", &cachedKeyData{APIKeyID: 1})
	})
}

func TestInvalidateCache(t *testing.T) {
	ctx := context.Background()

	t.Run("does nothing for nil redis", func(t *testing.T) {
		svc, _ := newTestService(t)
		// Should not panic
		svc.invalidateCache(ctx, "hash")
	})

	t.Run("removes key from redis", func(t *testing.T) {
		svc, mr := newTestServiceWithRedis(t)

		// Set a value first
		svc.setCache(ctx, "removeme", &cachedKeyData{APIKeyID: 1})
		assert.True(t, mr.Exists(cachePrefix+"removeme"))

		// Invalidate
		svc.invalidateCache(ctx, "removeme")
		assert.False(t, mr.Exists(cachePrefix+"removeme"))
	})

	t.Run("logs warning on redis Del error", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		db := setupTestDB(t)
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		// Close miniredis to cause Del to fail
		mr.Close()

		// Should not panic, just logs warning
		svc.invalidateCache(ctx, "failhash")
	})
}

func TestGetFromCacheInvalidJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for invalid JSON in cache", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { rdb.Close() })
		db := setupTestDB(t)
		svc := NewService(infra.NewAPIKeyRepository(db), rdb)

		// Write invalid JSON directly to the cache key
		mr.Set(cachePrefix+"badjson", "not-valid-json")

		cached, err := svc.getFromCache(ctx, "badjson")
		assert.Error(t, err)
		assert.Nil(t, cached)
	})
}

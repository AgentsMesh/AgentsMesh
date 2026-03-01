package apikey

import (
	"context"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMiddlewareAdapter(t *testing.T) {
	svc, _ := newTestService(t)
	adapter := NewMiddlewareAdapter(svc)
	require.NotNil(t, adapter)
	assert.Equal(t, svc, adapter.svc)
}

func TestMiddlewareAdapterValidateKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success maps fields correctly", func(t *testing.T) {
		svc, _ := newTestService(t)
		adapter := NewMiddlewareAdapter(svc)

		resp, _ := createTestAPIKey(t, svc, 1, "adapter-key", []string{"pods:read", "tickets:write"})

		result, err := adapter.ValidateKey(ctx, resp.RawKey)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, resp.APIKey.ID, result.APIKeyID)
		assert.Equal(t, int64(1), result.OrganizationID)
		assert.Equal(t, int64(1), result.CreatedBy)
		assert.Equal(t, "adapter-key", result.KeyName)
		assert.ElementsMatch(t, []string{"pods:read", "tickets:write"}, result.Scopes)
	})

	t.Run("not found error is translated to middleware sentinel", func(t *testing.T) {
		svc, _ := newTestService(t)
		adapter := NewMiddlewareAdapter(svc)

		_, err := adapter.ValidateKey(ctx, "amk_nonexistent_key_0000000000000000000000000000000000000000000000000000000000000000000000000000000")
		require.Error(t, err)
		// translateError wraps middleware.ErrAPIKeyNotFound, not service ErrAPIKeyNotFound
		assert.ErrorIs(t, err, middleware.ErrAPIKeyNotFound)
	})

	t.Run("disabled error is translated to middleware sentinel", func(t *testing.T) {
		svc, db := newTestService(t)
		adapter := NewMiddlewareAdapter(svc)
		resp, _ := createDisabledAPIKey(t, svc, db, 1, "disabled-adapt", []string{"pods:read"})

		_, err := adapter.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, middleware.ErrAPIKeyDisabled)
	})

	t.Run("expired error is translated to middleware sentinel", func(t *testing.T) {
		svc, db := newTestService(t)
		adapter := NewMiddlewareAdapter(svc)
		resp, _ := createExpiredAPIKey(t, svc, db, 1, "expired-adapt", []string{"pods:read"})

		_, err := adapter.ValidateKey(ctx, resp.RawKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, middleware.ErrAPIKeyExpired)
	})
}

func TestMiddlewareAdapterUpdateLastUsed(t *testing.T) {
	ctx := context.Background()

	svc, _ := newTestService(t)
	adapter := NewMiddlewareAdapter(svc)
	_, key := createTestAPIKey(t, svc, 1, "last-used-adapter", []string{"pods:read"})

	err := adapter.UpdateLastUsed(ctx, key.ID)
	require.NoError(t, err)
}

func TestTranslateError(t *testing.T) {
	t.Run("ErrAPIKeyNotFound is translated to middleware error", func(t *testing.T) {
		err := translateError(ErrAPIKeyNotFound)
		require.Error(t, err)
		assert.ErrorIs(t, err, middleware.ErrAPIKeyNotFound)
		// Should NOT match the service-level error
		assert.NotErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("ErrAPIKeyDisabled is translated to middleware error", func(t *testing.T) {
		err := translateError(ErrAPIKeyDisabled)
		require.Error(t, err)
		assert.ErrorIs(t, err, middleware.ErrAPIKeyDisabled)
		assert.NotErrorIs(t, err, ErrAPIKeyDisabled)
	})

	t.Run("ErrAPIKeyExpired is translated to middleware error", func(t *testing.T) {
		err := translateError(ErrAPIKeyExpired)
		require.Error(t, err)
		assert.ErrorIs(t, err, middleware.ErrAPIKeyExpired)
		assert.NotErrorIs(t, err, ErrAPIKeyExpired)
	})

	t.Run("unknown error passes through unchanged", func(t *testing.T) {
		original := assert.AnError
		err := translateError(original)
		assert.Equal(t, original, err)
	})
}

package apikey

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── UpdateAPIKey ──────────────────────────────────────────────────

func TestUpdateAPIKey(t *testing.T) {
	ctx := context.Background()

	t.Run("update name", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "original-name", []string{"pods:read"})

		updated, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr("new-name"),
		})
		require.NoError(t, err)
		assert.Equal(t, "new-name", updated.Name)
	})

	t.Run("update description", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "desc-key", []string{"pods:read"})

		updated, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Description: strPtr("new description"),
		})
		require.NoError(t, err)
		require.NotNil(t, updated.Description)
		assert.Equal(t, "new description", *updated.Description)
	})

	t.Run("update scopes", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "scope-key", []string{"pods:read"})

		updated, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Scopes: []string{"tickets:read", "tickets:write"},
		})
		require.NoError(t, err)
		assert.Equal(t, apikey.Scopes{apikey.ScopeTicketRead, apikey.ScopeTicketWrite}, updated.Scopes)
	})

	t.Run("update is_enabled", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "enable-key", []string{"pods:read"})

		updated, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			IsEnabled: boolPtr(false),
		})
		require.NoError(t, err)
		assert.False(t, updated.IsEnabled)

		// Re-enable
		updated2, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			IsEnabled: boolPtr(true),
		})
		require.NoError(t, err)
		assert.True(t, updated2.IsEnabled)
	})

	t.Run("invalid scope error", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "bad-scope-update", []string{"pods:read"})

		_, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Scopes: []string{"invalid:scope"},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScope)
	})

	t.Run("duplicate name error", func(t *testing.T) {
		svc, _ := newTestService(t)
		createTestAPIKey(t, svc, 1, "existing-name", []string{"pods:read"})
		_, key2 := createTestAPIKey(t, svc, 1, "other-name", []string{"pods:read"})

		_, err := svc.UpdateAPIKey(ctx, key2.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr("existing-name"),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateKeyName)
	})

	t.Run("renaming to same name (self) is OK", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "keep-same", []string{"pods:read"})

		updated, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr("keep-same"),
		})
		require.NoError(t, err)
		assert.Equal(t, "keep-same", updated.Name)
	})

	t.Run("wrong org returns ErrAPIKeyNotFound", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "org-check-key", []string{"pods:read"})

		_, err := svc.UpdateAPIKey(ctx, key.ID, 2, &UpdateAPIKeyRequest{
			Name: strPtr("sneaky"),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("name too long error", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "long-name-key", []string{"pods:read"})

		longName := strings.Repeat("x", 256)
		_, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: &longName,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNameTooLong)
	})

	t.Run("non-existent key returns ErrAPIKeyNotFound", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.UpdateAPIKey(ctx, 9999, 1, &UpdateAPIKeyRequest{
			Name: strPtr("ghost"),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("empty name returns ErrNameEmpty", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "update-empty-name", []string{"pods:read"})

		_, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr(""),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNameEmpty)
	})

	t.Run("whitespace-only name returns ErrNameEmpty", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "update-ws-name", []string{"pods:read"})

		_, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr("   \t  "),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNameEmpty)
	})

	t.Run("name with spaces is trimmed on update", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "trim-update-key", []string{"pods:read"})

		updated, err := svc.UpdateAPIKey(ctx, key.ID, 1, &UpdateAPIKeyRequest{
			Name: strPtr("  trimmed-name  "),
		})
		require.NoError(t, err)
		assert.Equal(t, "trimmed-name", updated.Name)
	})
}

// ─── RevokeAPIKey ──────────────────────────────────────────────────

func TestRevokeAPIKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success sets is_enabled false", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "revoke-key", []string{"pods:read"})
		assert.True(t, key.IsEnabled)

		err := svc.RevokeAPIKey(ctx, key.ID, 1)
		require.NoError(t, err)

		// Verify in DB
		fetched, err := svc.GetAPIKey(ctx, key.ID, 1)
		require.NoError(t, err)
		assert.False(t, fetched.IsEnabled)
	})

	t.Run("non-existent key returns ErrAPIKeyNotFound", func(t *testing.T) {
		svc, _ := newTestService(t)

		err := svc.RevokeAPIKey(ctx, 9999, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("wrong org returns ErrAPIKeyNotFound", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "revoke-wrong-org", []string{"pods:read"})

		err := svc.RevokeAPIKey(ctx, key.ID, 2)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})
}

// ─── DeleteAPIKey ──────────────────────────────────────────────────

func TestDeleteAPIKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "delete-key", []string{"pods:read"})

		err := svc.DeleteAPIKey(ctx, key.ID, 1)
		require.NoError(t, err)

		// Verify it's gone
		_, err = svc.GetAPIKey(ctx, key.ID, 1)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("non-existent key returns ErrAPIKeyNotFound", func(t *testing.T) {
		svc, _ := newTestService(t)

		err := svc.DeleteAPIKey(ctx, 9999, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("wrong org returns ErrAPIKeyNotFound", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "delete-wrong-org", []string{"pods:read"})

		err := svc.DeleteAPIKey(ctx, key.ID, 2)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})
}

// ─── UpdateLastUsed ────────────────────────────────────────────────

func TestUpdateLastUsed(t *testing.T) {
	ctx := context.Background()

	t.Run("updates timestamp", func(t *testing.T) {
		svc, _ := newTestService(t)
		_, key := createTestAPIKey(t, svc, 1, "last-used-key", []string{"pods:read"})
		assert.Nil(t, key.LastUsedAt)

		before := time.Now().Add(-time.Second)
		err := svc.UpdateLastUsed(ctx, key.ID)
		require.NoError(t, err)

		fetched, err := svc.GetAPIKey(ctx, key.ID, 1)
		require.NoError(t, err)
		require.NotNil(t, fetched.LastUsedAt)
		assert.True(t, fetched.LastUsedAt.After(before))
	})

	t.Run("non-existent key does not error (0 rows affected)", func(t *testing.T) {
		svc, _ := newTestService(t)

		// UpdateLastUsed uses a simple UPDATE ... WHERE id = ?, which succeeds
		// even when 0 rows match. This is by design (fire-and-forget).
		err := svc.UpdateLastUsed(ctx, 9999)
		require.NoError(t, err)
	})
}

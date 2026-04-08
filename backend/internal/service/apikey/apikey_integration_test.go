package apikey

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationService creates a Service backed by an in-memory DB for
// integration tests. Returns the service and a background context.
func setupIntegrationService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	svc, _ := newTestService(t)
	return svc, context.Background()
}

func TestAPIKey_CreateAndValidate(t *testing.T) {
	svc, ctx := setupIntegrationService(t)

	resp, err := svc.CreateAPIKey(ctx, &CreateAPIKeyRequest{
		OrganizationID: 1,
		CreatedBy:      10,
		Name:           "ci-key",
		Description:    strPtr("CI pipeline key"),
		Scopes:         []string{"pods:read", "tickets:write"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, strings.HasPrefix(resp.RawKey, "amk_"))

	// ValidateKey with the raw key should return correct metadata
	result, err := svc.ValidateKey(ctx, resp.RawKey)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.OrganizationID)
	assert.Equal(t, int64(10), result.CreatedBy)
	assert.Equal(t, "ci-key", result.KeyName)
	assert.Contains(t, result.Scopes, "pods:read")
	assert.Contains(t, result.Scopes, "tickets:write")
	assert.Equal(t, resp.APIKey.ID, result.APIKeyID)
}

func TestAPIKey_ListAndGet(t *testing.T) {
	svc, ctx := setupIntegrationService(t)

	resp1, _ := createTestAPIKey(t, svc, 1, "key-alpha", []string{"pods:read"})
	createTestAPIKey(t, svc, 1, "key-beta", []string{"tickets:read"})

	// List — should see both keys
	keys, total, err := svc.ListAPIKeys(ctx, &ListAPIKeysFilter{
		OrganizationID: 1,
		Limit:          10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, keys, 2)

	// Get by ID — should return correct key
	fetched, err := svc.GetAPIKey(ctx, resp1.APIKey.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "key-alpha", fetched.Name)

	// Org isolation — different org should not find the key
	_, err = svc.GetAPIKey(ctx, resp1.APIKey.ID, 999)
	assert.ErrorIs(t, err, ErrAPIKeyNotFound)
}

func TestAPIKey_Update(t *testing.T) {
	svc, ctx := setupIntegrationService(t)

	resp, _ := createTestAPIKey(t, svc, 1, "original-name", []string{"pods:read"})

	updated, err := svc.UpdateAPIKey(ctx, resp.APIKey.ID, 1, &UpdateAPIKeyRequest{
		Name:   strPtr("renamed-key"),
		Scopes: []string{"pods:read", "pods:write"},
	})
	require.NoError(t, err)
	assert.Equal(t, "renamed-key", updated.Name)
	assert.Len(t, updated.Scopes, 2)

	// Re-fetch to confirm persistence
	fetched, err := svc.GetAPIKey(ctx, resp.APIKey.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "renamed-key", fetched.Name)
	assert.True(t, fetched.Scopes.HasScope("pods:write"))
}

func TestAPIKey_RevokeAndDelete(t *testing.T) {
	svc, ctx := setupIntegrationService(t)

	resp, _ := createTestAPIKey(t, svc, 1, "revoke-me", []string{"pods:read"})
	rawKey := resp.RawKey

	// Revoke — disables the key
	err := svc.RevokeAPIKey(ctx, resp.APIKey.ID, 1)
	require.NoError(t, err)

	// ValidateKey on revoked key should fail with ErrAPIKeyDisabled
	_, err = svc.ValidateKey(ctx, rawKey)
	assert.ErrorIs(t, err, ErrAPIKeyDisabled)

	// Delete — permanently removes
	err = svc.DeleteAPIKey(ctx, resp.APIKey.ID, 1)
	require.NoError(t, err)

	// Get after delete should fail
	_, err = svc.GetAPIKey(ctx, resp.APIKey.ID, 1)
	assert.ErrorIs(t, err, ErrAPIKeyNotFound)
}

func TestAPIKey_ValidationErrors(t *testing.T) {
	svc, ctx := setupIntegrationService(t)

	// Empty name
	_, err := svc.CreateAPIKey(ctx, &CreateAPIKeyRequest{
		OrganizationID: 1, CreatedBy: 1,
		Name: "", Scopes: []string{"pods:read"},
	})
	assert.ErrorIs(t, err, ErrNameEmpty)

	// Empty scopes
	_, err = svc.CreateAPIKey(ctx, &CreateAPIKeyRequest{
		OrganizationID: 1, CreatedBy: 1,
		Name: "no-scope", Scopes: []string{},
	})
	assert.ErrorIs(t, err, ErrScopesRequired)

	// Expired key should fail validation
	svcWithDB, db := newTestService(t)
	resp, _ := createExpiredAPIKey(t, svcWithDB, db, 1, "expired-key", []string{"pods:read"})
	_, err = svcWithDB.ValidateKey(ctx, resp.RawKey)
	assert.ErrorIs(t, err, ErrAPIKeyExpired)
}

package sso

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSSOService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewSSOConfigRepository(db)
	cfg := &config.Config{}
	svc := NewService(repo, testEncryptionKey, cfg)
	return svc, context.Background()
}

func TestSSO_CRUDConfig(t *testing.T) {
	svc, ctx := setupSSOService(t)

	// Create an OIDC SSO config
	created, err := svc.CreateConfig(ctx, &CreateConfigRequest{
		Domain:       "example.com",
		Name:         "Example OIDC",
		Protocol:     "oidc",
		IsEnabled:    true,
		OIDCIssuerURL:  "https://idp.example.com",
		OIDCClientID:   "client-123",
		OIDCClientSecret: "secret-456",
	}, 1)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "example.com", created.Domain)
	assert.Equal(t, "Example OIDC", created.Name)
	assert.True(t, created.IsEnabled)

	// GetConfig by ID
	fetched, err := svc.GetConfig(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "example.com", fetched.Domain)

	// ListConfigs
	configs, total, err := svc.ListConfigs(ctx, nil, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, configs, 1)
	assert.Equal(t, created.ID, configs[0].ID)

	// DeleteConfig
	err = svc.DeleteConfig(ctx, created.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = svc.GetConfig(ctx, created.ID)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestSSO_EnabledConfigs(t *testing.T) {
	svc, ctx := setupSSOService(t)

	// Create an enabled config
	_, err := svc.CreateConfig(ctx, &CreateConfigRequest{
		Domain:       "acme.org",
		Name:         "Acme OIDC",
		Protocol:     "oidc",
		IsEnabled:    true,
		OIDCIssuerURL:  "https://idp.acme.org",
		OIDCClientID:   "acme-client",
	}, 1)
	require.NoError(t, err)

	// Create a disabled config for a different domain
	_, err = svc.CreateConfig(ctx, &CreateConfigRequest{
		Domain:       "disabled.org",
		Name:         "Disabled OIDC",
		Protocol:     "oidc",
		IsEnabled:    false,
		OIDCIssuerURL:  "https://idp.disabled.org",
		OIDCClientID:   "disabled-client",
	}, 1)
	require.NoError(t, err)

	// GetEnabledConfigs for acme.org should return the enabled one
	enabled, err := svc.GetEnabledConfigs(ctx, "acme.org")
	require.NoError(t, err)
	assert.Len(t, enabled, 1)
	assert.Equal(t, "acme.org", enabled[0].Domain)

	// GetEnabledConfigs for disabled.org should return nothing
	enabled, err = svc.GetEnabledConfigs(ctx, "disabled.org")
	require.NoError(t, err)
	assert.Empty(t, enabled)
}

func TestSSO_EnforceSSO(t *testing.T) {
	svc, ctx := setupSSOService(t)

	// Create config with enforce_sso=true and is_enabled=true
	_, err := svc.CreateConfig(ctx, &CreateConfigRequest{
		Domain:       "strict.io",
		Name:         "Strict OIDC",
		Protocol:     "oidc",
		IsEnabled:    true,
		EnforceSSO:   true,
		OIDCIssuerURL:  "https://idp.strict.io",
		OIDCClientID:   "strict-client",
	}, 1)
	require.NoError(t, err)

	// HasEnforcedSSO should be true for strict.io
	enforced, err := svc.HasEnforcedSSO(ctx, "strict.io")
	require.NoError(t, err)
	assert.True(t, enforced)

	// HasEnforcedSSO should be false for non-existent domain
	enforced, err = svc.HasEnforcedSSO(ctx, "nope.io")
	require.NoError(t, err)
	assert.False(t, enforced)
}

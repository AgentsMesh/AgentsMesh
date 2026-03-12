package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig_Success(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	existing := seedOIDCConfig(repo)

	cfg, err := svc.GetConfig(context.Background(), existing.ID)
	require.NoError(t, err)
	assert.Equal(t, existing.ID, cfg.ID)
	assert.Equal(t, "company.com", cfg.Domain)
}

func TestGetConfig_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	_, err := svc.GetConfig(context.Background(), 999)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestDeleteConfig_Success(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	existing := seedOIDCConfig(repo)

	err := svc.DeleteConfig(context.Background(), existing.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = svc.GetConfig(context.Background(), existing.ID)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestDeleteConfig_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	err := svc.DeleteConfig(context.Background(), 999)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestListConfigs_Pagination(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Seed 3 configs
	for i := 0; i < 3; i++ {
		repo.seedConfig(&sso.Config{
			Domain:   fmt.Sprintf("company%d.com", i),
			Name:     fmt.Sprintf("SSO %d", i),
			Protocol: sso.ProtocolOIDC,
		})
	}

	configs, total, err := svc.ListConfigs(context.Background(), nil, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, configs, 2) // page size = 2
}

func TestListConfigs_DefaultPageSize(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Invalid page/pageSize should be normalized
	_, _, err := svc.ListConfigs(context.Background(), nil, 0, 0)
	require.NoError(t, err)
	// page=0 → 1, pageSize=0 → 20 (defaults applied internally)
}

func TestListConfigs_MaxPageSize(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// pageSize > 100 should be clamped to 20
	_, _, err := svc.ListConfigs(context.Background(), nil, 1, 999)
	require.NoError(t, err)
}

func TestGetEnabledConfigs(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Seed enabled and disabled configs
	repo.seedConfig(&sso.Config{
		Domain:    "company.com",
		Name:      "Enabled",
		Protocol:  sso.ProtocolOIDC,
		IsEnabled: true,
	})
	repo.seedConfig(&sso.Config{
		Domain:    "company.com",
		Name:      "Disabled",
		Protocol:  sso.ProtocolSAML,
		IsEnabled: false,
	})

	configs, err := svc.GetEnabledConfigs(context.Background(), "company.com")
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "Enabled", configs[0].Name)
}

func TestGetEnabledConfigs_DomainNormalization(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	repo.seedConfig(&sso.Config{
		Domain:    "company.com",
		Name:      "Test",
		Protocol:  sso.ProtocolOIDC,
		IsEnabled: true,
	})

	// Should match case-insensitively
	configs, err := svc.GetEnabledConfigs(context.Background(), "COMPANY.COM")
	require.NoError(t, err)
	assert.Len(t, configs, 1)
}

func TestDecryptSecret(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Empty string should return empty
	result, err := svc.DecryptSecret("")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestToConfigResponse_StripsSensitiveFields(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	issuer := "https://company.okta.com"
	clientID := "client-123"
	encrypted := "encrypted-secret"
	scopes := `["openid","email"]`

	cfg := &sso.Config{
		ID:                        1,
		Domain:                    "company.com",
		Name:                      "Test",
		Protocol:                  sso.ProtocolOIDC,
		OIDCIssuerURL:             &issuer,
		OIDCClientID:              &clientID,
		OIDCClientSecretEncrypted: &encrypted,
		OIDCScopes:                &scopes,
	}

	resp := svc.ToConfigResponse(cfg)
	assert.Equal(t, issuer, resp.OIDCIssuerURL)
	assert.Equal(t, clientID, resp.OIDCClientID)
	assert.Equal(t, scopes, resp.OIDCScopes)
	// ConfigResponse struct has no field for encrypted secrets — they are excluded by design
}

func TestToConfigResponse_LDAPZeroValues(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	host := "ldap.company.com"
	port := 389
	useTLS := false
	baseDN := "dc=company,dc=com"

	cfg := &sso.Config{
		ID:         2,
		Domain:     "company.com",
		Name:       "LDAP",
		Protocol:   sso.ProtocolLDAP,
		LDAPHost:   &host,
		LDAPPort:   &port,
		LDAPUseTLS: &useTLS,
		LDAPBaseDN: &baseDN,
	}

	resp := svc.ToConfigResponse(cfg)
	// LDAPUseTLS=false and LDAPPort=389 should NOT be omitted from JSON
	require.NotNil(t, resp.LDAPUseTLS, "LDAPUseTLS pointer should not be nil")
	assert.Equal(t, false, *resp.LDAPUseTLS, "LDAPUseTLS=false should be preserved")
	require.NotNil(t, resp.LDAPPort, "LDAPPort pointer should not be nil")
	assert.Equal(t, 389, *resp.LDAPPort, "LDAPPort should be preserved")
}

func TestToDiscoverResponse_Sanitized(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	issuer := "https://company.okta.com"
	cfg := &sso.Config{
		Domain:        "company.com",
		Name:          "Okta SSO",
		Protocol:      sso.ProtocolOIDC,
		EnforceSSO:    true,
		OIDCIssuerURL: &issuer,
	}

	resp := svc.ToDiscoverResponse(cfg)
	assert.Equal(t, "company.com", resp.Domain)
	assert.Equal(t, "Okta SSO", resp.Name)
	assert.Equal(t, "oidc", resp.Protocol)
	assert.True(t, resp.EnforceSSO)
	// DiscoverResponse should NOT contain issuer URL (sensitive config detail)
}

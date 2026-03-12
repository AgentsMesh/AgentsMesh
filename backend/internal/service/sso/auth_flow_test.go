package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// mockProvider implements ssoprovider.Provider for unit tests.
type mockProvider struct {
	getAuthURLResult     string
	getAuthURLErr        error
	handleCallbackResult *ssoprovider.UserInfo
	handleCallbackErr    error
	authenticateResult   *ssoprovider.UserInfo
	authenticateErr      error
}

func (m *mockProvider) GetAuthURL(_ context.Context, state string) (string, error) {
	if m.getAuthURLErr != nil {
		return "", m.getAuthURLErr
	}
	return m.getAuthURLResult + "?state=" + state, nil
}

func (m *mockProvider) HandleCallback(_ context.Context, _ map[string]string) (*ssoprovider.UserInfo, error) {
	return m.handleCallbackResult, m.handleCallbackErr
}

func (m *mockProvider) Authenticate(_ context.Context, _, _ string) (*ssoprovider.UserInfo, error) {
	return m.authenticateResult, m.authenticateErr
}

// newTestServiceWithMockProvider creates a service with a mock provider factory.
func newTestServiceWithMockProvider(repo *mockRepository, mp *mockProvider) *Service {
	svc := newTestService(repo)
	svc.providerFactory = func(_ context.Context, _ *sso.Config) (ssoprovider.Provider, error) {
		return mp, nil
	}
	return svc
}

// --- GetAuthURL tests ---

func TestGetAuthURL_OIDC_Success(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{getAuthURLResult: "https://idp.example.com/auth"}
	svc := newTestServiceWithMockProvider(repo, mp)
	seedOIDCConfig(repo)

	authURL, err := svc.GetAuthURL(context.Background(), "company.com", sso.ProtocolOIDC, "test-state")
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://idp.example.com/auth")
	assert.Contains(t, authURL, "test-state")
}

func TestGetAuthURL_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	_, err := svc.GetAuthURL(context.Background(), "nonexistent.com", sso.ProtocolOIDC, "state")
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestGetAuthURL_Disabled(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	cfg := seedOIDCConfig(repo)
	// Disable the config directly in the mock store
	repo.mu.Lock()
	repo.configs[cfg.ID].IsEnabled = false
	repo.mu.Unlock()

	_, err := svc.GetAuthURL(context.Background(), "company.com", sso.ProtocolOIDC, "state")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestGetAuthURL_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByDomainErr = fmt.Errorf("database connection lost")
	svc := newTestService(repo)

	_, err := svc.GetAuthURL(context.Background(), "company.com", sso.ProtocolOIDC, "state")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query SSO config")
}

func TestGetAuthURL_DomainNormalization(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{getAuthURLResult: "https://idp.example.com/auth"}
	svc := newTestServiceWithMockProvider(repo, mp)
	seedOIDCConfig(repo) // domain = "company.com"

	// Should find config even with uppercase domain
	authURL, err := svc.GetAuthURL(context.Background(), "COMPANY.COM", sso.ProtocolOIDC, "state")
	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
}

// --- HandleCallback tests ---

func TestHandleCallback_OIDC_Success(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{
		handleCallbackResult: &ssoprovider.UserInfo{
			ExternalID: "user-123",
			Email:      "user@company.com",
			Username:   "user",
			Name:       "Test User",
		},
	}
	svc := newTestServiceWithMockProvider(repo, mp)
	existing := seedOIDCConfig(repo)

	userInfo, configID, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolOIDC, map[string]string{"code": "auth-code"})
	require.NoError(t, err)
	assert.Equal(t, "user@company.com", userInfo.Email)
	assert.Equal(t, existing.ID, configID)
}

func TestHandleCallback_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	_, _, err := svc.HandleCallback(context.Background(), "nonexistent.com", sso.ProtocolOIDC, nil)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestHandleCallback_Disabled(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	cfg := seedOIDCConfig(repo)
	repo.mu.Lock()
	repo.configs[cfg.ID].IsEnabled = false
	repo.mu.Unlock()

	_, _, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolOIDC, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestHandleCallback_ProviderError(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{handleCallbackErr: fmt.Errorf("invalid token")}
	svc := newTestServiceWithMockProvider(repo, mp)
	seedOIDCConfig(repo)

	_, _, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolOIDC, map[string]string{"code": "bad-code"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSO callback failed")
}

func TestHandleCallback_NilUserInfo(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{handleCallbackResult: nil, handleCallbackErr: nil}
	svc := newTestServiceWithMockProvider(repo, mp)
	seedOIDCConfig(repo)

	_, _, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolOIDC, map[string]string{"code": "code"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no user info")
}

func TestHandleCallback_SAML_WithRelayState(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{
		handleCallbackResult: &ssoprovider.UserInfo{
			ExternalID: "saml-user-1",
			Email:      "user@company.com",
		},
	}
	svc := newTestServiceWithMockProvider(repo, mp)
	// No Redis → storeSAMLRequestID is a no-op, but RelayState handling still runs
	seedSAMLConfig(repo)

	params := map[string]string{
		"SAMLResponse": "base64-encoded-response",
		"RelayState":   "relay-state-value",
	}
	userInfo, _, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolSAML, params)
	require.NoError(t, err)
	assert.Equal(t, "saml-user-1", userInfo.ExternalID)
}

func TestHandleCallback_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByDomainErr = fmt.Errorf("connection timeout")
	svc := newTestService(repo)

	_, _, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolOIDC, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query SSO config")
}

// --- AuthenticateLDAP tests ---

func TestAuthenticateLDAP_Success(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{
		authenticateResult: &ssoprovider.UserInfo{
			ExternalID: "cn=user,dc=company,dc=com",
			Email:      "user@company.com",
			Username:   "user",
		},
	}
	svc := newTestServiceWithMockProvider(repo, mp)
	existing := seedLDAPConfig(repo)

	userInfo, configID, err := svc.AuthenticateLDAP(context.Background(), "company.com", "user", "pass")
	require.NoError(t, err)
	assert.Equal(t, "user@company.com", userInfo.Email)
	assert.Equal(t, existing.ID, configID)
}

func TestAuthenticateLDAP_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	_, _, err := svc.AuthenticateLDAP(context.Background(), "nonexistent.com", "user", "pass")
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestAuthenticateLDAP_Disabled(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	cfg := seedLDAPConfig(repo)
	repo.mu.Lock()
	repo.configs[cfg.ID].IsEnabled = false
	repo.mu.Unlock()

	_, _, err := svc.AuthenticateLDAP(context.Background(), "company.com", "user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestAuthenticateLDAP_AuthError(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{authenticateErr: fmt.Errorf("invalid credentials")}
	svc := newTestServiceWithMockProvider(repo, mp)
	seedLDAPConfig(repo)

	_, _, err := svc.AuthenticateLDAP(context.Background(), "company.com", "user", "bad-pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LDAP authentication failed")
}

func TestAuthenticateLDAP_NilUserInfo(t *testing.T) {
	repo := newMockRepository()
	mp := &mockProvider{authenticateResult: nil, authenticateErr: nil}
	svc := newTestServiceWithMockProvider(repo, mp)
	seedLDAPConfig(repo)

	_, _, err := svc.AuthenticateLDAP(context.Background(), "company.com", "user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no user info")
}

func TestAuthenticateLDAP_BuildProviderError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	svc.providerFactory = func(_ context.Context, _ *sso.Config) (ssoprovider.Provider, error) {
		return nil, fmt.Errorf("build failed")
	}
	seedLDAPConfig(repo)

	_, _, err := svc.AuthenticateLDAP(context.Background(), "company.com", "user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build LDAP provider")
}

func TestAuthenticateLDAP_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByDomainErr = fmt.Errorf("db error")
	svc := newTestService(repo)

	_, _, err := svc.AuthenticateLDAP(context.Background(), "company.com", "user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query SSO config")
}

// --- TestConnection tests ---

func TestTestConnection_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	err := svc.TestConnection(context.Background(), 999)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestTestConnection_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByIDErr = fmt.Errorf("db error")
	svc := newTestService(repo)

	err := svc.TestConnection(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query SSO config")
}

func TestTestConnection_InvalidProtocol(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	repo.seedConfig(&sso.Config{
		Domain:   "bad.com",
		Protocol: "kerberos",
	})

	err := svc.TestConnection(context.Background(), 1)
	assert.ErrorIs(t, err, ErrInvalidProtocol)
}

func TestTestConnection_SAML_WithFactory(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	seedSAMLConfig(repo)

	// Mock samlProviderFactory to return a provider that passes ValidateConfig.
	// We can't easily create a real SAMLProvider without valid metadata, so
	// we test the SAML path by checking it calls buildSAMLProvider then ValidateConfig.
	svc.samlProviderFactory = func(_ *sso.Config) (*ssoprovider.SAMLProvider, error) {
		// Return an error to verify the path is exercised
		return nil, fmt.Errorf("saml provider build error")
	}

	err := svc.TestConnection(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saml provider build error")
}

func TestTestConnection_LDAP_BuildSuccess_ConnectFails(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	seedLDAPConfig(repo)

	// buildLDAPProvider succeeds (no network), but provider.TestConnection
	// fails because no LDAP server is running.
	err := svc.TestConnection(context.Background(), 1)
	require.Error(t, err)
	// The error should come from the LDAP connection attempt
	assert.Contains(t, err.Error(), "connection failed")
}

// --- GetSAMLMetadata tests ---

func TestGetSAMLMetadata_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	_, err := svc.GetSAMLMetadata(context.Background(), "nonexistent.com")
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestGetSAMLMetadata_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByDomainErr = fmt.Errorf("db error")
	svc := newTestService(repo)

	_, err := svc.GetSAMLMetadata(context.Background(), "company.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query SSO config")
}

func TestGetSAMLMetadata_BuildProviderError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	seedSAMLConfig(repo)

	svc.samlProviderFactory = func(_ *sso.Config) (*ssoprovider.SAMLProvider, error) {
		return nil, fmt.Errorf("invalid SAML config")
	}

	_, err := svc.GetSAMLMetadata(context.Background(), "company.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build SAML provider")
}

// --- storeSAMLRequestID / retrieveSAMLRequestID ---

func TestStoreSAMLRequestID_NilRedis(t *testing.T) {
	svc := newTestService(newMockRepository())
	// redis is nil by default
	err := svc.storeSAMLRequestID(context.Background(), "state-1", "req-id-1")
	assert.NoError(t, err, "should gracefully degrade when Redis is nil")
}

func TestRetrieveSAMLRequestID_NilRedis(t *testing.T) {
	svc := newTestService(newMockRepository())
	result, err := svc.retrieveSAMLRequestID(context.Background(), "state-1")
	assert.NoError(t, err, "should gracefully degrade when Redis is nil")
	assert.Equal(t, "", result)
}

// --- HasEnforcedSSO ---

func TestHasEnforcedSSO_True(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOVal = true
	svc := newTestService(repo)

	enforced, err := svc.HasEnforcedSSO(context.Background(), "COMPANY.COM")
	require.NoError(t, err)
	assert.True(t, enforced)
}

func TestHasEnforcedSSO_False(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOVal = false
	svc := newTestService(repo)

	enforced, err := svc.HasEnforcedSSO(context.Background(), "company.com")
	require.NoError(t, err)
	assert.False(t, enforced)
}

func TestHasEnforcedSSO_Error(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOErr = fmt.Errorf("db error")
	svc := newTestService(repo)

	_, err := svc.HasEnforcedSSO(context.Background(), "company.com")
	require.Error(t, err)
}

// --- GetConfig additional ---

func TestGetConfig_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByIDErr = gorm.ErrRecordNotFound
	svc := newTestService(repo)

	_, err := svc.GetConfig(context.Background(), 1)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestGetConfig_GenericError(t *testing.T) {
	repo := newMockRepository()
	repo.getByIDErr = fmt.Errorf("connection refused")
	svc := newTestService(repo)

	_, err := svc.GetConfig(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get SSO config")
}

// --- DeleteConfig additional ---

func TestDeleteConfig_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.deleteErr = fmt.Errorf("disk full")
	svc := newTestService(repo)
	seedOIDCConfig(repo)

	err := svc.DeleteConfig(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete SSO config")
}

// --- NewServiceWithRedis ---

func TestNewServiceWithRedis(t *testing.T) {
	repo := newMockRepository()
	svc := NewServiceWithRedis(repo, "key", nil, nil)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.redis) // nil redis client passed
}

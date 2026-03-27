package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

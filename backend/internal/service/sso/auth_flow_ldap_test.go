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

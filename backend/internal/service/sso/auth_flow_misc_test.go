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

	svc.samlProviderFactory = func(_ *sso.Config) (*ssoprovider.SAMLProvider, error) {
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

	err := svc.TestConnection(context.Background(), 1)
	require.Error(t, err)
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

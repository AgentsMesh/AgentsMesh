package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- setLDAPFields: all optional field paths ---

func TestCreateConfig_LDAP_AllOptionalFields(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:           "company.com",
		Name:             "LDAP Full",
		Protocol:         "ldap",
		LDAPHost:         "ldap.company.com",
		LDAPPort:         636,
		LDAPUseTLS:       true,
		LDAPBindDN:       "cn=admin,dc=company,dc=com",
		LDAPBindPassword: "secret",
		LDAPBaseDN:       "dc=company,dc=com",
		LDAPUserFilter:   "(sAMAccountName={{username}})",
		LDAPEmailAttr:    "userPrincipalName",
		LDAPNameAttr:     "displayName",
		LDAPUsernameAttr: "sAMAccountName",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.LDAPHost)
	assert.NotNil(t, cfg.LDAPPort)
	assert.Equal(t, 636, *cfg.LDAPPort)
	assert.NotNil(t, cfg.LDAPUseTLS)
	assert.True(t, *cfg.LDAPUseTLS)
	assert.NotNil(t, cfg.LDAPBindDN)
	assert.NotNil(t, cfg.LDAPBindPasswordEncrypted)
	assert.NotNil(t, cfg.LDAPBaseDN)
	assert.NotNil(t, cfg.LDAPUserFilter)
	assert.Equal(t, "(sAMAccountName={{username}})", *cfg.LDAPUserFilter)
	assert.NotNil(t, cfg.LDAPEmailAttr)
	assert.Equal(t, "userPrincipalName", *cfg.LDAPEmailAttr)
	assert.NotNil(t, cfg.LDAPNameAttr)
	assert.Equal(t, "displayName", *cfg.LDAPNameAttr)
	assert.NotNil(t, cfg.LDAPUsernameAttr)
	assert.Equal(t, "sAMAccountName", *cfg.LDAPUsernameAttr)
}

// --- setOIDCFields: scopes optional field ---

func TestCreateConfig_OIDC_WithScopes(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "company.com",
		Name:          "OIDC With Scopes",
		Protocol:      "oidc",
		OIDCIssuerURL: "https://company.okta.com",
		OIDCClientID:  "client-123",
		OIDCScopes:    `["openid","email","profile","groups"]`,
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.OIDCScopes)
	assert.Equal(t, `["openid","email","profile","groups"]`, *cfg.OIDCScopes)
}

// --- GetSAMLMetadata: success path ---

func TestGetSAMLMetadata_Success(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	repo := newMockRepository()
	svc := newTestService(repo)

	ssoURL := "https://idp.example.com/sso"
	encryptedCert, err := crypto.EncryptWithKey(certPEM, testEncryptionKey)
	require.NoError(t, err)
	spEntityID := "http://localhost/api/v1/auth/sso/company.com/saml/metadata"

	repo.seedConfig(&sso.Config{
		Domain:               "company.com",
		Protocol:             sso.ProtocolSAML,
		IsEnabled:            true,
		SAMLIDPSSOURL:        &ssoURL,
		SAMLIDPCertEncrypted: &encryptedCert,
		SAMLSPEntityID:       &spEntityID,
	})

	metadata, err := svc.GetSAMLMetadata(context.Background(), "company.com")
	require.NoError(t, err)
	assert.Contains(t, string(metadata), "EntityDescriptor")
	assert.Contains(t, string(metadata), spEntityID)
}

// --- HandleCallback: build provider error ---

func TestHandleCallback_BuildProviderError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	svc.providerFactory = func(_ context.Context, _ *sso.Config) (ssoprovider.Provider, error) {
		return nil, fmt.Errorf("provider init failed")
	}
	seedOIDCConfig(repo)

	_, _, err := svc.HandleCallback(context.Background(), "company.com", sso.ProtocolOIDC, map[string]string{"code": "abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build SSO provider")
}

// --- CreateConfig: repo create error ---

func TestCreateConfig_RepoCreateError(t *testing.T) {
	repo := newMockRepository()
	repo.createErr = fmt.Errorf("disk full")
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "company.com",
		Name:          "Test",
		Protocol:      "oidc",
		OIDCIssuerURL: "https://issuer.com",
		OIDCClientID:  "client-id",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create SSO config")
}

// --- CreateConfig: repo query error during duplicate check ---

func TestCreateConfig_DuplicateCheckRepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getByDomainErr = fmt.Errorf("connection refused")
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "company.com",
		Name:          "Test",
		Protocol:      "oidc",
		OIDCIssuerURL: "https://issuer.com",
		OIDCClientID:  "client-id",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check duplicate")
}

// --- UpdateConfig: repo.Update returns not-found error ---

func TestUpdateConfig_RepoUpdateNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	seedOIDCConfig(repo)

	// Simulate: config exists for GetConfig but Update returns not-found (race condition)
	repo.updateErr = fmt.Errorf("record not found")

	_, err := svc.UpdateConfig(context.Background(), 1, &UpdateConfigRequest{Name: ptr("New")})
	require.Error(t, err)
}

// --- ListConfigs: repo error ---

func TestListConfigs_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.listErr = fmt.Errorf("query timeout")
	svc := newTestService(repo)

	_, _, err := svc.ListConfigs(context.Background(), nil, 1, 20)
	require.Error(t, err)
}

// --- GetEnabledConfigs: repo error ---

func TestGetEnabledConfigs_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getEnabledErr = fmt.Errorf("connection lost")
	svc := newTestService(repo)

	_, err := svc.GetEnabledConfigs(context.Background(), "company.com")
	require.Error(t, err)
}

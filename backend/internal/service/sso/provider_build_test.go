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

// --- buildProvider dispatch ---

func TestBuildProvider_InvalidProtocol(t *testing.T) {
	svc := newTestService(newMockRepository())
	cfg := &sso.Config{Protocol: "kerberos"}

	_, err := svc.buildProvider(context.Background(), cfg)
	assert.ErrorIs(t, err, ErrInvalidProtocol)
}

func TestBuildProvider_LDAP_Dispatch(t *testing.T) {
	svc := newTestService(newMockRepository())
	host := "ldap.test.com"
	port := 389
	baseDN := "dc=test,dc=com"
	cfg := &sso.Config{
		Protocol:   sso.ProtocolLDAP,
		LDAPHost:   &host,
		LDAPPort:   &port,
		LDAPBaseDN: &baseDN,
	}

	provider, err := svc.buildProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildProvider_UsesFactory(t *testing.T) {
	svc := newTestService(newMockRepository())
	called := false
	svc.providerFactory = func(_ context.Context, _ *sso.Config) (ssoprovider.Provider, error) {
		called = true
		return &mockProvider{}, nil
	}

	_, err := svc.buildProvider(context.Background(), &sso.Config{Protocol: sso.ProtocolOIDC})
	require.NoError(t, err)
	assert.True(t, called)
}

// --- buildLDAPProvider ---

func TestBuildLDAPProvider_FullConfig(t *testing.T) {
	svc := newTestService(newMockRepository())

	host := "ldap.company.com"
	port := 636
	useTLS := true
	bindDN := "cn=admin,dc=company,dc=com"
	baseDN := "dc=company,dc=com"
	userFilter := "(sAMAccountName={{username}})"
	emailAttr := "userPrincipalName"
	nameAttr := "displayName"
	usernameAttr := "sAMAccountName"

	encrypted, err := crypto.EncryptWithKey("bind-password", testEncryptionKey)
	require.NoError(t, err)

	cfg := &sso.Config{
		Protocol:                  sso.ProtocolLDAP,
		LDAPHost:                  &host,
		LDAPPort:                  &port,
		LDAPUseTLS:                &useTLS,
		LDAPBindDN:                &bindDN,
		LDAPBindPasswordEncrypted: &encrypted,
		LDAPBaseDN:                &baseDN,
		LDAPUserFilter:            &userFilter,
		LDAPEmailAttr:             &emailAttr,
		LDAPNameAttr:              &nameAttr,
		LDAPUsernameAttr:          &usernameAttr,
	}

	provider, err := svc.buildLDAPProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildLDAPProvider_Minimal(t *testing.T) {
	svc := newTestService(newMockRepository())
	host := "ldap.test.com"
	baseDN := "dc=test,dc=com"

	cfg := &sso.Config{
		Protocol:   sso.ProtocolLDAP,
		LDAPHost:   &host,
		LDAPBaseDN: &baseDN,
	}

	provider, err := svc.buildLDAPProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildLDAPProvider_DecryptionError(t *testing.T) {
	svc := newTestService(newMockRepository())
	host := "ldap.test.com"
	baseDN := "dc=test,dc=com"
	badEncrypted := "not-a-valid-encrypted-string"

	cfg := &sso.Config{
		Protocol:                  sso.ProtocolLDAP,
		LDAPHost:                  &host,
		LDAPBaseDN:                &baseDN,
		LDAPBindPasswordEncrypted: &badEncrypted,
	}

	_, err := svc.buildLDAPProvider(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt bind password")
}

func TestBuildLDAPProvider_MissingHost(t *testing.T) {
	svc := newTestService(newMockRepository())
	baseDN := "dc=test,dc=com"

	cfg := &sso.Config{
		Protocol:   sso.ProtocolLDAP,
		LDAPBaseDN: &baseDN,
	}

	_, err := svc.buildLDAPProvider(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing LDAP host")
}

// --- buildSAMLProvider ---

func TestBuildSAMLProvider_UsesFactory(t *testing.T) {
	svc := newTestService(newMockRepository())
	called := false
	svc.samlProviderFactory = func(_ *sso.Config) (*ssoprovider.SAMLProvider, error) {
		called = true
		return nil, fmt.Errorf("mock error")
	}

	_, err := svc.buildSAMLProvider(&sso.Config{Domain: "test.com"})
	require.Error(t, err)
	assert.True(t, called)
}

func TestBuildSAMLProvider_DecryptionError(t *testing.T) {
	svc := newTestService(newMockRepository())
	badEncrypted := "not-valid"
	cfg := &sso.Config{
		Domain:               "test.com",
		SAMLIDPCertEncrypted: &badEncrypted,
	}

	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt IdP cert")
}

func TestBuildSAMLProvider_WithMetadataURL(t *testing.T) {
	svc := newTestService(newMockRepository())
	metadataURL := "https://nonexistent.example.com/metadata"
	cfg := &sso.Config{
		Domain:             "test.com",
		SAMLIDPMetadataURL: &metadataURL,
	}

	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestBuildSAMLProvider_MissingIDPSource(t *testing.T) {
	svc := newTestService(newMockRepository())
	cfg := &sso.Config{Domain: "test.com"}

	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
}

func TestBuildSAMLProvider_CustomEntityIDAndNameIDFormat(t *testing.T) {
	svc := newTestService(newMockRepository())
	metadataURL := "https://nonexistent.example.com/metadata"
	customEntityID := "https://custom-entity-id.example.com"
	customNameIDFormat := "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
	cfg := &sso.Config{
		Domain:             "test.com",
		SAMLIDPMetadataURL: &metadataURL,
		SAMLSPEntityID:     &customEntityID,
		SAMLNameIDFormat:   &customNameIDFormat,
	}

	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
}

// --- testSAMLConnection ---

func TestTestSAMLConnection_ProviderBuildError(t *testing.T) {
	svc := newTestService(newMockRepository())
	svc.samlProviderFactory = func(_ *sso.Config) (*ssoprovider.SAMLProvider, error) {
		return nil, fmt.Errorf("invalid SAML config")
	}

	err := svc.testSAMLConnection(&sso.Config{})
	require.Error(t, err)
}

// --- testLDAPConnection ---

func TestTestLDAPConnection_BuildError(t *testing.T) {
	svc := newTestService(newMockRepository())
	badEncrypted := "not-valid"
	cfg := &sso.Config{
		Protocol:                  sso.ProtocolLDAP,
		LDAPBindPasswordEncrypted: &badEncrypted,
	}

	err := svc.testLDAPConnection(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt")
}

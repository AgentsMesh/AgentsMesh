package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	// We can't easily test metadata URL fetch in unit tests (HTTP call),
	// but we can test with inline metadata XML or cert+SSO URL.
	// This test verifies the field mapping path for metadata URL
	// (which will fail because the URL is not reachable).
	svc := newTestService(newMockRepository())
	metadataURL := "https://nonexistent.example.com/metadata"
	cfg := &sso.Config{
		Domain:             "test.com",
		SAMLIDPMetadataURL: &metadataURL,
	}

	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
	// Error from trying to fetch metadata URL
	assert.Contains(t, err.Error(), "failed")
}

func TestBuildSAMLProvider_MissingIDPSource(t *testing.T) {
	svc := newTestService(newMockRepository())
	cfg := &sso.Config{
		Domain: "test.com",
	}

	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
	// NewSAMLProvider returns error for missing IdP metadata
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

	// Will fail on metadata URL fetch, but exercises the entity ID and name ID format paths
	_, err := svc.buildSAMLProvider(cfg)
	require.Error(t, err)
}

// --- buildOIDCProvider (with httptest) ---

func newFakeOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srvURL,
			"authorization_endpoint": srvURL + "/auth",
			"token_endpoint":         srvURL + "/token",
			"jwks_uri":               srvURL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	})
	srv := httptest.NewServer(mux)
	srvURL = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestBuildOIDCProvider_Success(t *testing.T) {
	srv := newFakeOIDCServer(t)
	svc := newTestService(newMockRepository())

	issuerURL := srv.URL
	clientID := "test-client"
	scopes := `["openid","email"]`
	secret := "my-secret"
	encrypted, err := crypto.EncryptWithKey(secret, testEncryptionKey)
	require.NoError(t, err)

	cfg := &sso.Config{
		Domain:                    "test.com",
		Protocol:                  sso.ProtocolOIDC,
		OIDCIssuerURL:             &issuerURL,
		OIDCClientID:              &clientID,
		OIDCClientSecretEncrypted: &encrypted,
		OIDCScopes:                &scopes,
	}

	provider, err := svc.buildOIDCProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildOIDCProvider_NilFields(t *testing.T) {
	srv := newFakeOIDCServer(t)
	svc := newTestService(newMockRepository())

	issuerURL := srv.URL
	clientID := "test-client"
	cfg := &sso.Config{
		Protocol:      sso.ProtocolOIDC,
		OIDCIssuerURL: &issuerURL,
		OIDCClientID:  &clientID,
		// No secret, no scopes — should use defaults
	}

	provider, err := svc.buildOIDCProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildOIDCProvider_AllNilFields(t *testing.T) {
	svc := newTestService(newMockRepository())
	cfg := &sso.Config{
		Protocol: sso.ProtocolOIDC,
		// All OIDC fields nil
	}

	// Should fail because issuer URL is empty
	_, err := svc.buildOIDCProvider(context.Background(), cfg)
	require.Error(t, err)
}

func TestBuildOIDCProvider_DecryptionError(t *testing.T) {
	srv := newFakeOIDCServer(t)
	svc := newTestService(newMockRepository())

	issuerURL := srv.URL
	clientID := "test-client"
	badEncrypted := "not-a-valid-encrypted-string"
	cfg := &sso.Config{
		Protocol:                  sso.ProtocolOIDC,
		OIDCIssuerURL:             &issuerURL,
		OIDCClientID:              &clientID,
		OIDCClientSecretEncrypted: &badEncrypted,
	}

	_, err := svc.buildOIDCProvider(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt client secret")
}

func TestBuildOIDCProvider_ScopesParsing(t *testing.T) {
	tests := []struct {
		name   string
		scopes string
	}{
		{"json_array", `["openid","email","profile"]`},
		{"space_separated", "openid email profile"},
		{"comma_separated", "openid,email,profile"},
		{"invalid_json_fallback_space", "{bad json} openid email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeOIDCServer(t)
			svc := newTestService(newMockRepository())

			issuerURL := srv.URL
			clientID := "test-client"
			cfg := &sso.Config{
				Protocol:      sso.ProtocolOIDC,
				OIDCIssuerURL: &issuerURL,
				OIDCClientID:  &clientID,
				OIDCScopes:    &tt.scopes,
			}

			provider, err := svc.buildOIDCProvider(context.Background(), cfg)
			require.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestBuildOIDCProvider_EmptyScopes(t *testing.T) {
	srv := newFakeOIDCServer(t)
	svc := newTestService(newMockRepository())

	issuerURL := srv.URL
	clientID := "test-client"
	emptyScopes := ""
	cfg := &sso.Config{
		Protocol:      sso.ProtocolOIDC,
		OIDCIssuerURL: &issuerURL,
		OIDCClientID:  &clientID,
		OIDCScopes:    &emptyScopes,
	}

	provider, err := svc.buildOIDCProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

// --- testOIDCConnection ---

func TestTestOIDCConnection_Success(t *testing.T) {
	srv := newFakeOIDCServer(t)
	svc := newTestService(newMockRepository())

	issuerURL := srv.URL
	clientID := "test-client"
	cfg := &sso.Config{
		Protocol:      sso.ProtocolOIDC,
		OIDCIssuerURL: &issuerURL,
		OIDCClientID:  &clientID,
	}

	err := svc.testOIDCConnection(context.Background(), cfg)
	assert.NoError(t, err)
}

func TestTestOIDCConnection_InvalidIssuer(t *testing.T) {
	svc := newTestService(newMockRepository())
	issuerURL := "https://nonexistent.invalid"
	clientID := "test-client"
	cfg := &sso.Config{
		Protocol:      sso.ProtocolOIDC,
		OIDCIssuerURL: &issuerURL,
		OIDCClientID:  &clientID,
	}

	err := svc.testOIDCConnection(context.Background(), cfg)
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

// --- TestConnection integration via dispatch ---

func TestTestConnection_OIDC_ViaDispatch(t *testing.T) {
	srv := newFakeOIDCServer(t)
	repo := newMockRepository()
	svc := newTestService(repo)

	issuerURL := srv.URL
	clientID := "test-client"
	repo.seedConfig(&sso.Config{
		Domain:        "test.com",
		Protocol:      sso.ProtocolOIDC,
		IsEnabled:     true,
		OIDCIssuerURL: &issuerURL,
		OIDCClientID:  &clientID,
	})

	err := svc.TestConnection(context.Background(), 1)
	assert.NoError(t, err)
}

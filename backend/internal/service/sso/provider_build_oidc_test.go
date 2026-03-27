package sso

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeOIDCServer creates a minimal OIDC discovery endpoint for testing.
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

// --- buildOIDCProvider (with httptest) ---

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
	}

	provider, err := svc.buildOIDCProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildOIDCProvider_AllNilFields(t *testing.T) {
	svc := newTestService(newMockRepository())
	cfg := &sso.Config{Protocol: sso.ProtocolOIDC}

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

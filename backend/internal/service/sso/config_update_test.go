package sso

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ptr returns a pointer to v (test helper for pointer fields in UpdateConfigRequest).
func ptr[T any](v T) *T { return &v }

func seedOIDCConfig(repo *mockRepository) *sso.Config {
	issuer := "https://company.okta.com"
	clientID := "client-123"
	cfg := &sso.Config{
		Domain:       "company.com",
		Name:         "Okta SSO",
		Protocol:     sso.ProtocolOIDC,
		IsEnabled:    true,
		OIDCIssuerURL: &issuer,
		OIDCClientID:  &clientID,
	}
	repo.seedConfig(cfg)
	return cfg
}

func seedSAMLConfig(repo *mockRepository) *sso.Config {
	metadataURL := "https://idp.company.com/metadata"
	spEntityID := "http://localhost/api/v1/auth/sso/company.com/saml/metadata"
	cfg := &sso.Config{
		Domain:             "company.com",
		Name:               "SAML SSO",
		Protocol:           sso.ProtocolSAML,
		IsEnabled:          true,
		SAMLIDPMetadataURL: &metadataURL,
		SAMLSPEntityID:     &spEntityID,
	}
	repo.seedConfig(cfg)
	return cfg
}

func seedLDAPConfig(repo *mockRepository) *sso.Config {
	host := "ldap.company.com"
	port := 389
	baseDN := "dc=company,dc=com"
	useTLS := false
	cfg := &sso.Config{
		Domain:     "company.com",
		Name:       "LDAP Auth",
		Protocol:   sso.ProtocolLDAP,
		IsEnabled:  true,
		LDAPHost:   &host,
		LDAPPort:   &port,
		LDAPBaseDN: &baseDN,
		LDAPUseTLS: &useTLS,
	}
	repo.seedConfig(cfg)
	return cfg
}

func TestUpdateConfig_CommonFields(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	existing := seedOIDCConfig(repo)

	newName := "Updated SSO"
	req := &UpdateConfigRequest{
		Name: &newName,
	}

	cfg, err := svc.UpdateConfig(context.Background(), existing.ID, req)
	require.NoError(t, err)
	// The mock doesn't apply updates, so we verify the returned config (re-fetched)
	assert.NotNil(t, cfg)
}

func TestUpdateConfig_EmptyRequest(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	existing := seedOIDCConfig(repo)

	req := &UpdateConfigRequest{} // no fields set
	cfg, err := svc.UpdateConfig(context.Background(), existing.ID, req)
	require.NoError(t, err)
	// Should return existing config unchanged (short-circuit on empty updates)
	assert.Equal(t, existing.ID, cfg.ID)
}

func TestUpdateConfig_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &UpdateConfigRequest{Name: ptr("test")}
	_, err := svc.UpdateConfig(context.Background(), 999, req)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestUpdateConfig_CrossProtocolFieldsSilentlyStripped(t *testing.T) {
	// Cross-protocol fields are unconditionally stripped by stripCrossProtocolEmptyFields,
	// so sending them should succeed (fields ignored, not rejected).
	tests := []struct {
		name string
		seed func(*mockRepository) *sso.Config
		req  *UpdateConfigRequest
	}{
		{
			name: "SAML fields on OIDC config",
			seed: seedOIDCConfig,
			req:  &UpdateConfigRequest{SAMLIDPMetadataURL: ptr("https://idp.com/metadata")},
		},
		{
			name: "LDAP fields on OIDC config",
			seed: seedOIDCConfig,
			req:  &UpdateConfigRequest{LDAPHost: ptr("ldap.company.com")},
		},
		{
			name: "OIDC fields on SAML config",
			seed: seedSAMLConfig,
			req:  &UpdateConfigRequest{OIDCIssuerURL: ptr("https://issuer.com")},
		},
		{
			name: "OIDC fields on LDAP config",
			seed: seedLDAPConfig,
			req:  &UpdateConfigRequest{OIDCClientID: ptr("client-id")},
		},
		{
			name: "SAML fields on LDAP config",
			seed: seedLDAPConfig,
			req:  &UpdateConfigRequest{SAMLIDPSSOURL: ptr("https://sso.com")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := newTestService(repo)
			existing := tt.seed(repo)

			cfg, err := svc.UpdateConfig(context.Background(), existing.ID, tt.req)
			require.NoError(t, err, "cross-protocol fields should be silently stripped, not rejected")
			assert.Equal(t, existing.ID, cfg.ID)
		})
	}
}

func TestUpdateConfig_ClearRequiredFieldRejected(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(*mockRepository) *sso.Config
		req     *UpdateConfigRequest
		wantErr string
	}{
		{
			name:    "clear OIDC issuer URL",
			seed:    seedOIDCConfig,
			req:     &UpdateConfigRequest{OIDCIssuerURL: ptr("")},
			wantErr: "OIDC issuer URL cannot be empty",
		},
		{
			name:    "clear OIDC client ID",
			seed:    seedOIDCConfig,
			req:     &UpdateConfigRequest{OIDCClientID: ptr("")},
			wantErr: "OIDC client ID cannot be empty",
		},
		{
			name:    "clear LDAP host",
			seed:    seedLDAPConfig,
			req:     &UpdateConfigRequest{LDAPHost: ptr("")},
			wantErr: "LDAP host cannot be empty",
		},
		{
			name:    "clear LDAP base DN",
			seed:    seedLDAPConfig,
			req:     &UpdateConfigRequest{LDAPBaseDN: ptr("")},
			wantErr: "LDAP base DN cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := newTestService(repo)
			existing := tt.seed(repo)

			_, err := svc.UpdateConfig(context.Background(), existing.ID, tt.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestUpdateConfig_SAML_ClearAllIdPSources(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	existing := seedSAMLConfig(repo)

	// Clear the only IdP source (metadata URL)
	req := &UpdateConfigRequest{SAMLIDPMetadataURL: ptr("")}
	_, err := svc.UpdateConfig(context.Background(), existing.ID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SAML requires at least one IdP source")
}

func TestBuildUpdateMap_OIDCSecretEncryption(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &UpdateConfigRequest{
		OIDCClientSecret: ptr("new-secret"),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	encrypted, ok := updates["oidc_client_secret_encrypted"]
	assert.True(t, ok, "should contain encrypted secret")
	assert.NotEqual(t, "new-secret", encrypted, "should not be plaintext")
}

func TestBuildUpdateMap_OIDCSecretClearing(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &UpdateConfigRequest{
		OIDCClientSecret: ptr(""),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	val, ok := updates["oidc_client_secret_encrypted"]
	assert.True(t, ok, "should contain key for clearing")
	assert.Nil(t, val, "empty secret should clear the encrypted field")
}

func TestBuildUpdateMap_SAMLCertClearing(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &UpdateConfigRequest{
		SAMLIDPCert: ptr(""),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	val, ok := updates["saml_idp_cert_encrypted"]
	assert.True(t, ok, "should contain key for clearing")
	assert.Nil(t, val, "empty cert should clear the encrypted field")
}

func TestBuildUpdateMap_LDAPPasswordClearing(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &UpdateConfigRequest{
		LDAPBindPassword: ptr(""),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	val, ok := updates["ldap_bind_password_encrypted"]
	assert.True(t, ok, "should contain key for clearing")
	assert.Nil(t, val, "empty password should clear the encrypted field")
}

func TestBuildUpdateMap_SAMLMetadataXMLSizeLimit(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	bigXML := strings.Repeat("x", maxMetadataXMLSize+1)
	req := &UpdateConfigRequest{
		SAMLIDPMetadataXML: &bigXML,
	}

	_, err := svc.buildUpdateMap(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestBuildUpdateMap_SAMLMetadataXMLInvalidFormat(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	invalidXML := "<not-saml>bad</not-saml>"
	req := &UpdateConfigRequest{
		SAMLIDPMetadataXML: &invalidXML,
	}

	_, err := svc.buildUpdateMap(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid SAML IdP metadata XML")
}

func TestBuildUpdateMap_AllCommonFields(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &UpdateConfigRequest{
		Name:       ptr("New Name"),
		IsEnabled:  ptr(true),
		EnforceSSO: ptr(false),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updates["name"])
	assert.Equal(t, true, updates["is_enabled"])
	assert.Equal(t, false, updates["enforce_sso"])
	assert.Len(t, updates, 3)
}

func TestStripCrossProtocolEmptyFields_PreservesSameProtocol(t *testing.T) {
	// OIDC fields on OIDC config should be preserved
	req := &UpdateConfigRequest{
		OIDCIssuerURL: ptr("https://new-issuer.com"),
		OIDCClientID:  ptr("new-client"),
		OIDCScopes:    ptr("openid email"),
	}
	stripCrossProtocolEmptyFields(sso.ProtocolOIDC, req)
	assert.NotNil(t, req.OIDCIssuerURL)
	assert.NotNil(t, req.OIDCClientID)
	assert.NotNil(t, req.OIDCScopes)

	// LDAP fields on LDAP config should be preserved
	req2 := &UpdateConfigRequest{
		LDAPHost:   ptr("new-host"),
		LDAPPort:   ptr(636),
		LDAPUseTLS: ptr(true),
	}
	stripCrossProtocolEmptyFields(sso.ProtocolLDAP, req2)
	assert.NotNil(t, req2.LDAPHost)
	assert.NotNil(t, req2.LDAPPort)
	assert.NotNil(t, req2.LDAPUseTLS)
}

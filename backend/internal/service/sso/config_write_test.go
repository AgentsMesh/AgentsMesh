package sso

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEncryptionKey = "test-encryption-key-32-chars-ok!"

func newTestService(repo *mockRepository) *Service {
	cfg := &config.Config{
		PrimaryDomain: "localhost",
	}
	return NewService(repo, testEncryptionKey, cfg)
}

func TestCreateConfig_OIDC_Success(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:       "company.com",
		Name:         "Okta SSO",
		Protocol:     "oidc",
		OIDCIssuerURL: "https://company.okta.com",
		OIDCClientID:  "client-123",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.Equal(t, "company.com", cfg.Domain)
	assert.NotNil(t, cfg.OIDCIssuerURL)
	assert.Equal(t, "https://company.okta.com", *cfg.OIDCIssuerURL)
	assert.NotNil(t, cfg.OIDCClientID)
	assert.Equal(t, "client-123", *cfg.OIDCClientID)
}

func TestCreateConfig_OIDC_WithSecret(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:           "company.com",
		Name:             "Okta SSO",
		Protocol:         "oidc",
		OIDCIssuerURL:    "https://company.okta.com",
		OIDCClientID:     "client-123",
		OIDCClientSecret: "secret-456",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	// Secret should be encrypted, not stored as plaintext
	assert.NotNil(t, cfg.OIDCClientSecretEncrypted)
	assert.NotEqual(t, "secret-456", *cfg.OIDCClientSecretEncrypted)

	// Decrypted value should match original
	decrypted, err := svc.DecryptSecret(*cfg.OIDCClientSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "secret-456", decrypted)
}

func TestCreateConfig_OIDC_MissingIssuerURL(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:       "company.com",
		Name:         "Okta SSO",
		Protocol:     "oidc",
		OIDCClientID: "client-123",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC issuer URL is required")
}

func TestCreateConfig_OIDC_MissingClientID(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "company.com",
		Name:          "Okta SSO",
		Protocol:      "oidc",
		OIDCIssuerURL: "https://company.okta.com",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC client ID is required")
}

func TestCreateConfig_InvalidProtocol(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:   "company.com",
		Name:     "Bad SSO",
		Protocol: "kerberos",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.ErrorIs(t, err, ErrInvalidProtocol)
}

func TestCreateConfig_InvalidDomain(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	tests := []struct {
		name   string
		domain string
	}{
		{"empty", ""},
		{"no_dot", "localhost"},
		{"starts_with_dash", "-company.com"},
		{"special_chars", "comp@ny.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CreateConfigRequest{
				Domain:        tt.domain,
				Name:          "Test SSO",
				Protocol:      "oidc",
				OIDCIssuerURL: "https://example.com",
				OIDCClientID:  "client-123",
			}
			_, err := svc.CreateConfig(context.Background(), req, 1)
			assert.Error(t, err, "domain %q should be rejected", tt.domain)
		})
	}
}

func TestCreateConfig_DuplicateDetection(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "company.com",
		Name:          "Okta SSO",
		Protocol:      "oidc",
		OIDCIssuerURL: "https://company.okta.com",
		OIDCClientID:  "client-123",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)

	// Second create with same domain+protocol should fail
	_, err = svc.CreateConfig(context.Background(), req, 1)
	assert.ErrorIs(t, err, ErrDuplicateConfig)
}

func TestCreateConfig_DomainNormalization(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "  Company.COM  ",
		Name:          "Test",
		Protocol:      "oidc",
		OIDCIssuerURL: "https://example.com",
		OIDCClientID:  "client-123",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.Equal(t, "company.com", cfg.Domain) // lowercased and trimmed
}

func TestCreateConfig_LDAP_Success(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:     "company.com",
		Name:       "LDAP Auth",
		Protocol:   "ldap",
		LDAPHost:   "ldap.company.com",
		LDAPBaseDN: "dc=company,dc=com",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.LDAPHost)
	assert.Equal(t, "ldap.company.com", *cfg.LDAPHost)
	// Default port should be set
	assert.NotNil(t, cfg.LDAPPort)
	assert.Equal(t, 389, *cfg.LDAPPort)
}

func TestCreateConfig_LDAP_MissingHost(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:     "company.com",
		Name:       "LDAP Auth",
		Protocol:   "ldap",
		LDAPBaseDN: "dc=company,dc=com",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LDAP host is required")
}

func TestCreateConfig_LDAP_MissingBaseDN(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:   "company.com",
		Name:     "LDAP Auth",
		Protocol: "ldap",
		LDAPHost: "ldap.company.com",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LDAP base DN is required")
}

func TestCreateConfig_LDAP_CustomPort(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:     "company.com",
		Name:       "LDAP Auth",
		Protocol:   "ldap",
		LDAPHost:   "ldap.company.com",
		LDAPPort:   636,
		LDAPBaseDN: "dc=company,dc=com",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.LDAPPort)
	assert.Equal(t, 636, *cfg.LDAPPort)
}

func TestCreateConfig_SAML_MetadataURL(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:             "company.com",
		Name:               "SAML SSO",
		Protocol:           "saml",
		SAMLIDPMetadataURL: "https://idp.company.com/metadata",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.SAMLIDPMetadataURL)
	assert.Equal(t, "https://idp.company.com/metadata", *cfg.SAMLIDPMetadataURL)
	// SP Entity ID should be auto-generated
	assert.NotNil(t, cfg.SAMLSPEntityID)
	assert.Contains(t, *cfg.SAMLSPEntityID, "company.com")
}

func TestCreateConfig_SAML_NoIdPSource(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:   "company.com",
		Name:     "SAML SSO",
		Protocol: "saml",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SAML requires IdP metadata URL")
}

func TestCreateConfig_SAML_InvalidMetadataXML(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:             "company.com",
		Name:               "SAML SSO",
		Protocol:           "saml",
		SAMLIDPMetadataXML: "<not-valid-saml>oops</not-valid-saml>",
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid SAML IdP metadata XML")
}

func TestCreateConfig_SAML_OversizedMetadataXML(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Create XML larger than 1MB
	bigXML := make([]byte, maxMetadataXMLSize+1)
	for i := range bigXML {
		bigXML[i] = 'x'
	}

	req := &CreateConfigRequest{
		Domain:             "company.com",
		Name:               "SAML SSO",
		Protocol:           "saml",
		SAMLIDPMetadataXML: string(bigXML),
	}

	_, err := svc.CreateConfig(context.Background(), req, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestCreateConfig_LDAP_InvalidPort(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	tests := []struct {
		name string
		port int
	}{
		{"negative", -1},
		{"zero_is_default_so_skip", 0}, // 0 means "use default" — not an error
		{"too_high", 70000},
	}
	for _, tt := range tests {
		if tt.port == 0 {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			req := &CreateConfigRequest{
				Domain:   "company.com",
				Name:     "LDAP",
				Protocol: "ldap",
				LDAPHost: "ldap.company.com",
				LDAPPort: tt.port,
				LDAPBaseDN: "dc=company,dc=com",
			}
			_, err := svc.CreateConfig(context.Background(), req, 1)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "LDAP port must be between")
		})
	}
}

func TestCreateConfig_LDAP_PasswordEncrypted(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:           "company.com",
		Name:             "LDAP Auth",
		Protocol:         "ldap",
		LDAPHost:         "ldap.company.com",
		LDAPBaseDN:       "dc=company,dc=com",
		LDAPBindDN:       "cn=admin,dc=company,dc=com",
		LDAPBindPassword: "secret-bind-pass",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.LDAPBindPasswordEncrypted)
	assert.NotEqual(t, "secret-bind-pass", *cfg.LDAPBindPasswordEncrypted)

	decrypted, err := svc.DecryptSecret(*cfg.LDAPBindPasswordEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "secret-bind-pass", decrypted)
}

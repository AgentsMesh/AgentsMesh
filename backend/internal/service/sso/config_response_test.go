package sso

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToConfigResponse_OIDC_AllFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	issuer := "https://company.okta.com"
	clientID := "client-123"
	encrypted := "encrypted-secret"
	scopes := `["openid","email","profile"]`
	now := time.Now()
	createdBy := int64(42)

	cfg := &sso.Config{
		ID:                        1,
		Domain:                    "company.com",
		Name:                      "Okta SSO",
		Protocol:                  sso.ProtocolOIDC,
		IsEnabled:                 true,
		EnforceSSO:                true,
		OIDCIssuerURL:             &issuer,
		OIDCClientID:              &clientID,
		OIDCClientSecretEncrypted: &encrypted,
		OIDCScopes:                &scopes,
		CreatedBy:                 &createdBy,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	resp := svc.ToConfigResponse(cfg)
	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "company.com", resp.Domain)
	assert.Equal(t, "Okta SSO", resp.Name)
	assert.Equal(t, "oidc", resp.Protocol)
	assert.True(t, resp.IsEnabled)
	assert.True(t, resp.EnforceSSO)
	assert.Equal(t, issuer, resp.OIDCIssuerURL)
	assert.Equal(t, clientID, resp.OIDCClientID)
	assert.Equal(t, scopes, resp.OIDCScopes)
	assert.Equal(t, &createdBy, resp.CreatedBy)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
}

func TestToConfigResponse_SAML_AllFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	metadataURL := "https://idp.company.com/metadata"
	ssoURL := "https://idp.company.com/sso"
	spEntityID := "http://sp.company.com/metadata"
	nameIDFormat := "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	now := time.Now()

	cfg := &sso.Config{
		ID:                 2,
		Domain:             "company.com",
		Name:               "SAML SSO",
		Protocol:           sso.ProtocolSAML,
		IsEnabled:          true,
		SAMLIDPMetadataURL: &metadataURL,
		SAMLIDPSSOURL:      &ssoURL,
		SAMLSPEntityID:     &spEntityID,
		SAMLNameIDFormat:   &nameIDFormat,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	resp := svc.ToConfigResponse(cfg)
	assert.Equal(t, int64(2), resp.ID)
	assert.Equal(t, "saml", resp.Protocol)
	assert.Equal(t, metadataURL, resp.SAMLIDPMetadataURL)
	assert.Equal(t, ssoURL, resp.SAMLIDPSSOURL)
	assert.Equal(t, spEntityID, resp.SAMLSPEntityID)
	assert.Equal(t, nameIDFormat, resp.SAMLNameIDFormat)
}

func TestToConfigResponse_LDAP_AllFields(t *testing.T) {
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
	now := time.Now()

	cfg := &sso.Config{
		ID:               3,
		Domain:           "company.com",
		Name:             "AD LDAP",
		Protocol:         sso.ProtocolLDAP,
		IsEnabled:        true,
		LDAPHost:         &host,
		LDAPPort:         &port,
		LDAPUseTLS:       &useTLS,
		LDAPBindDN:       &bindDN,
		LDAPBaseDN:       &baseDN,
		LDAPUserFilter:   &userFilter,
		LDAPEmailAttr:    &emailAttr,
		LDAPNameAttr:     &nameAttr,
		LDAPUsernameAttr: &usernameAttr,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resp := svc.ToConfigResponse(cfg)
	assert.Equal(t, int64(3), resp.ID)
	assert.Equal(t, "ldap", resp.Protocol)
	assert.Equal(t, host, resp.LDAPHost)
	require.NotNil(t, resp.LDAPPort)
	assert.Equal(t, 636, *resp.LDAPPort)
	require.NotNil(t, resp.LDAPUseTLS)
	assert.True(t, *resp.LDAPUseTLS)
	assert.Equal(t, bindDN, resp.LDAPBindDN)
	assert.Equal(t, baseDN, resp.LDAPBaseDN)
	assert.Equal(t, userFilter, resp.LDAPUserFilter)
	assert.Equal(t, emailAttr, resp.LDAPEmailAttr)
	assert.Equal(t, nameAttr, resp.LDAPNameAttr)
	assert.Equal(t, usernameAttr, resp.LDAPUsernameAttr)
}

func TestToConfigResponse_NilOptionalFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	now := time.Now()
	cfg := &sso.Config{
		ID:        1,
		Domain:    "company.com",
		Name:      "Minimal",
		Protocol:  sso.ProtocolOIDC,
		CreatedAt: now,
		UpdatedAt: now,
		// All optional fields nil
	}

	resp := svc.ToConfigResponse(cfg)
	assert.Equal(t, "", resp.OIDCIssuerURL)
	assert.Equal(t, "", resp.OIDCClientID)
	assert.Equal(t, "", resp.OIDCScopes)
	assert.Nil(t, resp.LDAPPort)
	assert.Nil(t, resp.LDAPUseTLS)
	assert.Nil(t, resp.CreatedBy)
}

func TestToConfigResponse_CrossProtocolFieldsExcluded(t *testing.T) {
	svc := newTestService(newMockRepository())

	// Simulate GORM defaults: an OIDC config with LDAP/SAML default values
	port := 389
	useTLS := false
	nameIDFormat := "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	issuer := "https://idp.example.com"
	clientID := "test-client"

	cfg := &sso.Config{
		ID:               1,
		Domain:           "example.com",
		Name:             "OIDC Only",
		Protocol:         sso.ProtocolOIDC,
		OIDCIssuerURL:    &issuer,
		OIDCClientID:     &clientID,
		LDAPPort:         &port,         // GORM default leaked into OIDC row
		LDAPUseTLS:       &useTLS,       // GORM default leaked into OIDC row
		SAMLNameIDFormat: &nameIDFormat, // GORM default leaked into OIDC row
	}

	resp := svc.ToConfigResponse(cfg)
	// OIDC fields should be present
	assert.Equal(t, issuer, resp.OIDCIssuerURL)
	assert.Equal(t, clientID, resp.OIDCClientID)
	// LDAP and SAML fields should NOT appear in OIDC response
	assert.Nil(t, resp.LDAPPort, "LDAP port should not appear in OIDC response")
	assert.Nil(t, resp.LDAPUseTLS, "LDAP use_tls should not appear in OIDC response")
	assert.Empty(t, resp.SAMLNameIDFormat, "SAML name_id_format should not appear in OIDC response")
	assert.Empty(t, resp.LDAPHost)
	assert.Empty(t, resp.LDAPBaseDN)
}

func TestToDiscoverResponse_AllFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	cfg := &sso.Config{
		Domain:     "company.com",
		Name:       "Corporate SSO",
		Protocol:   sso.ProtocolSAML,
		EnforceSSO: true,
	}

	resp := svc.ToDiscoverResponse(cfg)
	assert.Equal(t, "company.com", resp.Domain)
	assert.Equal(t, "Corporate SSO", resp.Name)
	assert.Equal(t, "saml", resp.Protocol)
	assert.True(t, resp.EnforceSSO)
}

func TestToDiscoverResponse_NoEnforcement(t *testing.T) {
	svc := newTestService(newMockRepository())

	cfg := &sso.Config{
		Domain:     "company.com",
		Name:       "Optional SSO",
		Protocol:   sso.ProtocolLDAP,
		EnforceSSO: false,
	}

	resp := svc.ToDiscoverResponse(cfg)
	assert.Equal(t, "ldap", resp.Protocol)
	assert.False(t, resp.EnforceSSO)
}

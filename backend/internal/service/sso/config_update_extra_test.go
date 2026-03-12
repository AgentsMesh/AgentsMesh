package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- buildUpdateMap: SAML cert encryption ---

func TestBuildUpdateMap_SAMLCertEncryption(t *testing.T) {
	svc := newTestService(newMockRepository())

	req := &UpdateConfigRequest{
		SAMLIDPCert: ptr("-----BEGIN CERTIFICATE-----\nMIIB...test...cert\n-----END CERTIFICATE-----"),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	encrypted, ok := updates["saml_idp_cert_encrypted"]
	assert.True(t, ok)
	assert.NotEqual(t, *req.SAMLIDPCert, encrypted, "cert should be encrypted")
}

// --- buildUpdateMap: LDAP password encryption ---

func TestBuildUpdateMap_LDAPPasswordEncryption(t *testing.T) {
	svc := newTestService(newMockRepository())

	req := &UpdateConfigRequest{
		LDAPBindPassword: ptr("super-secret-password"),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	encrypted, ok := updates["ldap_bind_password_encrypted"]
	assert.True(t, ok)
	assert.NotEqual(t, "super-secret-password", encrypted)
}

// --- buildUpdateMap: all OIDC fields ---

func TestBuildUpdateMap_AllOIDCFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	req := &UpdateConfigRequest{
		OIDCIssuerURL:    ptr("https://new-issuer.com"),
		OIDCClientID:     ptr("new-client-id"),
		OIDCClientSecret: ptr("new-secret"),
		OIDCScopes:       ptr(`["openid","email"]`),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	assert.Equal(t, "https://new-issuer.com", updates["oidc_issuer_url"])
	assert.Equal(t, "new-client-id", updates["oidc_client_id"])
	assert.Contains(t, updates, "oidc_client_secret_encrypted")
	assert.Equal(t, `["openid","email"]`, updates["oidc_scopes"])
}

// --- buildUpdateMap: all SAML fields ---

func TestBuildUpdateMap_AllSAMLPlainFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	validXML := `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com">
		<IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
			<SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
		</IDPSSODescriptor>
	</EntityDescriptor>`

	req := &UpdateConfigRequest{
		SAMLIDPMetadataURL: ptr("https://idp.example.com/metadata"),
		SAMLIDPMetadataXML: ptr(validXML),
		SAMLIDPSSOURL:      ptr("https://idp.example.com/sso"),
		SAMLSPEntityID:     ptr("https://sp.example.com/metadata"),
		SAMLNameIDFormat:   ptr("urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	assert.Equal(t, "https://idp.example.com/metadata", updates["saml_idp_metadata_url"])
	assert.Equal(t, validXML, updates["saml_idp_metadata_xml"])
	assert.Equal(t, "https://idp.example.com/sso", updates["saml_idp_sso_url"])
	assert.Equal(t, "https://sp.example.com/metadata", updates["saml_sp_entity_id"])
	assert.Equal(t, "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent", updates["saml_name_id_format"])
}

// --- buildUpdateMap: all LDAP fields ---

func TestBuildUpdateMap_AllLDAPPlainFields(t *testing.T) {
	svc := newTestService(newMockRepository())

	req := &UpdateConfigRequest{
		LDAPHost:         ptr("new-ldap.company.com"),
		LDAPPort:         ptr(636),
		LDAPUseTLS:       ptr(true),
		LDAPBindDN:       ptr("cn=admin,dc=company,dc=com"),
		LDAPBaseDN:       ptr("dc=company,dc=com"),
		LDAPUserFilter:   ptr("(sAMAccountName={{username}})"),
		LDAPEmailAttr:    ptr("mail"),
		LDAPNameAttr:     ptr("cn"),
		LDAPUsernameAttr: ptr("uid"),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	assert.Equal(t, "new-ldap.company.com", updates["ldap_host"])
	assert.Equal(t, 636, updates["ldap_port"])
	assert.Equal(t, true, updates["ldap_use_tls"])
	assert.Equal(t, "cn=admin,dc=company,dc=com", updates["ldap_bind_dn"])
	assert.Equal(t, "dc=company,dc=com", updates["ldap_base_dn"])
	assert.Equal(t, "(sAMAccountName={{username}})", updates["ldap_user_filter"])
	assert.Equal(t, "mail", updates["ldap_email_attr"])
	assert.Equal(t, "cn", updates["ldap_name_attr"])
	assert.Equal(t, "uid", updates["ldap_username_attr"])
}

// --- buildUpdateMap: valid SAML metadata XML ---

func TestBuildUpdateMap_SAMLMetadataXMLValid(t *testing.T) {
	svc := newTestService(newMockRepository())

	validXML := `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com">
		<IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
			<SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
		</IDPSSODescriptor>
	</EntityDescriptor>`

	req := &UpdateConfigRequest{
		SAMLIDPMetadataXML: ptr(validXML),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	assert.Equal(t, validXML, updates["saml_idp_metadata_xml"])
}

// --- buildUpdateMap: empty SAML metadata XML clears ---

func TestBuildUpdateMap_SAMLMetadataXMLEmpty(t *testing.T) {
	svc := newTestService(newMockRepository())

	req := &UpdateConfigRequest{
		SAMLIDPMetadataXML: ptr(""),
	}

	updates, err := svc.buildUpdateMap(req)
	require.NoError(t, err)
	assert.Equal(t, "", updates["saml_idp_metadata_xml"])
}

// --- validateRequiredFieldsNotCleared: SAML with cert but no SSO URL ---

func TestValidateRequiredFieldsNotCleared_SAML_CertAndSSOURL(t *testing.T) {
	ssoURL := "https://idp.example.com/sso"
	encrypted := "encrypted-cert"
	existing := &sso.Config{
		Protocol:             sso.ProtocolSAML,
		SAMLIDPSSOURL:        &ssoURL,
		SAMLIDPCertEncrypted: &encrypted,
	}

	// Clearing SSO URL while cert exists → still invalid (both needed)
	req := &UpdateConfigRequest{
		SAMLIDPSSOURL: ptr(""),
	}
	err := validateRequiredFieldsNotCleared(existing, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one IdP source")
}

func TestValidateRequiredFieldsNotCleared_SAML_AddCertKeepsValid(t *testing.T) {
	ssoURL := "https://idp.example.com/sso"
	existing := &sso.Config{
		Protocol:      sso.ProtocolSAML,
		SAMLIDPSSOURL: &ssoURL,
	}

	// Adding cert to existing SSO URL → valid
	req := &UpdateConfigRequest{
		SAMLIDPCert: ptr("new-cert-pem"),
	}
	err := validateRequiredFieldsNotCleared(existing, req)
	assert.NoError(t, err)
}

func TestValidateRequiredFieldsNotCleared_SAML_MetadataXMLKeepsValid(t *testing.T) {
	xml := "<EntityDescriptor>...</EntityDescriptor>"
	existing := &sso.Config{
		Protocol:           sso.ProtocolSAML,
		SAMLIDPMetadataXML: &xml,
	}

	// No changes → existing XML is valid
	req := &UpdateConfigRequest{}
	err := validateRequiredFieldsNotCleared(existing, req)
	assert.NoError(t, err)
}

// --- stripCrossProtocolEmptyFields: LDAP preserves own fields ---

func TestStripCrossProtocol_LDAPPreservesAllLDAPFields(t *testing.T) {
	req := &UpdateConfigRequest{
		LDAPHost:         ptr("ldap.test.com"),
		LDAPPort:         ptr(636),
		LDAPUseTLS:       ptr(true),
		LDAPBindDN:       ptr("cn=admin"),
		LDAPBindPassword: ptr("secret"),
		LDAPBaseDN:       ptr("dc=test"),
		LDAPUserFilter:   ptr("(uid={{username}})"),
		LDAPEmailAttr:    ptr("mail"),
		LDAPNameAttr:     ptr("cn"),
		LDAPUsernameAttr: ptr("uid"),
	}
	stripCrossProtocolEmptyFields(sso.ProtocolLDAP, req)
	assert.NotNil(t, req.LDAPHost)
	assert.NotNil(t, req.LDAPPort)
	assert.NotNil(t, req.LDAPUseTLS)
	assert.NotNil(t, req.LDAPBindDN)
	assert.NotNil(t, req.LDAPBindPassword)
	assert.NotNil(t, req.LDAPBaseDN)
	assert.NotNil(t, req.LDAPUserFilter)
	assert.NotNil(t, req.LDAPEmailAttr)
	assert.NotNil(t, req.LDAPNameAttr)
	assert.NotNil(t, req.LDAPUsernameAttr)
}

// --- stripCrossProtocolEmptyFields: SAML preserves own fields ---

func TestStripCrossProtocol_SAMLPreservesAllSAMLFields(t *testing.T) {
	req := &UpdateConfigRequest{
		SAMLIDPMetadataURL: ptr("https://metadata"),
		SAMLIDPMetadataXML: ptr("<xml/>"),
		SAMLIDPSSOURL:      ptr("https://sso"),
		SAMLIDPCert:        ptr("cert"),
		SAMLSPEntityID:     ptr("entity"),
		SAMLNameIDFormat:   ptr("format"),
	}
	stripCrossProtocolEmptyFields(sso.ProtocolSAML, req)
	assert.NotNil(t, req.SAMLIDPMetadataURL)
	assert.NotNil(t, req.SAMLIDPMetadataXML)
	assert.NotNil(t, req.SAMLIDPSSOURL)
	assert.NotNil(t, req.SAMLIDPCert)
	assert.NotNil(t, req.SAMLSPEntityID)
	assert.NotNil(t, req.SAMLNameIDFormat)
}

// --- ptrStringOr ---

func TestPtrStringOr(t *testing.T) {
	val := "hello"
	assert.Equal(t, "hello", ptrStringOr(&val, "default"))
	assert.Equal(t, "default", ptrStringOr(nil, "default"))
}

// --- UpdateConfig: repo update error ---

func TestUpdateConfig_RepoUpdateError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	seedOIDCConfig(repo)
	repo.updateErr = fmt.Errorf("disk full")

	_, err := svc.UpdateConfig(context.Background(), 1, &UpdateConfigRequest{Name: ptr("New")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update SSO config")
}

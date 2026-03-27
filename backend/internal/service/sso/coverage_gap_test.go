package sso

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCertPEM creates a self-signed certificate PEM for testing.
func generateTestCertPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test IdP"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}

// --- GetAuthURL: SAML path ---

func TestGetAuthURL_SAML_Success(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	repo := newMockRepository()
	svc := newTestService(repo)

	// Seed a SAML config with manual cert + SSO URL (avoids metadata URL fetch)
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

	authURL, err := svc.GetAuthURL(context.Background(), "company.com", sso.ProtocolSAML, "test-state")
	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	// SAML redirect should contain the state as RelayState
	assert.Contains(t, authURL, "idp.example.com")
}

func TestGetAuthURL_SAML_BuildProviderError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	svc.samlProviderFactory = func(_ *sso.Config) (*ssoprovider.SAMLProvider, error) {
		return nil, fmt.Errorf("bad saml config")
	}
	seedSAMLConfig(repo)

	_, err := svc.GetAuthURL(context.Background(), "company.com", sso.ProtocolSAML, "state")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build SAML provider")
}

func TestGetAuthURL_BuildProviderError(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)
	svc.providerFactory = func(_ context.Context, _ *sso.Config) (ssoprovider.Provider, error) {
		return nil, fmt.Errorf("provider creation failed")
	}
	seedOIDCConfig(repo)

	_, err := svc.GetAuthURL(context.Background(), "company.com", sso.ProtocolOIDC, "state")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build SSO provider")
}

// --- buildProvider: SAML dispatch ---

func TestBuildProvider_SAML_Dispatch(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	svc := newTestService(newMockRepository())

	ssoURL := "https://idp.example.com/sso"
	encryptedCert, err := crypto.EncryptWithKey(certPEM, testEncryptionKey)
	require.NoError(t, err)
	spEntityID := "http://localhost/sp"

	cfg := &sso.Config{
		Protocol:             sso.ProtocolSAML,
		Domain:               "test.com",
		SAMLIDPSSOURL:        &ssoURL,
		SAMLIDPCertEncrypted: &encryptedCert,
		SAMLSPEntityID:       &spEntityID,
	}

	provider, err := svc.buildProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

// --- buildSAMLProvider: all field paths ---

func TestBuildSAMLProvider_WithCertAndSSOURL(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	svc := newTestService(newMockRepository())

	ssoURL := "https://idp.example.com/sso"
	encryptedCert, err := crypto.EncryptWithKey(certPEM, testEncryptionKey)
	require.NoError(t, err)
	spEntityID := "https://custom-sp-entity.com"
	nameIDFormat := "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"

	cfg := &sso.Config{
		Domain:               "test.com",
		Protocol:             sso.ProtocolSAML,
		SAMLIDPSSOURL:        &ssoURL,
		SAMLIDPCertEncrypted: &encryptedCert,
		SAMLSPEntityID:       &spEntityID,
		SAMLNameIDFormat:     &nameIDFormat,
	}

	provider, err := svc.buildSAMLProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestBuildSAMLProvider_WithIDPMetadataXML(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	svc := newTestService(newMockRepository())

	// Build minimal valid SAML metadata XML with the test cert
	block, _ := pem.Decode([]byte(certPEM))
	require.NotNil(t, block)

	metadataXML := fmt.Sprintf(`<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com">
		<IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
			<SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
		</IDPSSODescriptor>
	</EntityDescriptor>`)

	cfg := &sso.Config{
		Domain:             "test.com",
		Protocol:           sso.ProtocolSAML,
		SAMLIDPMetadataXML: &metadataXML,
	}

	provider, err := svc.buildSAMLProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

// --- testSAMLConnection: success path ---

func TestTestSAMLConnection_Success(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	svc := newTestService(newMockRepository())

	ssoURL := "https://idp.example.com/sso"
	encryptedCert, err := crypto.EncryptWithKey(certPEM, testEncryptionKey)
	require.NoError(t, err)
	spEntityID := "http://localhost/sp"

	cfg := &sso.Config{
		Domain:               "test.com",
		Protocol:             sso.ProtocolSAML,
		SAMLIDPSSOURL:        &ssoURL,
		SAMLIDPCertEncrypted: &encryptedCert,
		SAMLSPEntityID:       &spEntityID,
	}

	err = svc.testSAMLConnection(cfg)
	assert.NoError(t, err)
}

// --- setSAMLFields: all optional field paths ---

func TestCreateConfig_SAML_AllOptionalFields(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:           "company.com",
		Name:             "SAML SSO",
		Protocol:         "saml",
		SAMLIDPMetadataURL: "https://idp.company.com/metadata",
		SAMLSPEntityID:   "https://custom-sp.com/entity",
		SAMLNameIDFormat: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.SAMLSPEntityID)
	assert.Equal(t, "https://custom-sp.com/entity", *cfg.SAMLSPEntityID)
	assert.NotNil(t, cfg.SAMLNameIDFormat)
	assert.Equal(t, "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent", *cfg.SAMLNameIDFormat)
}

func TestCreateConfig_SAML_WithCertAndSSOURL(t *testing.T) {
	certPEM := generateTestCertPEM(t)
	repo := newMockRepository()
	svc := newTestService(repo)

	req := &CreateConfigRequest{
		Domain:        "company.com",
		Name:          "SAML Manual",
		Protocol:      "saml",
		SAMLIDPSSOURL: "https://idp.example.com/sso",
		SAMLIDPCert:   certPEM,
	}

	cfg, err := svc.CreateConfig(context.Background(), req, 1)
	require.NoError(t, err)
	assert.NotNil(t, cfg.SAMLIDPSSOURL)
	assert.NotNil(t, cfg.SAMLIDPCertEncrypted)
	assert.NotEqual(t, certPEM, *cfg.SAMLIDPCertEncrypted, "cert should be encrypted")
	// SP Entity ID should be auto-generated since not provided
	assert.NotNil(t, cfg.SAMLSPEntityID)
	assert.Contains(t, *cfg.SAMLSPEntityID, "company.com")
}


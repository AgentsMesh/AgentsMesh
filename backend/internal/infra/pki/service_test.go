package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCA creates a self-signed CA certificate for testing
func createTestCA(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create CA certificate template
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Self-sign the CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	// Encode to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	keyDER, err := x509.MarshalECPrivateKey(caKey)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

// setupTestPKI creates a test PKI service with temporary CA files
func setupTestPKI(t *testing.T) (*Service, string) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "pki-test-*")
	require.NoError(t, err)

	// Create test CA
	certPEM, keyPEM := createTestCA(t)

	// Write CA files
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")
	require.NoError(t, os.WriteFile(certFile, certPEM, 0644))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0600))

	// Create service
	cfg := &Config{
		CACertFile:   certFile,
		CAKeyFile:    keyFile,
		ValidityDays: 365,
	}

	service, err := NewService(cfg)
	require.NoError(t, err)

	return service, tmpDir
}

func TestNewService(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	assert.NotNil(t, service)
	assert.NotNil(t, service.CACert())
	assert.NotNil(t, service.CACertPool())
	assert.NotEmpty(t, service.CACertPEM())
	assert.Equal(t, 365, service.ValidityDays())
}

func TestNewService_MissingConfig(t *testing.T) {
	_, err := NewService(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewService_MissingCAFile(t *testing.T) {
	cfg := &Config{
		CACertFile: "/nonexistent/ca.crt",
		CAKeyFile:  "/nonexistent/ca.key",
	}

	_, err := NewService(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CA cert file")
}

func TestIssueRunnerCertificate(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	// Issue certificate
	certInfo, err := service.IssueRunnerCertificate("test-runner-001", "test-org")
	require.NoError(t, err)

	// Verify certificate info
	assert.NotEmpty(t, certInfo.CertPEM)
	assert.NotEmpty(t, certInfo.KeyPEM)
	assert.NotEmpty(t, certInfo.SerialNumber)
	assert.NotEmpty(t, certInfo.Fingerprint)
	assert.True(t, certInfo.IssuedAt.Before(certInfo.ExpiresAt))

	// Parse and verify the certificate
	block, _ := pem.Decode(certInfo.CertPEM)
	require.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	// Check certificate fields
	assert.Equal(t, "test-runner-001", cert.Subject.CommonName)
	assert.Contains(t, cert.Subject.Organization, "test-org")
	assert.Contains(t, cert.Subject.OrganizationalUnit, "runners")
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)

	// Verify signature chain
	opts := x509.VerifyOptions{
		Roots:     service.CACertPool(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = cert.Verify(opts)
	assert.NoError(t, err)
}

func TestIssueRunnerCertificate_EmptyNodeID(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	_, err := service.IssueRunnerCertificate("", "test-org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node_id is required")
}

func TestIssueRunnerCertificate_EmptyOrgSlug(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	_, err := service.IssueRunnerCertificate("test-runner", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org_slug is required")
}

func TestValidateCertificate(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	// Issue a certificate
	certInfo, err := service.IssueRunnerCertificate("validate-test-runner", "validate-org")
	require.NoError(t, err)

	// Validate the certificate
	nodeID, orgSlug, serialNumber, err := service.ValidateCertificate(certInfo.CertPEM)
	require.NoError(t, err)

	assert.Equal(t, "validate-test-runner", nodeID)
	assert.Equal(t, "validate-org", orgSlug)
	assert.Equal(t, certInfo.SerialNumber, serialNumber)
}

func TestValidateCertificate_InvalidPEM(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	_, _, _, err := service.ValidateCertificate([]byte("invalid pem data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode certificate PEM")
}

func TestValidateCertificate_WrongCA(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	// Create a different CA and issue a certificate with it
	otherCertPEM, otherKeyPEM := createTestCA(t)

	// Parse other CA
	block, _ := pem.Decode(otherCertPEM)
	require.NotNil(t, block)
	otherCACert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	keyBlock, _ := pem.Decode(otherKeyPEM)
	require.NotNil(t, keyBlock)
	otherCAKey, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	require.NoError(t, err)

	// Generate certificate signed by other CA
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "rogue-runner",
			Organization: []string{"rogue-org"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, otherCACert, &key.PublicKey, otherCAKey)
	require.NoError(t, err)

	rogueCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Try to validate with our service - should fail
	_, _, _, err = service.ValidateCertificate(rogueCertPEM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate verification failed")
}

func TestGetCertificateExpiry(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	// Issue a certificate
	certInfo, err := service.IssueRunnerCertificate("expiry-test", "test-org")
	require.NoError(t, err)

	// Get expiry
	expiry, err := service.GetCertificateExpiry(certInfo.CertPEM)
	require.NoError(t, err)

	// Should match the CertificateInfo expiry (within a second tolerance)
	assert.WithinDuration(t, certInfo.ExpiresAt, expiry, time.Second)
}

func TestServerCert(t *testing.T) {
	service, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	serverCert := service.ServerCert()
	assert.NotEmpty(t, serverCert.Certificate)
}

func TestDefaultValidityDays(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pki-default-validity-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	certPEM, keyPEM := createTestCA(t)
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")
	require.NoError(t, os.WriteFile(certFile, certPEM, 0644))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0600))

	// Create service without specifying validity days
	cfg := &Config{
		CACertFile:   certFile,
		CAKeyFile:    keyFile,
		ValidityDays: 0, // Should default to 365
	}

	service, err := NewService(cfg)
	require.NoError(t, err)

	assert.Equal(t, 365, service.ValidityDays())
}

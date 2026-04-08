package runner

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

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
// Delegates to testkit.SetupTestDB for shared schema.
func setupTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

// newTestService creates a Service backed by an in-memory DB for testing.
func newTestService(db *gorm.DB) *Service {
	return NewService(infra.NewRunnerRepository(db))
}

// testOrg represents a test organization
type testOrg struct {
	ID   int64
	Name string
	Slug string
}

// createTestOrg creates a test organization in the database
func createTestOrg(t *testing.T, db *gorm.DB, slug string) *testOrg {
	result := db.Exec(`
		INSERT INTO organizations (name, slug) VALUES (?, ?)
	`, "Test Org "+slug, slug)
	if result.Error != nil {
		t.Fatalf("failed to create test org: %v", result.Error)
	}

	var org testOrg
	err := db.Raw(`SELECT id, name, slug FROM organizations WHERE slug = ?`, slug).Scan(&org).Error
	if err != nil {
		t.Fatalf("failed to get test org: %v", err)
	}
	return &org
}

// createTestCA creates a self-signed CA certificate for testing
func createTestCA(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	// Create CA certificate template
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("failed to generate serial: %v", err)
	}

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
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	// Encode to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	keyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		t.Fatalf("failed to marshal CA key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

// setupTestPKI creates a test PKI service with temporary CA files
func setupTestPKI(t *testing.T) (*pki.Service, string) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "pki-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create test CA
	certPEM, keyPEM := createTestCA(t)

	// Write CA files
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	// Create service
	cfg := &pki.Config{
		CACertFile:   certFile,
		CAKeyFile:    keyFile,
		ValidityDays: 365,
	}

	service, err := pki.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create PKI service: %v", err)
	}

	return service, tmpDir
}

// Package pki provides PKI (Public Key Infrastructure) services for Runner certificate management.
// It handles CA certificate loading, Runner certificate issuance, and certificate validation.
package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// Service provides PKI operations for Runner certificate management.
type Service struct {
	caCert       *x509.Certificate
	caKey        crypto.PrivateKey
	caCertPEM    []byte
	serverCert   tls.Certificate
	certPool     *x509.CertPool
	validityDays int
}

// Config holds PKI service configuration.
type Config struct {
	CACertFile     string // Path to CA certificate file
	CAKeyFile      string // Path to CA private key file
	ServerCertFile string // Path to server certificate file (optional)
	ServerKeyFile  string // Path to server private key file (optional)
	ValidityDays   int    // Certificate validity period in days (default: 365)
}

// CertificateInfo holds information about an issued certificate.
type CertificateInfo struct {
	CertPEM      []byte
	KeyPEM       []byte
	SerialNumber string
	Fingerprint  string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// NewService creates a new PKI service instance.
func NewService(cfg *Config) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("PKI config is required")
	}

	// Load CA certificate
	caCertPEM, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert file: %w", err)
	}

	// Load CA private key
	caKeyPEM, err := os.ReadFile(cfg.CAKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA key file: %w", err)
	}

	// Parse CA certificate and key
	caCert, caKey, err := parseCA(caCertPEM, caKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA: %w", err)
	}

	// Build CA certificate pool
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	// Set default validity
	validityDays := cfg.ValidityDays
	if validityDays <= 0 {
		validityDays = 365 // Default: 1 year
	}

	s := &Service{
		caCert:       caCert,
		caKey:        caKey,
		caCertPEM:    caCertPEM,
		certPool:     certPool,
		validityDays: validityDays,
	}

	// Load or generate server certificate
	serverCert, err := s.loadOrGenerateServerCert(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load/generate server cert: %w", err)
	}
	s.serverCert = serverCert

	return s, nil
}

// parseCA parses CA certificate and private key from PEM data.
func parseCA(certPEM, keyPEM []byte) (*x509.Certificate, crypto.PrivateKey, error) {
	// Parse certificate
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse private key
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA key PEM")
	}

	var key crypto.PrivateKey

	// Try parsing as PKCS#8 first (more modern format)
	key, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try EC private key format
		key, err = x509.ParseECPrivateKey(keyBlock.Bytes)
		if err != nil {
			// Try RSA private key format
			key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse CA key: unsupported key format")
			}
		}
	}

	return cert, key, nil
}

// loadOrGenerateServerCert loads server certificate from files or generates a new one.
func (s *Service) loadOrGenerateServerCert(cfg *Config) (tls.Certificate, error) {
	// Try to load existing server certificate
	if cfg.ServerCertFile != "" && cfg.ServerKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ServerCertFile, cfg.ServerKeyFile)
		if err == nil {
			return cert, nil
		}
		// If files don't exist, generate new certificate
	}

	// Generate new server certificate
	return s.generateServerCert()
}

// generateServerCert generates a new server certificate signed by CA.
func (s *Service) generateServerCert() (tls.Certificate, error) {
	// Generate ECDSA key pair
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate server key: %w", err)
	}

	// Generate serial number
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	// Server certificate valid for 1 year
	expiresAt := now.Add(365 * 24 * time.Hour)

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "agentmesh-backend",
			Organization: []string{"AgentMesh"},
		},
		NotBefore:             now,
		NotAfter:              expiresAt,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// Add DNS names for local development
		DNSNames: []string{
			"localhost",
			"backend",
			"agentmesh-backend",
		},
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, s.caCert, &key.PublicKey, s.caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// IssueRunnerCertificate issues a client certificate for a Runner.
// The certificate CN contains the node_id and Organization contains the org_slug.
func (s *Service) IssueRunnerCertificate(nodeID, orgSlug string) (*CertificateInfo, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("org_slug is required")
	}

	// Generate ECDSA key pair (P-256 for good security/performance balance)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Generate serial number (128-bit random)
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.validityDays) * 24 * time.Hour)

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:         nodeID,                   // CN = node_id for identification
			Organization:       []string{orgSlug},        // O = org_slug for organization routing
			OrganizationalUnit: []string{"runners"},      // OU = runners to identify certificate type
		},
		NotBefore:             now,
		NotAfter:              expiresAt,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Sign with CA private key
	certDER, err := x509.CreateCertificate(rand.Reader, template, s.caCert, &key.PublicKey, s.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Calculate fingerprint (SHA-256 of DER-encoded certificate)
	fingerprint := sha256.Sum256(certDER)

	return &CertificateInfo{
		CertPEM:      certPEM,
		KeyPEM:       keyPEM,
		SerialNumber: serial.String(),
		Fingerprint:  hex.EncodeToString(fingerprint[:]),
		IssuedAt:     now,
		ExpiresAt:    expiresAt,
	}, nil
}

// ServerCert returns the server TLS certificate for gRPC server.
func (s *Service) ServerCert() tls.Certificate {
	return s.serverCert
}

// CACertPool returns the CA certificate pool for validating client certificates.
func (s *Service) CACertPool() *x509.CertPool {
	return s.certPool
}

// CACertPEM returns the CA certificate in PEM format.
// This is returned to Runners during registration for them to verify the server.
func (s *Service) CACertPEM() []byte {
	return s.caCertPEM
}

// CACert returns the parsed CA certificate.
func (s *Service) CACert() *x509.Certificate {
	return s.caCert
}

// ValidityDays returns the configured certificate validity period.
func (s *Service) ValidityDays() int {
	return s.validityDays
}

// ValidateCertificate validates a client certificate.
// Returns the node_id (CN) and org_slug (O) if valid.
func (s *Service) ValidateCertificate(certPEM []byte) (nodeID, orgSlug, serialNumber string, err error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", "", "", fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Verify certificate was signed by our CA
	opts := x509.VerifyOptions{
		Roots: s.certPool,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	if _, err := cert.Verify(opts); err != nil {
		return "", "", "", fmt.Errorf("certificate verification failed: %w", err)
	}

	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return "", "", "", fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return "", "", "", fmt.Errorf("certificate has expired")
	}

	// Extract identity from certificate
	nodeID = cert.Subject.CommonName
	if len(cert.Subject.Organization) > 0 {
		orgSlug = cert.Subject.Organization[0]
	}
	serialNumber = cert.SerialNumber.String()

	return nodeID, orgSlug, serialNumber, nil
}

// GetCertificateExpiry returns the expiry time of a certificate.
func (s *Service) GetCertificateExpiry(certPEM []byte) (time.Time, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert.NotAfter, nil
}

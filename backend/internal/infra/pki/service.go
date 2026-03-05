// Package pki provides PKI (Public Key Infrastructure) services for Runner certificate management.
// It handles CA certificate loading, Runner certificate issuance, and certificate validation.
package pki

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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
	CACertFile     string   // Path to CA certificate file
	CAKeyFile      string   // Path to CA private key file
	ServerCertFile string   // Path to server certificate file (optional)
	ServerKeyFile  string   // Path to server private key file (optional)
	ValidityDays   int      // Certificate validity period in days (default: 365)
	ServerCertSANs []string // Additional DNS SANs for auto-generated server certificate (e.g., public domain names)
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

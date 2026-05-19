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

type Service struct {
	caCert       *x509.Certificate
	caKey        crypto.PrivateKey
	caCertPEM    []byte
	serverCert   tls.Certificate
	certPool     *x509.CertPool
	validityDays int
}

type Config struct {
	CACertFile     string
	CAKeyFile      string
	ServerCertFile string
	ServerKeyFile  string
	ValidityDays   int      // Certificate validity period in days (default: 365)
	ServerCertSANs []string // Additional DNS SANs for auto-generated server certificate (e.g., public domain names)
}

type CertificateInfo struct {
	CertPEM      []byte
	KeyPEM       []byte
	SerialNumber string
	Fingerprint  string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

func NewService(cfg *Config) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("PKI config is required")
	}

	caCertPEM, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert file: %w", err)
	}

	caKeyPEM, err := os.ReadFile(cfg.CAKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA key file: %w", err)
	}

	caCert, caKey, err := parseCA(caCertPEM, caKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

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

	serverCert, err := s.loadOrGenerateServerCert(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load/generate server cert: %w", err)
	}
	s.serverCert = serverCert

	return s, nil
}

func parseCA(certPEM, keyPEM []byte) (*x509.Certificate, crypto.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA key PEM")
	}

	var key crypto.PrivateKey

	key, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		key, err = x509.ParseECPrivateKey(keyBlock.Bytes)
		if err != nil {
			key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse CA key: unsupported key format")
			}
		}
	}

	return cert, key, nil
}

func (s *Service) CACertPool() *x509.CertPool {
	return s.certPool
}

func (s *Service) CACertPEM() []byte {
	return s.caCertPEM
}

func (s *Service) CACert() *x509.Certificate {
	return s.caCert
}

func (s *Service) ValidityDays() int {
	return s.validityDays
}

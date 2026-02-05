// Package pki provides PKI (Public Key Infrastructure) services for Runner certificate management.
package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

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

// ServerCert returns the server TLS certificate for gRPC server.
func (s *Service) ServerCert() tls.Certificate {
	return s.serverCert
}

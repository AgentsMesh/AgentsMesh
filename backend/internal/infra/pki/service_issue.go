// Package pki provides PKI (Public Key Infrastructure) services for Runner certificate management.
package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

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
			CommonName:         nodeID,             // CN = node_id for identification
			Organization:       []string{orgSlug},  // O = org_slug for organization routing
			OrganizationalUnit: []string{"runners"}, // OU = runners to identify certificate type
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

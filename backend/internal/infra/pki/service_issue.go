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

func (s *Service) IssueRunnerCertificate(nodeID, orgSlug string) (*CertificateInfo, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("org_slug is required")
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.validityDays) * 24 * time.Hour)

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

	certDER, err := x509.CreateCertificate(rand.Reader, template, s.caCert, &key.PublicKey, s.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

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

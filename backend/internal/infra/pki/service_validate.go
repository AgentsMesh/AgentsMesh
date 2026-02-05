// Package pki provides PKI (Public Key Infrastructure) services for Runner certificate management.
package pki

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

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

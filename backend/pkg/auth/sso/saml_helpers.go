package sso

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/crewjam/saml"
)

// extractUserInfoFromAssertion extracts user info from a SAML assertion.
// Returns an error if the NameID (used as ExternalID) is missing or empty.
func extractUserInfoFromAssertion(assertion *saml.Assertion) (*UserInfo, error) {
	info := &UserInfo{}

	if assertion.Subject == nil || assertion.Subject.NameID == nil || assertion.Subject.NameID.Value == "" {
		return nil, fmt.Errorf("%w: SAML NameID is missing or empty", ErrAuthFailed)
	}

	info.ExternalID = assertion.Subject.NameID.Value
	// If NameID format is email, use it as email
	if assertion.Subject.NameID.Format == "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" {
		info.Email = assertion.Subject.NameID.Value
	}

	// Extract attributes
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) == 0 {
				continue
			}
			val := attr.Values[0].Value
			switch attr.Name {
			case "email", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
				info.Email = val
			case "name", "displayName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name":
				info.Name = val
			case "username", "uid", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn":
				info.Username = val
			}
		}
	}

	return info, nil
}

// parsePEMCertificate parses a PEM-encoded certificate.
// Rejects non-CERTIFICATE PEM blocks (e.g., private keys) with a clear error.
func parsePEMCertificate(pemData string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("expected PEM block type CERTIFICATE, got %q", block.Type)
	}
	return x509.ParseCertificate(block.Bytes)
}

// encodeCertificateDER returns the base64-encoded DER of a certificate
func encodeCertificateDER(cert *x509.Certificate) string {
	return base64.StdEncoding.EncodeToString(cert.Raw)
}

// fetchIDPMetadata retrieves and parses SAML IdP metadata from a URL.
func fetchIDPMetadata(metadataURL string) (*saml.EntityDescriptor, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(metadataURL) //nolint:gosec // URL is admin-configured, not user input
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from metadata URL", resp.StatusCode)
	}

	// Limit read to 1 MB to prevent memory bomb attacks
	const maxMetadataSize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMetadataSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	var metadata saml.EntityDescriptor
	if err := xml.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata XML: %w", err)
	}
	return &metadata, nil
}

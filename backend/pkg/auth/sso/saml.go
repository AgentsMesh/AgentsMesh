package sso

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
)

// SAMLConfig holds SAML provider configuration
type SAMLConfig struct {
	IDPMetadataURL string
	IDPMetadataXML string // raw XML if URL not provided
	IDPSSOURL      string
	IDPCert        string // PEM-encoded certificate
	SPEntityID     string
	SPACSURL       string
	NameIDFormat   string
}

// SAMLProvider implements Provider for SAML 2.0
type SAMLProvider struct {
	config *SAMLConfig
	sp     *saml.ServiceProvider
}

// NewSAMLProvider creates a new SAML provider
func NewSAMLProvider(cfg *SAMLConfig) (*SAMLProvider, error) {
	if cfg.SPEntityID == "" || cfg.SPACSURL == "" {
		return nil, fmt.Errorf("%w: missing SAML SP configuration", ErrInvalidConfig)
	}

	acsURL, err := url.Parse(cfg.SPACSURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ACS URL: %w", err)
	}

	entityIDURL, err := url.Parse(cfg.SPEntityID)
	if err != nil {
		return nil, fmt.Errorf("invalid SP entity ID: %w", err)
	}

	sp := &saml.ServiceProvider{
		EntityID:          entityIDURL.String(),
		AcsURL:            *acsURL,
		AllowIDPInitiated: true,
	}

	// Parse IdP metadata: prefer XML, then URL fetch, then manual cert+SSO URL
	if cfg.IDPMetadataXML != "" {
		var metadata saml.EntityDescriptor
		if err := xml.Unmarshal([]byte(cfg.IDPMetadataXML), &metadata); err != nil {
			return nil, fmt.Errorf("failed to parse IdP metadata XML: %w", err)
		}
		sp.IDPMetadata = &metadata
	} else if cfg.IDPMetadataURL != "" {
		metadata, err := fetchIDPMetadata(cfg.IDPMetadataURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch IdP metadata from URL: %w", err)
		}
		sp.IDPMetadata = metadata
	} else if cfg.IDPCert != "" && cfg.IDPSSOURL != "" {
		// Build minimal metadata from cert and SSO URL
		cert, err := parsePEMCertificate(cfg.IDPCert)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IdP certificate: %w", err)
		}

		idpSSOURL, err := url.Parse(cfg.IDPSSOURL)
		if err != nil {
			return nil, fmt.Errorf("invalid IdP SSO URL: %w", err)
		}

		idpDescriptor := saml.IDPSSODescriptor{
			SingleSignOnServices: []saml.Endpoint{
				{
					Binding:  saml.HTTPRedirectBinding,
					Location: idpSSOURL.String(),
				},
			},
		}
		idpDescriptor.KeyDescriptors = []saml.KeyDescriptor{
			{
				Use: "signing",
				KeyInfo: saml.KeyInfo{
					X509Data: saml.X509Data{
						X509Certificates: []saml.X509Certificate{
							{Data: encodeCertificateDER(cert)},
						},
					},
				},
			},
		}
		sp.IDPMetadata = &saml.EntityDescriptor{
			IDPSSODescriptors: []saml.IDPSSODescriptor{idpDescriptor},
		}
	} else {
		return nil, fmt.Errorf("%w: must provide IdP metadata XML or (cert + SSO URL)", ErrInvalidConfig)
	}

	return &SAMLProvider{
		config: cfg,
		sp:     sp,
	}, nil
}

// GetAuthURL returns the SAML AuthnRequest redirect URL.
// Only HTTPRedirectBinding is supported because GetAuthURL returns a URL string.
// HTTPPostBinding requires an HTML auto-submit form, which is incompatible with
// the redirect-based flow. Most IdPs support HTTPRedirectBinding.
func (p *SAMLProvider) GetAuthURL(ctx context.Context, state string) (string, error) {
	authURL, _, err := p.GetAuthURLWithRequestID(ctx, state)
	return authURL, err
}

// GetAuthURLWithRequestID returns the SAML AuthnRequest redirect URL along with
// the AuthnRequest ID. The caller should store this ID for InResponseTo validation.
func (p *SAMLProvider) GetAuthURLWithRequestID(_ context.Context, state string) (string, string, error) {
	ssoURL := p.sp.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	if ssoURL == "" {
		return "", "", fmt.Errorf("%w: IdP only supports HTTPPostBinding, which is not supported; configure HTTPRedirectBinding in your IdP", ErrInvalidConfig)
	}

	authnRequest, err := p.sp.MakeAuthenticationRequest(ssoURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", "", fmt.Errorf("failed to create AuthnRequest: %w", err)
	}

	requestID := authnRequest.ID

	redirectURL, err := authnRequest.Redirect(state, p.sp)
	if err != nil {
		return "", "", fmt.Errorf("failed to build redirect URL: %w", err)
	}

	return redirectURL.String(), requestID, nil
}

// HandleCallback validates the SAML response and extracts user info.
// It creates a synthetic http.Request from the SAMLResponse parameter
// so that crewjam/saml can properly parse and validate the response.
//
// Params:
//   - "SAMLResponse": the base64-encoded SAML response (required)
//   - "possibleRequestIDs": comma-separated AuthnRequest IDs for InResponseTo
//     validation. When provided, crewjam/saml verifies the response's InResponseTo
//     matches one of these IDs (SP-initiated flow). When empty, IdP-initiated
//     responses are still accepted because AllowIDPInitiated=true.
func (p *SAMLProvider) HandleCallback(_ context.Context, params map[string]string) (*UserInfo, error) {
	samlResponse := params["SAMLResponse"]
	if samlResponse == "" {
		return nil, fmt.Errorf("%w: missing SAMLResponse", ErrAuthFailed)
	}

	// Build a synthetic POST request with the SAMLResponse form data,
	// as crewjam/saml's ParseResponse reads from http.Request.FormValue.
	form := url.Values{}
	form.Set("SAMLResponse", samlResponse)
	syntheticReq, err := http.NewRequest("POST", p.config.SPACSURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build synthetic request: %w", err)
	}
	syntheticReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// ParseForm populates req.PostForm so crewjam/saml can read SAMLResponse.
	// Without this, req.PostForm is nil and ParseResponse gets an empty string.
	if err := syntheticReq.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse synthetic form: %w", err)
	}

	// Build possibleRequestIDs from params for InResponseTo validation.
	// For SP-initiated flows the service layer stores the AuthnRequest ID in Redis
	// and passes it here. For IdP-initiated flows this is empty, and
	// AllowIDPInitiated=true lets the response through without InResponseTo.
	var possibleRequestIDs []string
	if ids := params["possibleRequestIDs"]; ids != "" {
		possibleRequestIDs = strings.Split(ids, ",")
	}

	assertion, err := p.sp.ParseResponse(syntheticReq, possibleRequestIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to validate SAML response: %w", err)
	}

	userInfo, err := extractUserInfoFromAssertion(assertion)
	if err != nil {
		return nil, err
	}
	if userInfo.Email == "" {
		return nil, fmt.Errorf("%w: email not found in SAML assertion", ErrAuthFailed)
	}
	return userInfo, nil
}

// Authenticate is not supported for SAML
func (p *SAMLProvider) Authenticate(_ context.Context, _, _ string) (*UserInfo, error) {
	return nil, ErrNotSupported
}

// GenerateMetadata returns the SP metadata XML
func (p *SAMLProvider) GenerateMetadata() ([]byte, error) {
	metadata := p.sp.Metadata()
	data, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SP metadata: %w", err)
	}
	return data, nil
}

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

// GetServiceProvider returns the underlying SAML ServiceProvider (for HTTP-based ACS handling)
func (p *SAMLProvider) GetServiceProvider() *saml.ServiceProvider {
	return p.sp
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

// ValidateConfig checks if the SAML configuration is valid (for test connection)
func (p *SAMLProvider) ValidateConfig() error {
	if p.sp.IDPMetadata == nil {
		return fmt.Errorf("%w: IdP metadata not loaded", ErrInvalidConfig)
	}
	if len(p.sp.IDPMetadata.IDPSSODescriptors) == 0 {
		return fmt.Errorf("%w: no IdP SSO descriptors found", ErrInvalidConfig)
	}
	return nil
}

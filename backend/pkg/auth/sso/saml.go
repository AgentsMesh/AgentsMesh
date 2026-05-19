package sso

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/crewjam/saml"
)

type SAMLConfig struct {
	IDPMetadataURL string
	IDPMetadataXML string // raw XML if URL not provided
	IDPSSOURL      string
	IDPCert        string // PEM-encoded certificate
	SPEntityID     string
	SPACSURL       string
	NameIDFormat   string
}

type SAMLProvider struct {
	config *SAMLConfig
	sp     *saml.ServiceProvider
}

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

	if err := loadIDPMetadata(sp, cfg); err != nil {
		return nil, err
	}

	return &SAMLProvider{
		config: cfg,
		sp:     sp,
	}, nil
}

func loadIDPMetadata(sp *saml.ServiceProvider, cfg *SAMLConfig) error {
	switch {
	case cfg.IDPMetadataXML != "":
		var metadata saml.EntityDescriptor
		if err := xml.Unmarshal([]byte(cfg.IDPMetadataXML), &metadata); err != nil {
			return fmt.Errorf("failed to parse IdP metadata XML: %w", err)
		}
		sp.IDPMetadata = &metadata
	case cfg.IDPMetadataURL != "":
		metadata, err := fetchIDPMetadata(cfg.IDPMetadataURL)
		if err != nil {
			return fmt.Errorf("failed to fetch IdP metadata from URL: %w", err)
		}
		sp.IDPMetadata = metadata
	case cfg.IDPCert != "" && cfg.IDPSSOURL != "":
		metadata, err := buildManualIDPMetadata(cfg.IDPCert, cfg.IDPSSOURL)
		if err != nil {
			return err
		}
		sp.IDPMetadata = metadata
	default:
		return fmt.Errorf("%w: must provide IdP metadata XML or (cert + SSO URL)", ErrInvalidConfig)
	}
	return nil
}

func buildManualIDPMetadata(idpCert, idpSSOURLStr string) (*saml.EntityDescriptor, error) {
	cert, err := parsePEMCertificate(idpCert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IdP certificate: %w", err)
	}

	idpSSOURL, err := url.Parse(idpSSOURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid IdP SSO URL: %w", err)
	}

	idpDescriptor := saml.IDPSSODescriptor{
		SingleSignOnServices: []saml.Endpoint{
			{Binding: saml.HTTPRedirectBinding, Location: idpSSOURL.String()},
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
	return &saml.EntityDescriptor{
		IDPSSODescriptors: []saml.IDPSSODescriptor{idpDescriptor},
	}, nil
}

func (p *SAMLProvider) GetAuthURL(ctx context.Context, state string) (string, error) {
	authURL, _, err := p.GetAuthURLWithRequestID(ctx, state)
	return authURL, err
}

func (p *SAMLProvider) GetAuthURLWithRequestID(_ context.Context, state string) (string, string, error) {
	ssoURL := p.sp.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	if ssoURL == "" {
		return "", "", fmt.Errorf("%w: IdP only supports HTTPPostBinding, which is not supported; configure HTTPRedirectBinding in your IdP", ErrInvalidConfig)
	}

	authnRequest, err := p.sp.MakeAuthenticationRequest(ssoURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", "", fmt.Errorf("failed to create AuthnRequest: %w", err)
	}

	redirectURL, err := authnRequest.Redirect(state, p.sp)
	if err != nil {
		return "", "", fmt.Errorf("failed to build redirect URL: %w", err)
	}

	return redirectURL.String(), authnRequest.ID, nil
}

func (p *SAMLProvider) HandleCallback(_ context.Context, params map[string]string) (*UserInfo, error) {
	samlResponse := params["SAMLResponse"]
	if samlResponse == "" {
		return nil, fmt.Errorf("%w: missing SAMLResponse", ErrAuthFailed)
	}

	syntheticReq, err := buildSyntheticRequest(p.config.SPACSURL, samlResponse)
	if err != nil {
		return nil, err
	}

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

func buildSyntheticRequest(acsURL, samlResponse string) (*http.Request, error) {
	form := url.Values{}
	form.Set("SAMLResponse", samlResponse)
	req, err := http.NewRequest("POST", acsURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build synthetic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse synthetic form: %w", err)
	}
	return req, nil
}

func (p *SAMLProvider) Authenticate(_ context.Context, _, _ string) (*UserInfo, error) {
	return nil, ErrNotSupported
}

func (p *SAMLProvider) GenerateMetadata() ([]byte, error) {
	metadata := p.sp.Metadata()
	data, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SP metadata: %w", err)
	}
	return data, nil
}

func (p *SAMLProvider) GetServiceProvider() *saml.ServiceProvider {
	return p.sp
}

func (p *SAMLProvider) ValidateConfig() error {
	if p.sp.IDPMetadata == nil {
		return fmt.Errorf("%w: IdP metadata not loaded", ErrInvalidConfig)
	}
	if len(p.sp.IDPMetadata.IDPSSODescriptors) == 0 {
		return fmt.Errorf("%w: no IdP SSO descriptors found", ErrInvalidConfig)
	}
	return nil
}

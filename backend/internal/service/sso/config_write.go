package sso

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	samlLib "github.com/crewjam/saml"
	"gorm.io/gorm"
)

// domainRegexp validates email domain format (e.g., "company.com", "sub.company.co.uk").
var domainRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$`)

// CreateConfig creates a new SSO configuration
func (s *Service) CreateConfig(ctx context.Context, req *CreateConfigRequest, createdBy int64) (*sso.Config, error) {
	protocol := sso.Protocol(req.Protocol)
	if !sso.IsValidProtocol(protocol) {
		return nil, ErrInvalidProtocol
	}

	// Validate domain format
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if !domainRegexp.MatchString(domain) {
		return nil, fmt.Errorf("invalid domain format: %q", req.Domain)
	}
	req.Domain = domain

	// Check for duplicate
	existing, err := s.repo.GetByDomainAndProtocol(ctx, req.Domain, protocol)
	if err == nil && existing != nil {
		return nil, ErrDuplicateConfig
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check duplicate SSO config: %w", err)
	}

	cfg := &sso.Config{
		Domain:     strings.ToLower(req.Domain),
		Name:       req.Name,
		Protocol:   protocol,
		IsEnabled:  req.IsEnabled,
		EnforceSSO: req.EnforceSSO,
		CreatedBy:  &createdBy,
	}

	// Set protocol-specific fields
	var fieldErr error
	switch protocol {
	case sso.ProtocolOIDC:
		fieldErr = s.setOIDCFields(cfg, req)
	case sso.ProtocolSAML:
		fieldErr = s.setSAMLFields(cfg, req)
	case sso.ProtocolLDAP:
		fieldErr = s.setLDAPFields(cfg, req)
	}
	if fieldErr != nil {
		return nil, fieldErr
	}

	if err := s.repo.Create(ctx, cfg); err != nil {
		slog.Error("failed to create SSO config",
			"domain", cfg.Domain, "protocol", string(protocol), "created_by", createdBy, "error", err)
		return nil, fmt.Errorf("failed to create SSO config: %w", err)
	}

	slog.Info("SSO config created",
		"config_id", cfg.ID, "domain", cfg.Domain, "protocol", string(protocol))
	return cfg, nil
}

// setOIDCFields sets OIDC-specific fields on a config
func (s *Service) setOIDCFields(cfg *sso.Config, req *CreateConfigRequest) error {
	if req.OIDCIssuerURL == "" {
		return fmt.Errorf("OIDC issuer URL is required")
	}
	if req.OIDCClientID == "" {
		return fmt.Errorf("OIDC client ID is required")
	}
	cfg.OIDCIssuerURL = &req.OIDCIssuerURL
	cfg.OIDCClientID = &req.OIDCClientID
	if req.OIDCClientSecret != "" {
		encrypted, err := crypto.EncryptWithKey(req.OIDCClientSecret, s.encryptionKey)
		if err != nil {
			slog.Error("failed to encrypt OIDC client secret", "error", err)
			return fmt.Errorf("failed to encrypt OIDC client secret: %w", err)
		}
		cfg.OIDCClientSecretEncrypted = &encrypted
	}
	if req.OIDCScopes != "" {
		cfg.OIDCScopes = &req.OIDCScopes
	}
	return nil
}

// maxMetadataXMLSize is the maximum allowed size for inline SAML IdP metadata XML (1 MB).
// This matches the limit applied when fetching metadata via URL in the provider layer.
const maxMetadataXMLSize = 1 << 20

// setSAMLFields sets SAML-specific fields on a config
func (s *Service) setSAMLFields(cfg *sso.Config, req *CreateConfigRequest) error {
	// Validate metadata XML size before any other checks
	if len(req.SAMLIDPMetadataXML) > maxMetadataXMLSize {
		return fmt.Errorf("SAML IdP metadata XML exceeds maximum size of %d bytes", maxMetadataXMLSize)
	}

	// At least one IdP source is required: metadata URL, metadata XML, or (SSO URL + cert)
	hasMetadataURL := req.SAMLIDPMetadataURL != ""
	hasMetadataXML := req.SAMLIDPMetadataXML != ""
	hasManualConfig := req.SAMLIDPSSOURL != "" && req.SAMLIDPCert != ""
	if !hasMetadataURL && !hasMetadataXML && !hasManualConfig {
		return fmt.Errorf("SAML requires IdP metadata URL, metadata XML, or SSO URL with certificate")
	}
	if req.SAMLIDPMetadataURL != "" {
		cfg.SAMLIDPMetadataURL = &req.SAMLIDPMetadataURL
	}
	if req.SAMLIDPMetadataXML != "" {
		// Validate XML is well-formed and parseable as SAML metadata
		var metadata samlLib.EntityDescriptor
		if err := xml.Unmarshal([]byte(req.SAMLIDPMetadataXML), &metadata); err != nil {
			return fmt.Errorf("invalid SAML IdP metadata XML: %w", err)
		}
		cfg.SAMLIDPMetadataXML = &req.SAMLIDPMetadataXML
	}
	if req.SAMLIDPSSOURL != "" {
		cfg.SAMLIDPSSOURL = &req.SAMLIDPSSOURL
	}
	if req.SAMLIDPCert != "" {
		encrypted, err := crypto.EncryptWithKey(req.SAMLIDPCert, s.encryptionKey)
		if err != nil {
			slog.Error("failed to encrypt SAML IdP cert", "error", err)
			return fmt.Errorf("failed to encrypt SAML IdP cert: %w", err)
		}
		cfg.SAMLIDPCertEncrypted = &encrypted
	}
	// Auto-generate SP Entity ID if not provided
	if req.SAMLSPEntityID != "" {
		cfg.SAMLSPEntityID = &req.SAMLSPEntityID
	} else {
		spEntityID := fmt.Sprintf("%s/api/v1/auth/sso/%s/saml/metadata", s.config.BaseURL(), cfg.Domain)
		cfg.SAMLSPEntityID = &spEntityID
	}
	if req.SAMLNameIDFormat != "" {
		cfg.SAMLNameIDFormat = &req.SAMLNameIDFormat
	}
	return nil
}

// setLDAPFields sets LDAP-specific fields on a config
func (s *Service) setLDAPFields(cfg *sso.Config, req *CreateConfigRequest) error {
	if req.LDAPHost == "" {
		return fmt.Errorf("LDAP host is required")
	}
	if req.LDAPBaseDN == "" {
		return fmt.Errorf("LDAP base DN is required")
	}
	cfg.LDAPHost = &req.LDAPHost
	if req.LDAPPort != 0 {
		if req.LDAPPort < 1 || req.LDAPPort > 65535 {
			return fmt.Errorf("LDAP port must be between 1 and 65535, got %d", req.LDAPPort)
		}
		cfg.LDAPPort = &req.LDAPPort
	} else {
		defaultPort := 389
		cfg.LDAPPort = &defaultPort
	}
	cfg.LDAPUseTLS = &req.LDAPUseTLS
	if req.LDAPBindDN != "" {
		cfg.LDAPBindDN = &req.LDAPBindDN
	}
	if req.LDAPBindPassword != "" {
		encrypted, err := crypto.EncryptWithKey(req.LDAPBindPassword, s.encryptionKey)
		if err != nil {
			slog.Error("failed to encrypt LDAP bind password", "error", err)
			return fmt.Errorf("failed to encrypt LDAP bind password: %w", err)
		}
		cfg.LDAPBindPasswordEncrypted = &encrypted
	}
	if req.LDAPBaseDN != "" {
		cfg.LDAPBaseDN = &req.LDAPBaseDN
	}
	if req.LDAPUserFilter != "" {
		cfg.LDAPUserFilter = &req.LDAPUserFilter
	}
	if req.LDAPEmailAttr != "" {
		cfg.LDAPEmailAttr = &req.LDAPEmailAttr
	}
	if req.LDAPNameAttr != "" {
		cfg.LDAPNameAttr = &req.LDAPNameAttr
	}
	if req.LDAPUsernameAttr != "" {
		cfg.LDAPUsernameAttr = &req.LDAPUsernameAttr
	}
	return nil
}

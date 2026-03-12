package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// buildProvider creates an SSO provider from a config
func (s *Service) buildProvider(ctx context.Context, cfg *sso.Config) (ssoprovider.Provider, error) {
	if s.providerFactory != nil {
		return s.providerFactory(ctx, cfg)
	}
	switch cfg.Protocol {
	case sso.ProtocolOIDC:
		return s.buildOIDCProvider(ctx, cfg)
	case sso.ProtocolSAML:
		return s.buildSAMLProvider(cfg)
	case sso.ProtocolLDAP:
		return s.buildLDAPProvider(cfg)
	default:
		return nil, ErrInvalidProtocol
	}
}

func (s *Service) buildOIDCProvider(ctx context.Context, cfg *sso.Config) (ssoprovider.Provider, error) {
	clientSecret := ""
	if cfg.OIDCClientSecretEncrypted != nil {
		var err error
		clientSecret, err = crypto.DecryptWithKey(*cfg.OIDCClientSecretEncrypted, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt client secret: %w", err)
		}
	}

	scopes := []string{"openid", "email", "profile"}
	if cfg.OIDCScopes != nil && *cfg.OIDCScopes != "" {
		if err := json.Unmarshal([]byte(*cfg.OIDCScopes), &scopes); err != nil {
			// Fallback: try space-separated or comma-separated format
			slog.Warn("OIDC scopes not valid JSON, falling back to text parsing",
				"domain", cfg.Domain, "scopes", *cfg.OIDCScopes, "error", err)
			raw := strings.TrimSpace(*cfg.OIDCScopes)
			if strings.Contains(raw, ",") {
				scopes = strings.Split(raw, ",")
			} else {
				scopes = strings.Fields(raw)
			}
		}
		// Trim whitespace from each scope
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}
	}

	redirectURL := fmt.Sprintf("%s/api/v1/auth/sso/%s/oidc/callback", s.config.BaseURL(), cfg.Domain)

	issuerURL := ""
	if cfg.OIDCIssuerURL != nil {
		issuerURL = *cfg.OIDCIssuerURL
	}
	clientID := ""
	if cfg.OIDCClientID != nil {
		clientID = *cfg.OIDCClientID
	}

	return ssoprovider.NewOIDCProvider(ctx, &ssoprovider.OIDCConfig{
		IssuerURL:    issuerURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	})
}

func (s *Service) buildSAMLProvider(cfg *sso.Config) (*ssoprovider.SAMLProvider, error) {
	if s.samlProviderFactory != nil {
		return s.samlProviderFactory(cfg)
	}
	idpCert := ""
	if cfg.SAMLIDPCertEncrypted != nil {
		var err error
		idpCert, err = crypto.DecryptWithKey(*cfg.SAMLIDPCertEncrypted, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt IdP cert: %w", err)
		}
	}

	spEntityID := fmt.Sprintf("%s/api/v1/auth/sso/%s/saml/metadata", s.config.BaseURL(), cfg.Domain)
	if cfg.SAMLSPEntityID != nil && *cfg.SAMLSPEntityID != "" {
		spEntityID = *cfg.SAMLSPEntityID
	}
	acsURL := fmt.Sprintf("%s/api/v1/auth/sso/%s/saml/acs", s.config.BaseURL(), cfg.Domain)

	samlCfg := &ssoprovider.SAMLConfig{
		SPEntityID:   spEntityID,
		SPACSURL:     acsURL,
		IDPCert:      idpCert,
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
	}

	if cfg.SAMLIDPMetadataURL != nil {
		samlCfg.IDPMetadataURL = *cfg.SAMLIDPMetadataURL
	}
	if cfg.SAMLIDPMetadataXML != nil {
		samlCfg.IDPMetadataXML = *cfg.SAMLIDPMetadataXML
	}
	if cfg.SAMLIDPSSOURL != nil {
		samlCfg.IDPSSOURL = *cfg.SAMLIDPSSOURL
	}
	if cfg.SAMLNameIDFormat != nil {
		samlCfg.NameIDFormat = *cfg.SAMLNameIDFormat
	}

	return ssoprovider.NewSAMLProvider(samlCfg)
}

func (s *Service) buildLDAPProvider(cfg *sso.Config) (*ssoprovider.LDAPProvider, error) {
	bindPassword := ""
	if cfg.LDAPBindPasswordEncrypted != nil {
		var err error
		bindPassword, err = crypto.DecryptWithKey(*cfg.LDAPBindPasswordEncrypted, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt bind password: %w", err)
		}
	}

	ldapCfg := &ssoprovider.LDAPConfig{
		BindPassword: bindPassword,
	}
	if cfg.LDAPHost != nil {
		ldapCfg.Host = *cfg.LDAPHost
	}
	if cfg.LDAPPort != nil {
		ldapCfg.Port = *cfg.LDAPPort
	}
	if cfg.LDAPUseTLS != nil {
		ldapCfg.UseTLS = *cfg.LDAPUseTLS
	}
	if cfg.LDAPBindDN != nil {
		ldapCfg.BindDN = *cfg.LDAPBindDN
	}
	if cfg.LDAPBaseDN != nil {
		ldapCfg.BaseDN = *cfg.LDAPBaseDN
	}
	if cfg.LDAPUserFilter != nil {
		ldapCfg.UserFilter = *cfg.LDAPUserFilter
	}
	if cfg.LDAPEmailAttr != nil {
		ldapCfg.EmailAttr = *cfg.LDAPEmailAttr
	}
	if cfg.LDAPNameAttr != nil {
		ldapCfg.NameAttr = *cfg.LDAPNameAttr
	}
	if cfg.LDAPUsernameAttr != nil {
		ldapCfg.UsernameAttr = *cfg.LDAPUsernameAttr
	}

	return ssoprovider.NewLDAPProvider(ldapCfg)
}

// Test connection helpers

func (s *Service) testOIDCConnection(ctx context.Context, cfg *sso.Config) error {
	_, err := s.buildOIDCProvider(ctx, cfg)
	return err // Provider creation performs Discovery, which validates the issuer
}

func (s *Service) testSAMLConnection(cfg *sso.Config) error {
	provider, err := s.buildSAMLProvider(cfg)
	if err != nil {
		return err
	}
	return provider.ValidateConfig()
}

func (s *Service) testLDAPConnection(cfg *sso.Config) error {
	provider, err := s.buildLDAPProvider(cfg)
	if err != nil {
		return err
	}
	return provider.TestConnection()
}

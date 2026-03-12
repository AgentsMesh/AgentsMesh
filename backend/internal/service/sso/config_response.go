package sso

import "github.com/anthropics/agentsmesh/backend/internal/domain/sso"

// ToConfigResponse converts a domain Config to a response DTO (strips secrets).
// Only includes fields relevant to the config's protocol to avoid leaking
// cross-protocol GORM default values (e.g., ldap_port=389 on an OIDC config).
func (s *Service) ToConfigResponse(cfg *sso.Config) *ConfigResponse {
	resp := &ConfigResponse{
		ID:         cfg.ID,
		Domain:     cfg.Domain,
		Name:       cfg.Name,
		Protocol:   string(cfg.Protocol),
		IsEnabled:  cfg.IsEnabled,
		EnforceSSO: cfg.EnforceSSO,
		CreatedBy:  cfg.CreatedBy,
		CreatedAt:  cfg.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  cfg.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Include non-sensitive fields filtered by protocol
	switch cfg.Protocol {
	case sso.ProtocolOIDC:
		if cfg.OIDCIssuerURL != nil {
			resp.OIDCIssuerURL = *cfg.OIDCIssuerURL
		}
		if cfg.OIDCClientID != nil {
			resp.OIDCClientID = *cfg.OIDCClientID
		}
		if cfg.OIDCScopes != nil {
			resp.OIDCScopes = *cfg.OIDCScopes
		}

	case sso.ProtocolSAML:
		if cfg.SAMLIDPMetadataURL != nil {
			resp.SAMLIDPMetadataURL = *cfg.SAMLIDPMetadataURL
		}
		if cfg.SAMLIDPSSOURL != nil {
			resp.SAMLIDPSSOURL = *cfg.SAMLIDPSSOURL
		}
		if cfg.SAMLSPEntityID != nil {
			resp.SAMLSPEntityID = *cfg.SAMLSPEntityID
		}
		if cfg.SAMLNameIDFormat != nil {
			resp.SAMLNameIDFormat = *cfg.SAMLNameIDFormat
		}

	case sso.ProtocolLDAP:
		if cfg.LDAPHost != nil {
			resp.LDAPHost = *cfg.LDAPHost
		}
		if cfg.LDAPPort != nil {
			resp.LDAPPort = cfg.LDAPPort
		}
		if cfg.LDAPUseTLS != nil {
			resp.LDAPUseTLS = cfg.LDAPUseTLS
		}
		if cfg.LDAPBindDN != nil {
			resp.LDAPBindDN = *cfg.LDAPBindDN
		}
		if cfg.LDAPBaseDN != nil {
			resp.LDAPBaseDN = *cfg.LDAPBaseDN
		}
		if cfg.LDAPUserFilter != nil {
			resp.LDAPUserFilter = *cfg.LDAPUserFilter
		}
		if cfg.LDAPEmailAttr != nil {
			resp.LDAPEmailAttr = *cfg.LDAPEmailAttr
		}
		if cfg.LDAPNameAttr != nil {
			resp.LDAPNameAttr = *cfg.LDAPNameAttr
		}
		if cfg.LDAPUsernameAttr != nil {
			resp.LDAPUsernameAttr = *cfg.LDAPUsernameAttr
		}
	}

	return resp
}

// ToDiscoverResponse creates a sanitized discover response
func (s *Service) ToDiscoverResponse(cfg *sso.Config) *DiscoverResponse {
	return &DiscoverResponse{
		Domain:     cfg.Domain,
		Name:       cfg.Name,
		Protocol:   string(cfg.Protocol),
		EnforceSSO: cfg.EnforceSSO,
	}
}

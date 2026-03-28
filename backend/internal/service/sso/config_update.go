package sso

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	samlLib "github.com/crewjam/saml"
	"gorm.io/gorm"
)

// UpdateConfig updates an SSO configuration
func (s *Service) UpdateConfig(ctx context.Context, id int64, req *UpdateConfigRequest) (*sso.Config, error) {
	// Load current config to validate protocol-specific field constraints.
	existing, err := s.GetConfig(ctx, id)
	if err != nil {
		return nil, err
	}

	// Strip all non-matching protocol fields unconditionally.
	// Frontends send every field (including defaults like ldap_port=389),
	// so we nil them before any validation to avoid false cross-protocol errors.
	stripCrossProtocolEmptyFields(existing.Protocol, req)

	// Reject explicitly clearing required fields (pointer non-nil but value empty).
	if err := validateRequiredFieldsNotCleared(existing, req); err != nil {
		return nil, err
	}

	updates, err := s.buildUpdateMap(req)
	if err != nil {
		slog.Error("failed to build SSO update map", "config_id", id, "protocol", existing.Protocol, "error", err)
		return nil, fmt.Errorf("failed to build update map: %w", err)
	}
	if len(updates) == 0 {
		return existing, nil
	}

	if err := s.repo.Update(ctx, id, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConfigNotFound
		}
		slog.Error("failed to update SSO config", "config_id", id, "protocol", existing.Protocol, "error", err)
		return nil, fmt.Errorf("failed to update SSO config: %w", err)
	}

	slog.Info("SSO config updated", "config_id", id, "protocol", existing.Protocol, "domain", existing.Domain)

	return s.GetConfig(ctx, id)
}

// buildUpdateMap constructs the GORM column→value map from an UpdateConfigRequest.
// Returns an error if secret encryption fails (never silently swallows encryption errors).
func (s *Service) buildUpdateMap(req *UpdateConfigRequest) (map[string]interface{}, error) {
	updates := make(map[string]interface{})

	// Common fields
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.EnforceSSO != nil {
		updates["enforce_sso"] = *req.EnforceSSO
	}

	// OIDC fields
	if req.OIDCIssuerURL != nil {
		updates["oidc_issuer_url"] = *req.OIDCIssuerURL
	}
	if req.OIDCClientID != nil {
		updates["oidc_client_id"] = *req.OIDCClientID
	}
	if req.OIDCClientSecret != nil {
		if *req.OIDCClientSecret != "" {
			encrypted, err := crypto.EncryptWithKey(*req.OIDCClientSecret, s.encryptionKey)
			if err != nil {
				slog.Error("failed to encrypt OIDC client secret", "error", err)
				return nil, fmt.Errorf("failed to encrypt OIDC client secret: %w", err)
			}
			updates["oidc_client_secret_encrypted"] = encrypted
		} else {
			// Clear secret (e.g., switching to public client)
			updates["oidc_client_secret_encrypted"] = nil
		}
	}
	if req.OIDCScopes != nil {
		updates["oidc_scopes"] = *req.OIDCScopes
	}

	// SAML fields
	if req.SAMLIDPMetadataURL != nil {
		updates["saml_idp_metadata_url"] = *req.SAMLIDPMetadataURL
	}
	if req.SAMLIDPMetadataXML != nil {
		if *req.SAMLIDPMetadataXML != "" {
			if len(*req.SAMLIDPMetadataXML) > maxMetadataXMLSize {
				return nil, fmt.Errorf("SAML IdP metadata XML exceeds maximum size of %d bytes", maxMetadataXMLSize)
			}
			var metadata samlLib.EntityDescriptor
			if err := xml.Unmarshal([]byte(*req.SAMLIDPMetadataXML), &metadata); err != nil {
				return nil, fmt.Errorf("invalid SAML IdP metadata XML: %w", err)
			}
		}
		updates["saml_idp_metadata_xml"] = *req.SAMLIDPMetadataXML
	}
	if req.SAMLIDPSSOURL != nil {
		updates["saml_idp_sso_url"] = *req.SAMLIDPSSOURL
	}
	if req.SAMLIDPCert != nil {
		if *req.SAMLIDPCert != "" {
			encrypted, err := crypto.EncryptWithKey(*req.SAMLIDPCert, s.encryptionKey)
			if err != nil {
				slog.Error("failed to encrypt SAML IdP cert", "error", err)
				return nil, fmt.Errorf("failed to encrypt SAML IdP cert: %w", err)
			}
			updates["saml_idp_cert_encrypted"] = encrypted
		} else {
			// Explicitly clear the certificate (validated by validateRequiredFieldsNotCleared)
			updates["saml_idp_cert_encrypted"] = nil
		}
	}
	if req.SAMLSPEntityID != nil {
		updates["saml_sp_entity_id"] = *req.SAMLSPEntityID
	}
	if req.SAMLNameIDFormat != nil {
		updates["saml_name_id_format"] = *req.SAMLNameIDFormat
	}

	// LDAP fields
	if req.LDAPHost != nil {
		updates["ldap_host"] = *req.LDAPHost
	}
	if req.LDAPPort != nil {
		updates["ldap_port"] = *req.LDAPPort
	}
	if req.LDAPUseTLS != nil {
		updates["ldap_use_tls"] = *req.LDAPUseTLS
	}
	if req.LDAPBindDN != nil {
		updates["ldap_bind_dn"] = *req.LDAPBindDN
	}
	if req.LDAPBindPassword != nil {
		if *req.LDAPBindPassword != "" {
			encrypted, err := crypto.EncryptWithKey(*req.LDAPBindPassword, s.encryptionKey)
			if err != nil {
				slog.Error("failed to encrypt LDAP bind password", "error", err)
				return nil, fmt.Errorf("failed to encrypt LDAP bind password: %w", err)
			}
			updates["ldap_bind_password_encrypted"] = encrypted
		} else {
			// Clear password (e.g., switching to anonymous bind)
			updates["ldap_bind_password_encrypted"] = nil
		}
	}
	if req.LDAPBaseDN != nil {
		updates["ldap_base_dn"] = *req.LDAPBaseDN
	}
	if req.LDAPUserFilter != nil {
		updates["ldap_user_filter"] = *req.LDAPUserFilter
	}
	if req.LDAPEmailAttr != nil {
		updates["ldap_email_attr"] = *req.LDAPEmailAttr
	}
	if req.LDAPNameAttr != nil {
		updates["ldap_name_attr"] = *req.LDAPNameAttr
	}
	if req.LDAPUsernameAttr != nil {
		updates["ldap_username_attr"] = *req.LDAPUsernameAttr
	}

	return updates, nil
}

// stripCrossProtocolEmptyFields nils out ALL fields that don't belong to the
// config's protocol. Frontends typically send every field (including defaults
// like ldap_port=389 or ldap_user_filter="(uid=%s)") — these must be stripped
// unconditionally, not just when empty.
func stripCrossProtocolEmptyFields(protocol sso.Protocol, req *UpdateConfigRequest) {
	if protocol != sso.ProtocolOIDC {
		req.OIDCIssuerURL = nil
		req.OIDCClientID = nil
		req.OIDCClientSecret = nil
		req.OIDCScopes = nil
	}
	if protocol != sso.ProtocolSAML {
		req.SAMLIDPMetadataURL = nil
		req.SAMLIDPMetadataXML = nil
		req.SAMLIDPSSOURL = nil
		req.SAMLIDPCert = nil
		req.SAMLSPEntityID = nil
		req.SAMLNameIDFormat = nil
	}
	if protocol != sso.ProtocolLDAP {
		req.LDAPHost = nil
		req.LDAPPort = nil
		req.LDAPUseTLS = nil
		req.LDAPBindDN = nil
		req.LDAPBindPassword = nil
		req.LDAPBaseDN = nil
		req.LDAPUserFilter = nil
		req.LDAPEmailAttr = nil
		req.LDAPNameAttr = nil
		req.LDAPUsernameAttr = nil
	}
}

// validateRequiredFieldsNotCleared rejects updates that clear protocol-specific
// required fields (pointer non-nil but value empty).
func validateRequiredFieldsNotCleared(existing *sso.Config, req *UpdateConfigRequest) error {
	switch existing.Protocol {
	case sso.ProtocolOIDC:
		if req.OIDCIssuerURL != nil && *req.OIDCIssuerURL == "" {
			return NewValidationError("OIDC issuer URL cannot be empty")
		}
		if req.OIDCClientID != nil && *req.OIDCClientID == "" {
			return NewValidationError("OIDC client ID cannot be empty")
		}
	case sso.ProtocolSAML:
		// SAML requires at least one IdP source to remain after the update.
		// Simulate the post-update state by overlaying request values on existing config.
		metadataURL := ptrStringOr(existing.SAMLIDPMetadataURL, "")
		if req.SAMLIDPMetadataURL != nil {
			metadataURL = *req.SAMLIDPMetadataURL
		}
		metadataXML := ptrStringOr(existing.SAMLIDPMetadataXML, "")
		if req.SAMLIDPMetadataXML != nil {
			metadataXML = *req.SAMLIDPMetadataXML
		}
		ssoURL := ptrStringOr(existing.SAMLIDPSSOURL, "")
		if req.SAMLIDPSSOURL != nil {
			ssoURL = *req.SAMLIDPSSOURL
		}
		hasCert := existing.SAMLIDPCertEncrypted != nil
		if req.SAMLIDPCert != nil && *req.SAMLIDPCert == "" {
			hasCert = false
		} else if req.SAMLIDPCert != nil && *req.SAMLIDPCert != "" {
			hasCert = true
		}
		if metadataURL == "" && metadataXML == "" && (ssoURL == "" || !hasCert) {
			return NewValidationError("SAML requires at least one IdP source (metadata URL, metadata XML, or SSO URL with certificate)")
		}
	case sso.ProtocolLDAP:
		if req.LDAPHost != nil && *req.LDAPHost == "" {
			return NewValidationError("LDAP host cannot be empty")
		}
		if req.LDAPBaseDN != nil && *req.LDAPBaseDN == "" {
			return NewValidationError("LDAP base DN cannot be empty")
		}
	}
	return nil
}

// ptrStringOr dereferences a *string, returning fallback if nil.
func ptrStringOr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

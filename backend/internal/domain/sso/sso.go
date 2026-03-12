package sso

import "time"

// Protocol represents SSO protocol type
type Protocol string

const (
	ProtocolOIDC Protocol = "oidc"
	ProtocolSAML Protocol = "saml"
	ProtocolLDAP Protocol = "ldap"
)

// Config represents an SSO configuration for a domain
type Config struct {
	ID         int64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Domain     string   `gorm:"size:255;not null;uniqueIndex:idx_sso_configs_domain_protocol" json:"domain"`
	Name       string   `gorm:"size:100;not null" json:"name"`
	Protocol   Protocol `gorm:"type:sso_protocol;not null;uniqueIndex:idx_sso_configs_domain_protocol" json:"protocol"`
	IsEnabled  bool     `gorm:"not null;default:false" json:"is_enabled"`
	EnforceSSO bool     `gorm:"not null;default:false" json:"enforce_sso"`

	// OIDC fields
	OIDCIssuerURL             *string `gorm:"column:oidc_issuer_url;type:text" json:"oidc_issuer_url,omitempty"`
	OIDCClientID              *string `gorm:"column:oidc_client_id;size:255" json:"oidc_client_id,omitempty"`
	OIDCClientSecretEncrypted *string `gorm:"column:oidc_client_secret_encrypted;type:text" json:"-"`
	OIDCScopes                *string `gorm:"column:oidc_scopes;type:text" json:"oidc_scopes,omitempty"`

	// SAML fields
	SAMLIDPMetadataURL   *string `gorm:"column:saml_idp_metadata_url;type:text" json:"saml_idp_metadata_url,omitempty"`
	SAMLIDPMetadataXML   *string `gorm:"column:saml_idp_metadata_xml;type:text" json:"-"`
	SAMLIDPSSOURL        *string `gorm:"column:saml_idp_sso_url;type:text" json:"saml_idp_sso_url,omitempty"`
	SAMLIDPCertEncrypted *string `gorm:"column:saml_idp_cert_encrypted;type:text" json:"-"`
	SAMLSPEntityID       *string `gorm:"column:saml_sp_entity_id;type:text" json:"saml_sp_entity_id,omitempty"`
	SAMLNameIDFormat     *string `gorm:"column:saml_name_id_format;size:100;default:'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress'" json:"saml_name_id_format,omitempty"`

	// LDAP fields
	LDAPHost                  *string `gorm:"column:ldap_host;size:255" json:"ldap_host,omitempty"`
	LDAPPort                  *int    `gorm:"column:ldap_port;default:389" json:"ldap_port,omitempty"`
	LDAPUseTLS                *bool   `gorm:"column:ldap_use_tls;default:false" json:"ldap_use_tls,omitempty"`
	LDAPBindDN                *string `gorm:"column:ldap_bind_dn;type:text" json:"ldap_bind_dn,omitempty"`
	LDAPBindPasswordEncrypted *string `gorm:"column:ldap_bind_password_encrypted;type:text" json:"-"`
	LDAPBaseDN                *string `gorm:"column:ldap_base_dn;type:text" json:"ldap_base_dn,omitempty"`
	LDAPUserFilter            *string `gorm:"column:ldap_user_filter;type:text" json:"ldap_user_filter,omitempty"`
	LDAPEmailAttr             *string `gorm:"column:ldap_email_attr;size:100" json:"ldap_email_attr,omitempty"`
	LDAPNameAttr              *string `gorm:"column:ldap_name_attr;size:100" json:"ldap_name_attr,omitempty"`
	LDAPUsernameAttr          *string `gorm:"column:ldap_username_attr;size:100" json:"ldap_username_attr,omitempty"`

	// Audit
	CreatedBy *int64    `gorm:"column:created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name
func (Config) TableName() string {
	return "sso_configs"
}

// IsValidProtocol checks if the protocol is valid
func IsValidProtocol(p Protocol) bool {
	switch p {
	case ProtocolOIDC, ProtocolSAML, ProtocolLDAP:
		return true
	default:
		return false
	}
}

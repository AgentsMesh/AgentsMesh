package sso

import "errors"

var (
	ErrConfigNotFound  = errors.New("SSO config not found")
	ErrDuplicateConfig = errors.New("SSO config already exists for this domain and protocol")
	ErrInvalidProtocol = errors.New("invalid SSO protocol")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func NewValidationError(msg string) *ValidationError { return &ValidationError{Message: msg} }

type CreateConfigRequest struct {
	Domain     string `json:"domain" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Protocol   string `json:"protocol" binding:"required"`
	IsEnabled  bool   `json:"is_enabled"`
	EnforceSSO bool   `json:"enforce_sso"`

	OIDCIssuerURL    string `json:"oidc_issuer_url,omitempty"`
	OIDCClientID     string `json:"oidc_client_id,omitempty"`
	OIDCClientSecret string `json:"oidc_client_secret,omitempty"`
	OIDCScopes       string `json:"oidc_scopes,omitempty"`

	SAMLIDPMetadataURL string `json:"saml_idp_metadata_url,omitempty"`
	SAMLIDPMetadataXML string `json:"saml_idp_metadata_xml,omitempty"`
	SAMLIDPSSOURL      string `json:"saml_idp_sso_url,omitempty"`
	SAMLIDPCert        string `json:"saml_idp_cert,omitempty"`
	SAMLSPEntityID     string `json:"saml_sp_entity_id,omitempty"`
	SAMLNameIDFormat   string `json:"saml_name_id_format,omitempty"`

	LDAPHost         string `json:"ldap_host,omitempty"`
	LDAPPort         int    `json:"ldap_port,omitempty"`
	LDAPUseTLS       bool   `json:"ldap_use_tls,omitempty"`
	LDAPBindDN       string `json:"ldap_bind_dn,omitempty"`
	LDAPBindPassword string `json:"ldap_bind_password,omitempty"`
	LDAPBaseDN       string `json:"ldap_base_dn,omitempty"`
	LDAPUserFilter   string `json:"ldap_user_filter,omitempty"`
	LDAPEmailAttr    string `json:"ldap_email_attr,omitempty"`
	LDAPNameAttr     string `json:"ldap_name_attr,omitempty"`
	LDAPUsernameAttr string `json:"ldap_username_attr,omitempty"`
}

type UpdateConfigRequest struct {
	Name       *string `json:"name,omitempty"`
	IsEnabled  *bool   `json:"is_enabled,omitempty"`
	EnforceSSO *bool   `json:"enforce_sso,omitempty"`

	OIDCIssuerURL    *string `json:"oidc_issuer_url,omitempty"`
	OIDCClientID     *string `json:"oidc_client_id,omitempty"`
	OIDCClientSecret *string `json:"oidc_client_secret,omitempty"`
	OIDCScopes       *string `json:"oidc_scopes,omitempty"`

	SAMLIDPMetadataURL *string `json:"saml_idp_metadata_url,omitempty"`
	SAMLIDPMetadataXML *string `json:"saml_idp_metadata_xml,omitempty"`
	SAMLIDPSSOURL      *string `json:"saml_idp_sso_url,omitempty"`
	SAMLIDPCert        *string `json:"saml_idp_cert,omitempty"`
	SAMLSPEntityID     *string `json:"saml_sp_entity_id,omitempty"`
	SAMLNameIDFormat   *string `json:"saml_name_id_format,omitempty"`

	LDAPHost         *string `json:"ldap_host,omitempty"`
	LDAPPort         *int    `json:"ldap_port,omitempty"`
	LDAPUseTLS       *bool   `json:"ldap_use_tls,omitempty"`
	LDAPBindDN       *string `json:"ldap_bind_dn,omitempty"`
	LDAPBindPassword *string `json:"ldap_bind_password,omitempty"`
	LDAPBaseDN       *string `json:"ldap_base_dn,omitempty"`
	LDAPUserFilter   *string `json:"ldap_user_filter,omitempty"`
	LDAPEmailAttr    *string `json:"ldap_email_attr,omitempty"`
	LDAPNameAttr     *string `json:"ldap_name_attr,omitempty"`
	LDAPUsernameAttr *string `json:"ldap_username_attr,omitempty"`
}

type DiscoverResponse struct {
	Domain     string `json:"domain"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	EnforceSSO bool   `json:"enforce_sso"`
}

type ConfigResponse struct {
	ID         int64  `json:"id"`
	Domain     string `json:"domain"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	IsEnabled  bool   `json:"is_enabled"`
	EnforceSSO bool   `json:"enforce_sso"`

	OIDCIssuerURL string `json:"oidc_issuer_url,omitempty"`
	OIDCClientID  string `json:"oidc_client_id,omitempty"`
	OIDCScopes    string `json:"oidc_scopes,omitempty"`

	SAMLIDPMetadataURL string `json:"saml_idp_metadata_url,omitempty"`
	SAMLIDPSSOURL      string `json:"saml_idp_sso_url,omitempty"`
	SAMLSPEntityID     string `json:"saml_sp_entity_id,omitempty"`
	SAMLNameIDFormat   string `json:"saml_name_id_format,omitempty"`

	LDAPHost         string `json:"ldap_host,omitempty"`
	LDAPPort         *int   `json:"ldap_port,omitempty"`
	LDAPUseTLS       *bool  `json:"ldap_use_tls,omitempty"`
	LDAPBindDN       string `json:"ldap_bind_dn,omitempty"`
	LDAPBaseDN       string `json:"ldap_base_dn,omitempty"`
	LDAPUserFilter   string `json:"ldap_user_filter,omitempty"`
	LDAPEmailAttr    string `json:"ldap_email_attr,omitempty"`
	LDAPNameAttr     string `json:"ldap_name_attr,omitempty"`
	LDAPUsernameAttr string `json:"ldap_username_attr,omitempty"`

	CreatedBy *int64 `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

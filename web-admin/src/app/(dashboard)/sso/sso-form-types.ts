import type { CreateSSOConfigRequest } from "@/lib/api/sso";

export type UpdateField = (field: keyof CreateSSOConfigRequest, value: unknown) => void;

export interface ProtocolSectionProps {
  form: CreateSSOConfigRequest;
  update: UpdateField;
  isEdit: boolean;
}

export const defaultForm: CreateSSOConfigRequest = {
  domain: "",
  name: "",
  protocol: "oidc",
  is_enabled: true,
  enforce_sso: false,
  oidc_issuer_url: "",
  oidc_client_id: "",
  oidc_client_secret: "",
  oidc_scopes: "openid profile email",
  saml_idp_metadata_url: "",
  saml_idp_sso_url: "",
  saml_idp_cert: "",
  saml_sp_entity_id: "",
  saml_name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  ldap_host: "",
  ldap_port: 389,
  ldap_use_tls: false,
  ldap_bind_dn: "",
  ldap_bind_password: "",
  ldap_base_dn: "",
  ldap_user_filter: "(uid=%s)",
  ldap_email_attr: "mail",
  ldap_name_attr: "cn",
  ldap_username_attr: "uid",
};

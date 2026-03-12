import { apiClient } from "./base";

export type SSOProtocol = "oidc" | "saml" | "ldap";

export interface SSOConfig {
  id: number;
  domain: string;
  name: string;
  protocol: SSOProtocol;
  is_enabled: boolean;
  enforce_sso: boolean;
  // OIDC
  oidc_issuer_url?: string;
  oidc_client_id?: string;
  oidc_scopes?: string;
  // SAML
  saml_idp_metadata_url?: string;
  saml_idp_sso_url?: string;
  saml_sp_entity_id?: string;
  saml_name_id_format?: string;
  // LDAP
  ldap_host?: string;
  ldap_port?: number;
  ldap_use_tls?: boolean;
  ldap_bind_dn?: string;
  ldap_base_dn?: string;
  ldap_user_filter?: string;
  ldap_email_attr?: string;
  ldap_name_attr?: string;
  ldap_username_attr?: string;
  created_at: string;
  updated_at: string;
}

export interface SSOConfigListParams {
  search?: string;
  protocol?: SSOProtocol;
  is_enabled?: boolean;
  page?: number;
  page_size?: number;
}

export interface CreateSSOConfigRequest {
  domain: string;
  name: string;
  protocol: SSOProtocol;
  is_enabled?: boolean;
  enforce_sso?: boolean;
  // OIDC
  oidc_issuer_url?: string;
  oidc_client_id?: string;
  oidc_client_secret?: string;
  oidc_scopes?: string;
  // SAML
  saml_idp_metadata_url?: string;
  saml_idp_sso_url?: string;
  saml_idp_cert?: string;
  saml_sp_entity_id?: string;
  saml_name_id_format?: string;
  // LDAP
  ldap_host?: string;
  ldap_port?: number;
  ldap_use_tls?: boolean;
  ldap_bind_dn?: string;
  ldap_bind_password?: string;
  ldap_base_dn?: string;
  ldap_user_filter?: string;
  ldap_email_attr?: string;
  ldap_name_attr?: string;
  ldap_username_attr?: string;
}

export type UpdateSSOConfigRequest = Partial<CreateSSOConfigRequest>;

export interface SSOTestResult {
  success: boolean;
  message?: string;
  error?: string;
  details?: Record<string, unknown>;
}

export async function listSSOConfigs(params?: SSOConfigListParams): Promise<{ data: SSOConfig[]; total: number }> {
  const queryParams: Record<string, string | number | undefined> = {};
  if (params) {
    if (params.search) queryParams.search = params.search;
    if (params.protocol) queryParams.protocol = params.protocol;
    if (params.is_enabled !== undefined) queryParams.is_enabled = params.is_enabled ? "true" : "false";
    if (params.page) queryParams.page = params.page;
    if (params.page_size) queryParams.page_size = params.page_size;
  }
  return apiClient.get<{ data: SSOConfig[]; total: number }>("/sso/configs", queryParams);
}

export async function getSSOConfig(id: number): Promise<SSOConfig> {
  return apiClient.get<SSOConfig>(`/sso/configs/${id}`);
}

export async function createSSOConfig(data: CreateSSOConfigRequest): Promise<SSOConfig> {
  return apiClient.post<SSOConfig>("/sso/configs", data);
}

export async function updateSSOConfig(id: number, data: UpdateSSOConfigRequest): Promise<SSOConfig> {
  return apiClient.put<SSOConfig>(`/sso/configs/${id}`, data);
}

export async function deleteSSOConfig(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/sso/configs/${id}`);
}

export async function enableSSOConfig(id: number): Promise<SSOConfig> {
  return apiClient.post<SSOConfig>(`/sso/configs/${id}/enable`);
}

export async function disableSSOConfig(id: number): Promise<SSOConfig> {
  return apiClient.post<SSOConfig>(`/sso/configs/${id}/disable`);
}

export async function testSSOConfig(id: number): Promise<SSOTestResult> {
  return apiClient.post<SSOTestResult>(`/sso/configs/${id}/test`);
}

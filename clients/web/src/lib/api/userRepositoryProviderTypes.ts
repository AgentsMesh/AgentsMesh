export interface RepositoryProviderData {
  id: number;
  user_id?: number;
  provider_type: string;
  name: string;
  base_url: string;
  client_id?: string;
  has_client_id: boolean;
  has_bot_token: boolean;
  has_identity: boolean;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProviderRepositoryData {
  id: string;
  name: string;
  slug: string;
  description: string;
  default_branch: string;
  visibility: string;
  http_clone_url: string;
  ssh_clone_url: string;
  web_url: string;
}

export interface CreateRepositoryProviderRequest {
  provider_type: string;
  name: string;
  base_url: string;
  client_id?: string;
  client_secret?: string;
  bot_token?: string;
}

export interface UpdateRepositoryProviderRequest {
  name?: string;
  base_url?: string;
  client_id?: string;
  client_secret?: string;
  bot_token?: string;
  is_active?: boolean;
}

// Repository ViewModels — UI-side projection of `proto.repository.v1` types.
// Owned in this zero-dep contract layer so web (fromProtoRepository) and desktop
// (electron-adapter projection) share one definition.

export interface WebhookStatus {
  registered: boolean;
  webhook_id?: string;
  webhook_url?: string;
  events?: string[];
  is_active: boolean;
  needs_manual_setup: boolean;
  last_error?: string;
  registered_at?: string;
}

export interface WebhookResult {
  repo_id: number;
  registered: boolean;
  webhook_id?: string;
  needs_manual_setup: boolean;
  manual_webhook_url?: string;
  manual_webhook_secret?: string;
  error?: string;
}

export interface WebhookSecretResponse {
  webhook_url: string;
  webhook_secret: string;
  events: string[];
}

export interface RepositoryData {
  id: number;
  organization_id: number;
  provider_type: string;
  provider_base_url: string;
  http_clone_url?: string;
  ssh_clone_url?: string;
  external_id: string;
  name: string;
  slug: string;
  default_branch: string;
  ticket_prefix?: string;
  visibility: string;
  imported_by_user_id?: number;
  is_active: boolean;
  webhook_config?: {
    id: string;
    url: string;
    events: string[];
    is_active: boolean;
    needs_manual_setup: boolean;
    last_error?: string;
    created_at?: string;
  };
  created_at: string;
  updated_at: string;
}

export interface CreateRepositoryRequest {
  provider_type: string;
  provider_base_url: string;
  http_clone_url?: string;
  ssh_clone_url?: string;
  external_id: string;
  name: string;
  slug: string;
  default_branch?: string;
  ticket_prefix?: string;
  visibility?: string;
}

export interface UpdateRepositoryRequest {
  name?: string;
  default_branch?: string;
  ticket_prefix?: string;
  is_active?: boolean;
  http_clone_url?: string;
  ssh_clone_url?: string;
}

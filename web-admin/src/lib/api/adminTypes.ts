export interface DashboardStats {
  total_users: number;
  active_users: number;
  total_organizations: number;
  total_runners: number;
  online_runners: number;
  total_pods: number;
  active_pods: number;
  total_subscriptions: number;
  active_subscriptions: number;
  new_users_today: number;
  new_users_this_week: number;
  new_users_this_month: number;
}

export interface User {
  id: number;
  email: string;
  username: string;
  name: string | null;
  avatar_url: string | null;
  is_active: boolean;
  is_system_admin: boolean;
  is_email_verified: boolean;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface UserListParams {
  search?: string;
  is_active?: boolean;
  is_admin?: boolean;
  page?: number;
  page_size?: number;
}

export interface Organization {
  id: number;
  name: string;
  slug: string;
  description: string | null;
  logo_url: string | null;
  subscription_plan?: string;
  subscription_status?: string;
  created_at: string;
  updated_at: string;
}

export interface OrganizationMember {
  id: number;
  user_id: number;
  org_id: number;
  role: string;
  created_at: string;
  joined_at?: string;
  user?: {
    id: number;
    email: string;
    username: string;
    name: string | null;
    avatar_url: string | null;
  };
}

export interface OrganizationListParams {
  search?: string;
  page?: number;
  page_size?: number;
}

export interface Runner {
  id: number;
  organization_id: number;
  node_id: string;
  description: string | null;
  status: string;
  is_enabled: boolean;
  runner_version: string | null;
  current_pods: number;
  max_concurrent_pods: number;
  available_agents: string[];
  host_info: Record<string, unknown> | null;
  last_heartbeat: string | null;
  created_at: string;
  updated_at: string;
  organization?: {
    id: number;
    name: string;
    slug: string;
  };
}

export interface RunnerListParams {
  search?: string;
  status?: string;
  org_id?: number;
  page?: number;
  page_size?: number;
}

export interface AuditLog {
  id: number;
  admin_user_id: number;
  action: string;
  target_type: string;
  target_id: number;
  old_data: string | null;
  new_data: string | null;
  ip_address: string | null;
  user_agent: string | null;
  created_at: string;
  admin_user?: {
    id: number;
    email: string;
    username: string;
    name: string | null;
    avatar_url: string | null;
  };
}

export interface AuditLogListParams {
  admin_user_id?: number;
  action?: string;
  target_type?: string;
  target_id?: number;
  start_time?: string;
  end_time?: string;
  page?: number;
  page_size?: number;
}

export type PromoCodeType = "media" | "partner" | "campaign" | "internal" | "referral";

export interface PromoCode {
  id: number;
  code: string;
  name: string;
  description: string;
  type: PromoCodeType;
  plan_name: string;
  duration_months: number;
  max_uses: number | null;
  used_count: number;
  max_uses_per_org: number;
  starts_at: string;
  expires_at: string | null;
  is_active: boolean;
  created_by_id: number | null;
  created_at: string;
  updated_at: string;
}

export interface PromoCodeListParams {
  search?: string;
  type?: PromoCodeType;
  plan_name?: string;
  is_active?: boolean;
  page?: number;
  page_size?: number;
}

export interface CreatePromoCodeRequest {
  code: string;
  name: string;
  description?: string;
  type: PromoCodeType;
  plan_name: string;
  duration_months: number;
  max_uses?: number;
  max_uses_per_org?: number;
  starts_at?: string;
  expires_at?: string;
}

export interface UpdatePromoCodeRequest {
  name?: string;
  description?: string;
  max_uses?: number;
  max_uses_per_org?: number;
  expires_at?: string;
}

export interface PromoCodeRedemption {
  id: number;
  promo_code_id: number;
  organization_id: number;
  user_id: number;
  plan_name: string;
  duration_months: number;
  new_period_end: string;
  ip_address: string | null;
  created_at: string;
  user?: User;
  organization?: Organization;
}

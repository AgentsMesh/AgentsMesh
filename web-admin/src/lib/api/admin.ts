import { apiClient, PaginatedResponse } from "./base";
import { AdminUser } from "@/stores/auth";

// Dashboard
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

export async function getDashboardStats(): Promise<DashboardStats> {
  return apiClient.get<DashboardStats>("/dashboard/stats");
}

// Users
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

export async function listUsers(params?: UserListParams): Promise<PaginatedResponse<User>> {
  return apiClient.get<PaginatedResponse<User>>("/users", params as Record<string, string | number | undefined>);
}

export async function getUser(id: number): Promise<User> {
  return apiClient.get<User>(`/users/${id}`);
}

export async function updateUser(id: number, data: { name?: string; username?: string; email?: string }): Promise<User> {
  return apiClient.put<User>(`/users/${id}`, data);
}

export async function disableUser(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/disable`);
}

export async function enableUser(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/enable`);
}

export async function grantAdmin(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/grant-admin`);
}

export async function revokeAdmin(id: number): Promise<User> {
  return apiClient.post<User>(`/users/${id}/revoke-admin`);
}

// Organizations
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

export async function listOrganizations(params?: OrganizationListParams): Promise<PaginatedResponse<Organization>> {
  return apiClient.get<PaginatedResponse<Organization>>("/organizations", params as Record<string, string | number | undefined>);
}

export async function getOrganization(id: number): Promise<Organization> {
  return apiClient.get<Organization>(`/organizations/${id}`);
}

export async function getOrganizationMembers(id: number): Promise<{ organization: Organization; members: OrganizationMember[] }> {
  return apiClient.get<{ organization: Organization; members: OrganizationMember[] }>(`/organizations/${id}/members`);
}

export async function deleteOrganization(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/organizations/${id}`);
}

// Runners
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

export async function listRunners(params?: RunnerListParams): Promise<PaginatedResponse<Runner>> {
  return apiClient.get<PaginatedResponse<Runner>>("/runners", params as Record<string, string | number | undefined>);
}

export async function getRunner(id: number): Promise<Runner> {
  return apiClient.get<Runner>(`/runners/${id}`);
}

export async function disableRunner(id: number): Promise<Runner> {
  return apiClient.post<Runner>(`/runners/${id}/disable`);
}

export async function enableRunner(id: number): Promise<Runner> {
  return apiClient.post<Runner>(`/runners/${id}/enable`);
}

export async function deleteRunner(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/runners/${id}`);
}

// Audit Logs
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

export async function listAuditLogs(params?: AuditLogListParams): Promise<PaginatedResponse<AuditLog>> {
  return apiClient.get<PaginatedResponse<AuditLog>>("/audit-logs", params as Record<string, string | number | undefined>);
}

// Promo Codes
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

export async function listPromoCodes(params?: PromoCodeListParams): Promise<PaginatedResponse<PromoCode>> {
  // Convert boolean to string for API compatibility
  const queryParams: Record<string, string | number | undefined> = {};
  if (params) {
    if (params.search) queryParams.search = params.search;
    if (params.type) queryParams.type = params.type;
    if (params.plan_name) queryParams.plan_name = params.plan_name;
    if (params.is_active !== undefined) queryParams.is_active = params.is_active ? "true" : "false";
    if (params.page) queryParams.page = params.page;
    if (params.page_size) queryParams.page_size = params.page_size;
  }
  return apiClient.get<PaginatedResponse<PromoCode>>("/promo-codes", queryParams);
}

export async function getPromoCode(id: number): Promise<PromoCode> {
  return apiClient.get<PromoCode>(`/promo-codes/${id}`);
}

export async function createPromoCode(data: CreatePromoCodeRequest): Promise<PromoCode> {
  return apiClient.post<PromoCode>("/promo-codes", data);
}

export async function updatePromoCode(id: number, data: UpdatePromoCodeRequest): Promise<PromoCode> {
  return apiClient.put<PromoCode>(`/promo-codes/${id}`, data);
}

export async function activatePromoCode(id: number): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(`/promo-codes/${id}/activate`);
}

export async function deactivatePromoCode(id: number): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(`/promo-codes/${id}/deactivate`);
}

export async function deletePromoCode(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/promo-codes/${id}`);
}

export async function listPromoCodeRedemptions(id: number, params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<PromoCodeRedemption>> {
  return apiClient.get<PaginatedResponse<PromoCodeRedemption>>(`/promo-codes/${id}/redemptions`, params as Record<string, string | number | undefined>);
}

// Subscriptions
export interface SubscriptionPlan {
  id: number;
  name: string;
  display_name: string;
  price_per_seat_monthly: number;
  price_per_seat_yearly: number;
  included_pod_minutes: number;
  max_users: number;
  max_runners: number;
  max_concurrent_pods: number;
  max_repositories: number;
  features: Record<string, unknown>;
  is_active: boolean;
}

export interface SeatUsage {
  total_seats: number;
  used_seats: number;
  available_seats: number;
  max_seats: number;
  can_add_seats: boolean;
}

export interface Subscription {
  id: number;
  organization_id: number;
  plan_id: number;
  status: string;
  billing_cycle: string;
  current_period_start: string;
  current_period_end: string;
  auto_renew: boolean;
  seat_count: number;
  cancel_at_period_end: boolean;
  custom_quotas: Record<string, number> | null;
  created_at: string;
  updated_at: string;
  // Payment info (reference only)
  payment_provider?: string;
  payment_method?: string;
  has_stripe: boolean;
  has_alipay: boolean;
  has_wechat: boolean;
  has_lemonsqueezy: boolean;
  // Optional fields
  canceled_at?: string;
  frozen_at?: string;
  downgrade_to_plan?: string;
  next_billing_cycle?: string;
  // Nested
  plan?: SubscriptionPlan;
  seat_usage?: SeatUsage;
}

export async function getOrganizationSubscription(orgId: number): Promise<Subscription> {
  return apiClient.get<Subscription>(`/organizations/${orgId}/subscription`);
}

export async function getSubscriptionPlans(orgId: number): Promise<{ data: SubscriptionPlan[] }> {
  return apiClient.get<{ data: SubscriptionPlan[] }>(`/organizations/${orgId}/subscription/plans`);
}

export async function updateSubscriptionPlan(orgId: number, planName: string): Promise<Subscription> {
  return apiClient.put<Subscription>(`/organizations/${orgId}/subscription/plan`, { plan_name: planName });
}

export async function updateSubscriptionSeats(orgId: number, seatCount: number): Promise<Subscription> {
  return apiClient.put<Subscription>(`/organizations/${orgId}/subscription/seats`, { seat_count: seatCount });
}

export async function updateSubscriptionCycle(orgId: number, billingCycle: string): Promise<Subscription> {
  return apiClient.put<Subscription>(`/organizations/${orgId}/subscription/cycle`, { billing_cycle: billingCycle });
}

export async function freezeSubscription(orgId: number): Promise<Subscription> {
  return apiClient.post<Subscription>(`/organizations/${orgId}/subscription/freeze`);
}

export async function unfreezeSubscription(orgId: number): Promise<Subscription> {
  return apiClient.post<Subscription>(`/organizations/${orgId}/subscription/unfreeze`);
}

export async function cancelSubscription(orgId: number): Promise<Subscription> {
  return apiClient.post<Subscription>(`/organizations/${orgId}/subscription/cancel`);
}

export async function renewSubscription(orgId: number, months: number): Promise<Subscription> {
  return apiClient.post<Subscription>(`/organizations/${orgId}/subscription/renew`, { months });
}

export async function setSubscriptionAutoRenew(orgId: number, autoRenew: boolean): Promise<Subscription> {
  return apiClient.put<Subscription>(`/organizations/${orgId}/subscription/auto-renew`, { auto_renew: autoRenew });
}

export async function setSubscriptionQuota(orgId: number, resource: string, limit: number): Promise<Subscription> {
  return apiClient.put<Subscription>(`/organizations/${orgId}/subscription/quotas`, { resource, limit });
}

// Relays
export interface RelayInfo {
  id: string;
  url: string;
  internal_url?: string;
  region: string;
  capacity: number;
  connections: number;
  cpu_usage: number;
  memory_usage: number;
  last_heartbeat: string;
  healthy: boolean;
}

export interface ActiveSession {
  pod_key: string;
  session_id: string;
  relay_url: string;
  relay_id: string;
  created_at: string;
  expire_at: string;
}

export interface RelayStats {
  total_relays: number;
  healthy_relays: number;
  total_connections: number;
  active_sessions: number;
}

export interface RelayListResponse {
  data: RelayInfo[];
  total: number;
}

export interface SessionListResponse {
  data: ActiveSession[];
  total: number;
}

export interface RelayDetailResponse {
  relay: RelayInfo;
  session_count: number;
  sessions: ActiveSession[];
}

export async function listRelays(): Promise<RelayListResponse> {
  return apiClient.get<RelayListResponse>("/relays");
}

export async function getRelayStats(): Promise<RelayStats> {
  return apiClient.get<RelayStats>("/relays/stats");
}

export async function getRelay(id: string): Promise<RelayDetailResponse> {
  return apiClient.get<RelayDetailResponse>(`/relays/${encodeURIComponent(id)}`);
}

export async function forceUnregisterRelay(id: string, migrateSessions: boolean = false): Promise<{ status: string; relay_id: string; affected_sessions: number }> {
  return apiClient.delete<{ status: string; relay_id: string; affected_sessions: number }>(`/relays/${encodeURIComponent(id)}`, { migrate_sessions: migrateSessions });
}

export async function listSessions(relayId?: string): Promise<SessionListResponse> {
  const params = relayId ? { relay_id: relayId } : undefined;
  return apiClient.get<SessionListResponse>("/relays/sessions", params);
}

export async function migrateSession(podKey: string, targetRelay: string): Promise<{ status: string; from_relay: string; to_relay: string }> {
  return apiClient.post<{ status: string; from_relay: string; to_relay: string }>("/relays/sessions/migrate", { pod_key: podKey, target_relay: targetRelay });
}

export async function bulkMigrateSessions(sourceRelay: string, targetRelay: string): Promise<{ status: string; total: number; migrated: number; failed: number }> {
  return apiClient.post<{ status: string; total: number; migrated: number; failed: number }>("/relays/sessions/bulk-migrate", { source_relay: sourceRelay, target_relay: targetRelay });
}

// Auth
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: AdminUser;
}

export async function login(req: LoginRequest): Promise<LoginResponse> {
  return apiClient.post<LoginResponse>("/auth/login", req);
}

export async function getCurrentAdmin(): Promise<AdminUser> {
  return apiClient.get<AdminUser>("/me");
}

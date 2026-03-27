import { AdminUser } from "@/stores/auth";

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
  payment_provider?: string;
  payment_method?: string;
  has_stripe: boolean;
  has_alipay: boolean;
  has_wechat: boolean;
  has_lemonsqueezy: boolean;
  canceled_at?: string;
  frozen_at?: string;
  downgrade_to_plan?: string;
  next_billing_cycle?: string;
  plan?: SubscriptionPlan;
  seat_usage?: SeatUsage;
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

// Skill Registries
export interface SkillRegistry {
  id: number;
  organization_id: number | null;
  repository_url: string;
  branch: string;
  source_type: string;
  sync_status: string;
  sync_error: string;
  skill_count: number;
  last_synced_at: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateSkillRegistryRequest {
  repository_url: string;
  branch?: string;
  source_type?: string;
}

// Support Tickets
export interface SupportTicket {
  id: number;
  user_id: number;
  title: string;
  category: string;
  status: string;
  priority: string;
  assigned_admin_id?: number;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
  user?: { id: number; name: string; email: string; avatar_url?: string };
  assigned_admin?: { id: number; name: string; email: string };
}

export interface SupportTicketMessage {
  id: number;
  ticket_id: number;
  user_id: number;
  content: string;
  is_admin_reply: boolean;
  created_at: string;
  user?: { id: number; name: string; email: string; avatar_url?: string };
  attachments?: SupportTicketAttachment[];
}

export interface SupportTicketAttachment {
  id: number;
  original_name: string;
  mime_type: string;
  size: number;
}

export interface SupportTicketStats {
  total: number;
  open: number;
  in_progress: number;
  resolved: number;
  closed: number;
}

export interface SupportTicketListParams {
  search?: string;
  status?: string;
  category?: string;
  priority?: string;
  page?: number;
  page_size?: number;
}

export interface SupportTicketDetail {
  ticket: SupportTicket;
  messages: SupportTicketMessage[];
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

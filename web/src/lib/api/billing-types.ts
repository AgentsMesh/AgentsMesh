// Billing types

export interface SubscriptionPlan {
  id: number;
  name: string;
  display_name: string;
  price_per_seat_monthly: number;
  price_per_seat_yearly: number;
  included_pod_minutes: number;
  price_per_extra_minute: number;
  max_users: number;
  max_runners: number;
  max_repositories: number;
  max_concurrent_pods: number;
  features: Record<string, unknown>;
  is_active: boolean;
  stripe_price_id_monthly?: string;
  stripe_price_id_yearly?: string;
}

// Multi-currency price for a plan
export interface PlanPrice {
  id: number;
  plan_id: number;
  currency: string; // USD, CNY
  price_monthly: number;
  price_yearly: number;
  stripe_price_id_monthly?: string;
  stripe_price_id_yearly?: string;
  plan?: SubscriptionPlan;
}

// Plan with price in specific currency
export interface PlanWithPrice {
  plan: SubscriptionPlan;
  price: PlanPrice;
}

// Currency type
export type Currency = "USD" | "CNY";

export interface UsageOverview {
  pod_minutes: number;
  included_pod_minutes: number;
  users: number;
  max_users: number;
  runners: number;
  max_runners: number;
  repositories: number;
  max_repositories: number;
}

export interface BillingOverview {
  plan: SubscriptionPlan;
  status: string;
  billing_cycle: string;
  current_period_start: string;
  current_period_end: string;
  usage: UsageOverview;
  cancel_at_period_end?: boolean;
  seat_count?: number;
  downgrade_to_plan?: string;
}

export interface Subscription {
  id: number;
  organization_id: number;
  plan_id: number;
  status: string;
  billing_cycle: string;
  current_period_start: string;
  current_period_end: string;
  plan?: SubscriptionPlan;
  payment_provider?: string;
  payment_method?: string;
  auto_renew: boolean;
  cancel_at_period_end: boolean;
  seat_count: number;
  stripe_customer_id?: string;
  stripe_subscription_id?: string;
  lemonsqueezy_customer_id?: string;
  lemonsqueezy_subscription_id?: string;
  frozen_at?: string;
  downgrade_to_plan?: string;
  next_billing_cycle?: string;
}

// Checkout types
export type OrderType = "subscription" | "seat_purchase" | "plan_upgrade" | "renewal";
export type BillingCycle = "monthly" | "yearly";
export type PaymentProvider = "stripe" | "lemonsqueezy" | "alipay" | "wechat";

export interface CheckoutRequest {
  order_type: OrderType;
  plan_name?: string;
  billing_cycle?: BillingCycle;
  seats?: number;
  provider?: PaymentProvider;
  success_url: string;
  cancel_url: string;
}

export interface CheckoutResponse {
  order_no: string;
  session_id: string;
  session_url: string;
  qr_code_url?: string;
  expires_at: string;
  provider?: PaymentProvider;
}

export interface CheckoutStatus {
  order_no: string;
  status: string;
  order_type: string;
  amount: number;
  currency: string;
  created_at: string;
  paid_at?: string;
}

// Seat types
export interface SeatUsage {
  total_seats: number;
  used_seats: number;
  available_seats: number;
  max_seats: number;
  can_add_seats: boolean;
}

// Invoice types
export interface Invoice {
  id: number;
  organization_id: number;
  invoice_no: string;
  order_no?: string;
  amount: number;
  tax_amount: number;
  total_amount: number;
  currency: string;
  status: string;
  billing_period_start: string;
  billing_period_end: string;
  paid_at?: string;
  created_at: string;
}

// Deployment info
export interface DeploymentInfo {
  deployment_type: "global" | "cn" | "onpremise";
  available_providers: string[];
}

// Public pricing info (no auth required)
export interface PublicPlanPricing {
  name: string;
  display_name: string;
  price_monthly: number;
  price_yearly: number;
  max_users: number;
  max_runners: number;
  max_repositories: number;
  max_concurrent_pods: number;
}

export interface PublicPricingResponse {
  deployment_type: string;
  currency: Currency;
  plans: PublicPlanPricing[];
}

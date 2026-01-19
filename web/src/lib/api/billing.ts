import { request, orgPath } from "./base";

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
  frozen_at?: string;
  downgrade_to_plan?: string;
  next_billing_cycle?: string;
}

// Checkout types
export type OrderType = "subscription" | "seat_purchase" | "plan_upgrade" | "renewal";
export type BillingCycle = "monthly" | "yearly";
export type PaymentProvider = "stripe" | "alipay" | "wechat";

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

// Billing API
export const billingApi = {
  // Get billing overview
  getOverview: () =>
    request<{ overview: BillingOverview }>(orgPath("/billing/overview")),

  // Get subscription
  getSubscription: () =>
    request<{ subscription: Subscription }>(orgPath("/billing/subscription")),

  // Create subscription (for free plans, use createCheckout for paid plans)
  createSubscription: (planName: string, billingCycle?: string) =>
    request<{ subscription: Subscription }>(orgPath("/billing/subscription"), {
      method: "POST",
      body: { plan_name: planName, billing_cycle: billingCycle || "monthly" },
    }),

  // Update subscription (for free plans, use createCheckout for paid upgrades)
  updateSubscription: (planName: string) =>
    request<{ subscription: Subscription }>(orgPath("/billing/subscription"), {
      method: "PUT",
      body: { plan_name: planName },
    }),

  // Cancel subscription
  cancelSubscription: () =>
    request<{ message: string }>(orgPath("/billing/subscription"), {
      method: "DELETE",
    }),

  // List available plans
  listPlans: () =>
    request<{ plans: SubscriptionPlan[] }>(orgPath("/billing/plans")),

  // Get usage
  getUsage: (type?: string) => {
    const params = type ? `?type=${type}` : "";
    return request<{ usage: UsageOverview | number; type?: string }>(
      `${orgPath("/billing/usage")}${params}`
    );
  },

  // Check quota
  checkQuota: (resource: string, amount?: number) => {
    const params = new URLSearchParams({ resource });
    if (amount) params.append("amount", String(amount));
    return request<{ available: boolean }>(`${orgPath("/billing/quota/check")}?${params.toString()}`);
  },

  // ===========================================
  // Payment Checkout APIs
  // ===========================================

  // Create checkout session
  createCheckout: (req: CheckoutRequest) =>
    request<CheckoutResponse>(orgPath("/billing/checkout"), {
      method: "POST",
      body: req,
    }),

  // Get checkout/order status
  getCheckoutStatus: (orderNo: string) =>
    request<CheckoutStatus>(orgPath(`/billing/checkout/${orderNo}`)),

  // ===========================================
  // Subscription Management APIs
  // ===========================================

  // Request subscription cancellation
  requestCancelSubscription: (immediate: boolean = false) =>
    request<{ message: string; current_period_end?: string }>(
      orgPath("/billing/subscription/cancel"),
      {
        method: "POST",
        body: { immediate },
      }
    ),

  // Reactivate subscription (undo pending cancellation)
  reactivateSubscription: () =>
    request<{ message: string; current_period_end?: string }>(
      orgPath("/billing/subscription/reactivate"),
      {
        method: "POST",
      }
    ),

  // Change billing cycle (takes effect on next renewal)
  changeBillingCycle: (billingCycle: BillingCycle) =>
    request<{
      message: string;
      current_cycle: string;
      next_cycle: string;
      effective_date: string;
    }>(orgPath("/billing/subscription/change-cycle"), {
      method: "POST",
      body: { billing_cycle: billingCycle },
    }),

  // Update auto-renew setting
  updateAutoRenew: (autoRenew: boolean) =>
    request<{ subscription: Subscription; auto_renew: boolean }>(
      orgPath("/billing/subscription/auto-renew"),
      {
        method: "PUT",
        body: { auto_renew: autoRenew },
      }
    ),

  // ===========================================
  // Seat Management APIs
  // ===========================================

  // Get seat usage
  getSeatUsage: () =>
    request<SeatUsage>(orgPath("/billing/seats")),

  // Purchase additional seats
  purchaseSeats: (seats: number, successUrl: string, cancelUrl: string) =>
    request<CheckoutResponse>(orgPath("/billing/seats/purchase"), {
      method: "POST",
      body: {
        seats,
        success_url: successUrl,
        cancel_url: cancelUrl,
      },
    }),

  // ===========================================
  // Invoice APIs
  // ===========================================

  // List invoices
  listInvoices: (limit: number = 20, offset: number = 0) =>
    request<{ invoices: Invoice[] }>(
      orgPath(`/billing/invoices?limit=${limit}&offset=${offset}`)
    ),

  // ===========================================
  // Customer Portal APIs
  // ===========================================

  // Get Stripe customer portal URL
  getCustomerPortal: (returnUrl: string) =>
    request<{ url: string }>(orgPath("/billing/customer-portal"), {
      method: "POST",
      body: { return_url: returnUrl },
    }),

  // ===========================================
  // Deployment Info APIs
  // ===========================================

  // Get deployment type and available providers
  getDeploymentInfo: () =>
    request<DeploymentInfo>(orgPath("/billing/deployment")),
};

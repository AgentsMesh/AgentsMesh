import { request, orgPath } from "./base";
import type {
  SubscriptionPlan,
  PlanPrice,
  PlanWithPrice,
  Currency,
  UsageOverview,
  BillingOverview,
  Subscription,
  CheckoutRequest,
  CheckoutResponse,
  CheckoutStatus,
  BillingCycle,
  SeatUsage,
  Invoice,
  DeploymentInfo,
} from "./billing-types";

// Re-export types that consumers need
export type {
  SubscriptionPlan, PlanPrice, PlanWithPrice, Currency, UsageOverview,
  BillingOverview, Subscription, CheckoutRequest, CheckoutResponse,
  CheckoutStatus, BillingCycle, SeatUsage, Invoice, DeploymentInfo,
} from "./billing-types";

export type { OrderType, PaymentProvider, PublicPlanPricing, PublicPricingResponse } from "./billing-types";

// Re-export publicBillingApi from dedicated file
export { publicBillingApi } from "./billing-public";

// Billing API (authenticated, org-scoped)
export const billingApi = {
  getOverview: () =>
    request<{ overview: BillingOverview }>(orgPath("/billing/overview")),

  getSubscription: () =>
    request<{ subscription: Subscription }>(orgPath("/billing/subscription")),

  createSubscription: (planName: string, billingCycle?: string) =>
    request<{ subscription: Subscription }>(orgPath("/billing/subscription"), {
      method: "POST",
      body: { plan_name: planName, billing_cycle: billingCycle || "monthly" },
    }),

  updateSubscription: (planName: string) =>
    request<{ subscription: Subscription }>(orgPath("/billing/subscription"), {
      method: "PUT",
      body: { plan_name: planName },
    }),

  cancelSubscription: () =>
    request<{ message: string }>(orgPath("/billing/subscription"), { method: "DELETE" }),

  listPlans: () =>
    request<{ plans: SubscriptionPlan[] }>(orgPath("/billing/plans")),

  listPlansWithPrices: (currency: Currency = "USD") =>
    request<{ plans: PlanWithPrice[]; currency: string }>(
      orgPath(`/billing/plans/prices?currency=${currency}`)
    ),

  getPlanPrices: (planName: string, currency: Currency = "USD") =>
    request<{ price: PlanPrice; currency: string }>(
      orgPath(`/billing/plans/${planName}/prices?currency=${currency}`)
    ),

  getAllPlanPrices: (planName: string) =>
    request<{ prices: PlanPrice[] }>(orgPath(`/billing/plans/${planName}/all-prices`)),

  getUsage: (type?: string) => {
    const params = type ? `?type=${type}` : "";
    return request<{ usage: UsageOverview | number; type?: string }>(
      `${orgPath("/billing/usage")}${params}`
    );
  },

  checkQuota: (resource: string, amount?: number) => {
    const params = new URLSearchParams({ resource });
    if (amount) params.append("amount", String(amount));
    return request<{ available: boolean }>(`${orgPath("/billing/quota/check")}?${params.toString()}`);
  },

  createCheckout: (req: CheckoutRequest) =>
    request<CheckoutResponse>(orgPath("/billing/checkout"), { method: "POST", body: req }),

  getCheckoutStatus: (orderNo: string) =>
    request<CheckoutStatus>(orgPath(`/billing/checkout/${orderNo}`)),

  requestCancelSubscription: (immediate: boolean = false) =>
    request<{ message: string; current_period_end?: string }>(
      orgPath("/billing/subscription/cancel"), { method: "POST", body: { immediate } }
    ),

  reactivateSubscription: () =>
    request<{ message: string; current_period_end?: string }>(
      orgPath("/billing/subscription/reactivate"), { method: "POST" }
    ),

  upgradeSubscription: (planName: string) =>
    request<{ message: string; subscription: Subscription }>(
      orgPath("/billing/subscription/upgrade"), { method: "POST", body: { plan_name: planName } }
    ),

  changeBillingCycle: (billingCycle: BillingCycle) =>
    request<{ message: string; current_cycle: string; next_cycle: string; effective_date: string }>(
      orgPath("/billing/subscription/change-cycle"), { method: "POST", body: { billing_cycle: billingCycle } }
    ),

  updateAutoRenew: (autoRenew: boolean) =>
    request<{ subscription: Subscription; auto_renew: boolean }>(
      orgPath("/billing/subscription/auto-renew"), { method: "PUT", body: { auto_renew: autoRenew } }
    ),

  getSeatUsage: () =>
    request<SeatUsage>(orgPath("/billing/seats")),

  purchaseSeats: (seats: number) =>
    request<{ message: string; seats?: SeatUsage }>(orgPath("/billing/seats/purchase"), {
      method: "POST", body: { seats },
    }),

  listInvoices: (limit: number = 20, offset: number = 0) =>
    request<{ invoices: Invoice[] }>(orgPath(`/billing/invoices?limit=${limit}&offset=${offset}`)),

  getCustomerPortal: (returnUrl: string) =>
    request<{ url: string }>(orgPath("/billing/customer-portal"), {
      method: "POST", body: { return_url: returnUrl },
    }),

  getDeploymentInfo: () =>
    request<DeploymentInfo>(orgPath("/billing/deployment")),
};

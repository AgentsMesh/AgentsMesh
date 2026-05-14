// Domain ↔ proto converters for the SubscriptionAdminService Connect
// adapter. Kept separate from adminSubscriptions.ts so the public RPC
// surface stays under the 200-line cap.
import type {
  AdminSubscription as ProtoAdminSubscription,
  AdminSubscriptionEntity as ProtoEntity,
  AdminSubscriptionPlan as ProtoPlan,
  AdminSeatUsage as ProtoSeatUsage,
} from "@proto/billing/v1/billing_admin_pb";

import type { Subscription, SubscriptionPlan, SeatUsage } from "./adminTypesExtended";

export function planFromProto(p: ProtoPlan): SubscriptionPlan {
  return {
    id: Number(p.id),
    name: p.name,
    display_name: p.displayName,
    price_per_seat_monthly: p.pricePerSeatMonthly,
    price_per_seat_yearly: p.pricePerSeatYearly,
    included_pod_minutes: p.includedPodMinutes,
    max_users: p.maxUsers,
    max_runners: p.maxRunners,
    max_concurrent_pods: p.maxConcurrentPods,
    max_repositories: p.maxRepositories,
    features: {},
    is_active: p.isActive,
  };
}

function seatUsageFromProto(u: ProtoSeatUsage): SeatUsage {
  return {
    total_seats: u.totalSeats,
    used_seats: u.usedSeats,
    available_seats: u.availableSeats,
    max_seats: u.maxSeats,
    can_add_seats: u.canAddSeats,
  };
}

export function subscriptionFromProto(resp: ProtoAdminSubscription): Subscription {
  const e = resp.subscription as ProtoEntity;
  const sub: Subscription = {
    id: Number(e.id),
    organization_id: Number(e.organizationId),
    plan_id: Number(e.planId),
    status: e.status,
    billing_cycle: e.billingCycle,
    current_period_start: e.currentPeriodStart,
    current_period_end: e.currentPeriodEnd,
    auto_renew: e.autoRenew,
    seat_count: e.seatCount,
    cancel_at_period_end: e.cancelAtPeriodEnd,
    custom_quotas: resp.customQuotasJson
      ? (JSON.parse(resp.customQuotasJson) as Record<string, number>)
      : null,
    created_at: e.createdAt,
    updated_at: e.updatedAt,
    has_stripe: resp.hasStripe,
    has_alipay: resp.hasAlipay,
    has_wechat: resp.hasWechat,
    has_lemonsqueezy: resp.hasLemonsqueezy,
  };
  if (e.paymentProvider !== undefined) sub.payment_provider = e.paymentProvider;
  if (e.paymentMethod !== undefined) sub.payment_method = e.paymentMethod;
  if (e.canceledAt !== undefined) sub.canceled_at = e.canceledAt;
  if (e.frozenAt !== undefined) sub.frozen_at = e.frozenAt;
  if (e.downgradeToPlan !== undefined) sub.downgrade_to_plan = e.downgradeToPlan;
  if (e.nextBillingCycle !== undefined) sub.next_billing_cycle = e.nextBillingCycle;
  if (e.plan) sub.plan = planFromProto(e.plan);
  if (resp.seatUsage) sub.seat_usage = seatUsageFromProto(resp.seatUsage);
  return sub;
}

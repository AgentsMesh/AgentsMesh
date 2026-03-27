import { apiClient } from "./base";
import type { Subscription, SubscriptionPlan } from "./adminTypesExtended";

export async function getOrganizationSubscription(orgId: number): Promise<Subscription> {
  return apiClient.get<Subscription>(`/organizations/${orgId}/subscription`);
}

export async function getSubscriptionPlans(orgId: number): Promise<{ data: SubscriptionPlan[] }> {
  return apiClient.get<{ data: SubscriptionPlan[] }>(`/organizations/${orgId}/subscription/plans`);
}

export async function createSubscription(orgId: number, planName: string, months: number = 1): Promise<Subscription> {
  return apiClient.post<Subscription>(`/organizations/${orgId}/subscription/create`, { plan_name: planName, months });
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

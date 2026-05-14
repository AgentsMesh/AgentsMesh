// Connect-RPC adapter for proto.billing.v1.SubscriptionAdminService.
//
// Migrated from REST `/api/v1/admin/organizations/:id/subscription/*`.
// Keeps the existing TS return shapes (snake_case + number) so admin
// pages don't need to change. Proto types are camelCase + bigint; the
// adapter bridges the gap via adminSubscriptionsConvert.ts.
import {
  AdminSubscriptionSchema,
  CancelAdminSubscriptionRequestSchema,
  CreateAdminSubscriptionRequestSchema,
  FreezeAdminSubscriptionRequestSchema,
  GetAdminSubscriptionRequestSchema,
  ListAdminPlansRequestSchema,
  ListAdminPlansResponseSchema,
  RenewAdminSubscriptionRequestSchema,
  SetAdminAutoRenewRequestSchema,
  SetAdminCustomQuotaRequestSchema,
  SubscriptionAdminService,
  UnfreezeAdminSubscriptionRequestSchema,
  UpdateAdminCycleRequestSchema,
  UpdateAdminPlanRequestSchema,
  UpdateAdminSeatsRequestSchema,
} from "@proto/billing/v1/billing_admin_pb";

import { callConnect } from "@/lib/connect/transport";
import { planFromProto, subscriptionFromProto } from "./adminSubscriptionsConvert";
import type { Subscription, SubscriptionPlan } from "./adminTypesExtended";

const SERVICE = "proto.billing.v1.SubscriptionAdminService";
void SubscriptionAdminService;

export async function getOrganizationSubscription(orgId: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "GetSubscription",
    GetAdminSubscriptionRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId) },
  );
  return subscriptionFromProto(resp);
}

export async function getSubscriptionPlans(orgId: number): Promise<{ data: SubscriptionPlan[] }> {
  const resp = await callConnect(
    SERVICE,
    "ListPlans",
    ListAdminPlansRequestSchema,
    ListAdminPlansResponseSchema,
    { orgId: BigInt(orgId) },
  );
  return { data: resp.data.map(planFromProto) };
}

export async function createSubscription(
  orgId: number,
  planName: string,
  months: number = 1,
): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "CreateSubscription",
    CreateAdminSubscriptionRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), planName, months },
  );
  return subscriptionFromProto(resp);
}

export async function updateSubscriptionPlan(orgId: number, planName: string): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "UpdatePlan",
    UpdateAdminPlanRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), planName },
  );
  return subscriptionFromProto(resp);
}

export async function updateSubscriptionSeats(orgId: number, seatCount: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "UpdateSeats",
    UpdateAdminSeatsRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), seatCount },
  );
  return subscriptionFromProto(resp);
}

export async function updateSubscriptionCycle(orgId: number, billingCycle: string): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "UpdateCycle",
    UpdateAdminCycleRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), billingCycle },
  );
  return subscriptionFromProto(resp);
}

export async function freezeSubscription(orgId: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "Freeze",
    FreezeAdminSubscriptionRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId) },
  );
  return subscriptionFromProto(resp);
}

export async function unfreezeSubscription(orgId: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "Unfreeze",
    UnfreezeAdminSubscriptionRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId) },
  );
  return subscriptionFromProto(resp);
}

export async function cancelSubscription(orgId: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "Cancel",
    CancelAdminSubscriptionRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId) },
  );
  return subscriptionFromProto(resp);
}

export async function renewSubscription(orgId: number, months: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "Renew",
    RenewAdminSubscriptionRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), months },
  );
  return subscriptionFromProto(resp);
}

export async function setSubscriptionAutoRenew(orgId: number, autoRenew: boolean): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "SetAutoRenew",
    SetAdminAutoRenewRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), autoRenew },
  );
  return subscriptionFromProto(resp);
}

export async function setSubscriptionQuota(orgId: number, resource: string, limit: number): Promise<Subscription> {
  const resp = await callConnect(
    SERVICE,
    "SetCustomQuota",
    SetAdminCustomQuotaRequestSchema,
    AdminSubscriptionSchema,
    { orgId: BigInt(orgId), resource, limit },
  );
  return subscriptionFromProto(resp);
}

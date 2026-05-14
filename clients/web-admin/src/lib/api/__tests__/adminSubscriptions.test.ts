import { describe, it, expect, vi, beforeEach } from "vitest";

const mockCallConnect = vi.fn();
vi.mock("@/lib/connect/transport", () => ({
  callConnect: (...args: unknown[]) => mockCallConnect(...args),
  ConnectError: class ConnectError extends Error {},
}));

import {
  getOrganizationSubscription,
  getSubscriptionPlans,
  createSubscription,
  updateSubscriptionPlan,
  updateSubscriptionSeats,
  updateSubscriptionCycle,
  freezeSubscription,
  unfreezeSubscription,
  cancelSubscription,
  renewSubscription,
  setSubscriptionAutoRenew,
  setSubscriptionQuota,
} from "../adminSubscriptions";

function fakeAdminSubscription(id: number) {
  return {
    subscription: {
      id: BigInt(id),
      organizationId: 10n,
      planId: 1n,
      status: "active",
      billingCycle: "monthly",
      currentPeriodStart: "",
      currentPeriodEnd: "",
      autoRenew: true,
      seatCount: 1,
      cancelAtPeriodEnd: false,
      createdAt: "",
      updatedAt: "",
    },
    hasStripe: false,
    hasAlipay: false,
    hasWechat: false,
    hasLemonsqueezy: false,
  };
}

describe("Admin API - Subscriptions (Connect-RPC)", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getOrganizationSubscription calls GetSubscription with orgId", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(1));
    const sub = await getOrganizationSubscription(10);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.billing.v1.SubscriptionAdminService",
      "GetSubscription",
      expect.anything(),
      expect.anything(),
      { orgId: 10n },
    );
    expect(sub.id).toBe(1);
    expect(sub.organization_id).toBe(10);
  });

  it("getSubscriptionPlans calls ListPlans", async () => {
    mockCallConnect.mockResolvedValue({ data: [] });
    await getSubscriptionPlans(10);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.billing.v1.SubscriptionAdminService",
      "ListPlans",
      expect.anything(),
      expect.anything(),
      { orgId: 10n },
    );
  });

  it("createSubscription calls CreateSubscription with plan_name and months", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(2));
    await createSubscription(10, "pro", 6);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.billing.v1.SubscriptionAdminService",
      "CreateSubscription",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, planName: "pro", months: 6 },
    );
  });

  it("createSubscription defaults months to 1", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(2));
    await createSubscription(10, "pro");
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "CreateSubscription",
      expect.anything(),
      expect.anything(),
      expect.objectContaining({ months: 1 }),
    );
  });

  it("updateSubscriptionPlan calls UpdatePlan", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(3));
    await updateSubscriptionPlan(10, "enterprise");
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "UpdatePlan",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, planName: "enterprise" },
    );
  });

  it("updateSubscriptionSeats calls UpdateSeats", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(4));
    await updateSubscriptionSeats(10, 25);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "UpdateSeats",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, seatCount: 25 },
    );
  });

  it("updateSubscriptionCycle calls UpdateCycle", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(5));
    await updateSubscriptionCycle(10, "yearly");
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "UpdateCycle",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, billingCycle: "yearly" },
    );
  });

  it("freezeSubscription calls Freeze", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(6));
    await freezeSubscription(10);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "Freeze",
      expect.anything(),
      expect.anything(),
      { orgId: 10n },
    );
  });

  it("unfreezeSubscription calls Unfreeze", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(7));
    await unfreezeSubscription(10);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "Unfreeze",
      expect.anything(),
      expect.anything(),
      { orgId: 10n },
    );
  });

  it("cancelSubscription calls Cancel", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(8));
    await cancelSubscription(10);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "Cancel",
      expect.anything(),
      expect.anything(),
      { orgId: 10n },
    );
  });

  it("renewSubscription calls Renew with months", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(9));
    await renewSubscription(10, 12);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "Renew",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, months: 12 },
    );
  });

  it("setSubscriptionAutoRenew calls SetAutoRenew", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(10));
    await setSubscriptionAutoRenew(10, false);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "SetAutoRenew",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, autoRenew: false },
    );
  });

  it("setSubscriptionQuota calls SetCustomQuota", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminSubscription(11));
    await setSubscriptionQuota(10, "max_runners", 50);
    expect(mockCallConnect).toHaveBeenLastCalledWith(
      expect.anything(),
      "SetCustomQuota",
      expect.anything(),
      expect.anything(),
      { orgId: 10n, resource: "max_runners", limit: 50 },
    );
  });
});

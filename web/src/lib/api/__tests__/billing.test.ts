import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { getApiBaseUrl } from "@/lib/env";

// Get the expected API URL - must match getApiBaseUrl() logic used by base.ts
const EXPECTED_API_URL = getApiBaseUrl();

import { billingApi } from "../billing";
import type {
  BillingOverview,
  Subscription,
  PlanWithPrice,
  PlanPrice,
} from "../billing";

// Mock useAuthStore
const mockGetState = vi.fn();
vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => mockGetState(),
  },
}));

// Mock global fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe("billingApi", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetState.mockReturnValue({
      token: "test-token",
      currentOrg: { slug: "test-org" },
    });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("getOverview", () => {
    it("should fetch billing overview", async () => {
      const mockOverview: BillingOverview = {
        plan: {
          id: 1,
          name: "based",
          display_name: "Based Plan",
          price_per_seat_monthly: 9.9,
          price_per_seat_yearly: 99,
          included_pod_minutes: 100,
          price_per_extra_minute: 0,
          max_users: 1,
          max_runners: 1,
          max_repositories: 5,
          max_concurrent_pods: 5,
          features: {},
          is_active: true,
        },
        status: "active",
        billing_cycle: "monthly",
        current_period_start: "2026-01-01T00:00:00Z",
        current_period_end: "2026-02-01T00:00:00Z",
        usage: {
          pod_minutes: 50,
          included_pod_minutes: 100,
          users: 1,
          max_users: 1,
          runners: 1,
          max_runners: 1,
          repositories: 3,
          max_repositories: 5,
        },
        seat_count: 1,
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ overview: mockOverview })),
      });

      const result = await billingApi.getOverview();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/overview`,
        expect.objectContaining({
          method: "GET",
          headers: expect.objectContaining({
            Authorization: "Bearer test-token",
          }),
        })
      );
      expect(result.overview).toEqual(mockOverview);
    });
  });

  describe("getSubscription", () => {
    it("should fetch subscription details", async () => {
      const mockSubscription: Subscription = {
        id: 1,
        organization_id: 1,
        plan_id: 1,
        status: "active",
        billing_cycle: "monthly",
        current_period_start: "2026-01-01T00:00:00Z",
        current_period_end: "2026-02-01T00:00:00Z",
        auto_renew: true,
        cancel_at_period_end: false,
        seat_count: 1,
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ subscription: mockSubscription })),
      });

      const result = await billingApi.getSubscription();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/subscription`,
        expect.objectContaining({ method: "GET" })
      );
      expect(result.subscription).toEqual(mockSubscription);
    });
  });

  describe("listPlansWithPrices", () => {
    it("should fetch plans with USD prices by default", async () => {
      const mockPlans: PlanWithPrice[] = [
        {
          plan: {
            id: 1,
            name: "based",
            display_name: "Based Plan",
            price_per_seat_monthly: 9.9,
            price_per_seat_yearly: 99,
            included_pod_minutes: 100,
            price_per_extra_minute: 0,
            max_users: 1,
            max_runners: 1,
            max_repositories: 5,
            max_concurrent_pods: 5,
            features: {},
            is_active: true,
          },
          price: {
            id: 1,
            plan_id: 1,
            currency: "USD",
            price_monthly: 9.9,
            price_yearly: 99,
          },
        },
      ];

      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ plans: mockPlans, currency: "USD" })),
      });

      const result = await billingApi.listPlansWithPrices();

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/plans/prices?currency=USD`,
        expect.objectContaining({ method: "GET" })
      );
      expect(result.plans).toEqual(mockPlans);
      expect(result.currency).toBe("USD");
    });

    it("should fetch plans with CNY prices", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ plans: [], currency: "CNY" })),
      });

      await billingApi.listPlansWithPrices("CNY");

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/plans/prices?currency=CNY`,
        expect.anything()
      );
    });
  });

  describe("getPlanPrices", () => {
    it("should fetch prices for a specific plan", async () => {
      const mockPrice: PlanPrice = {
        id: 1,
        plan_id: 1,
        currency: "USD",
        price_monthly: 9.9,
        price_yearly: 99,
      };

      mockFetch.mockResolvedValue({
        ok: true,
        text: () =>
          Promise.resolve(JSON.stringify({ price: mockPrice, currency: "USD" })),
      });

      const result = await billingApi.getPlanPrices("based");

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/plans/based/prices?currency=USD`,
        expect.anything()
      );
      expect(result.price).toEqual(mockPrice);
    });
  });

  describe("getAllPlanPrices", () => {
    it("should fetch all currency prices for a plan", async () => {
      const mockPrices: PlanPrice[] = [
        { id: 1, plan_id: 1, currency: "USD", price_monthly: 9.9, price_yearly: 99 },
        { id: 2, plan_id: 1, currency: "CNY", price_monthly: 69, price_yearly: 690 },
      ];

      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(JSON.stringify({ prices: mockPrices })),
      });

      const result = await billingApi.getAllPlanPrices("based");

      expect(mockFetch).toHaveBeenCalledWith(
        `${EXPECTED_API_URL}/api/v1/orgs/test-org/billing/plans/based/all-prices`,
        expect.anything()
      );
      expect(result.prices).toEqual(mockPrices);
    });
  });
});

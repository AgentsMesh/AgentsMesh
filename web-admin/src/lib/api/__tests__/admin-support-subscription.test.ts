import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock the apiClient
const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPut = vi.fn();
const mockPatch = vi.fn();
const mockPostFormData = vi.fn();

vi.mock("@/lib/api/base", () => ({
  apiClient: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
    patch: (...args: unknown[]) => mockPatch(...args),
    postFormData: (...args: unknown[]) => mockPostFormData(...args),
  },
}));

import {
  listSupportTickets,
  getSupportTicketStats,
  getSupportTicketDetail,
  getSupportTicketMessages,
  replySupportTicket,
  updateSupportTicketStatus,
  assignSupportTicket,
  getSupportTicketAttachmentUrl,
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
} from "../admin";

describe("Admin API - Support Tickets & Subscriptions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Support Tickets", () => {
    it("listSupportTickets calls GET /support-tickets with params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listSupportTickets({ status: "open", page: 1 });
      expect(mockGet).toHaveBeenCalledWith(
        "/support-tickets",
        expect.objectContaining({ status: "open", page: 1 })
      );
    });

    it("getSupportTicketStats calls GET /support-tickets/stats", async () => {
      mockGet.mockResolvedValue({ total: 10, open: 3 });
      await getSupportTicketStats();
      expect(mockGet).toHaveBeenCalledWith("/support-tickets/stats");
    });

    it("getSupportTicketDetail calls GET /support-tickets/:id", async () => {
      mockGet.mockResolvedValue({ ticket: {}, messages: [] });
      await getSupportTicketDetail(7);
      expect(mockGet).toHaveBeenCalledWith("/support-tickets/7");
    });

    it("replySupportTicket sends FormData via postFormData", async () => {
      mockPostFormData.mockResolvedValue({ id: 1 });
      await replySupportTicket(7, "Hello");

      expect(mockPostFormData).toHaveBeenCalledWith(
        "/support-tickets/7/reply",
        expect.any(FormData)
      );
      const formData = mockPostFormData.mock.calls[0][1] as FormData;
      expect(formData.get("content")).toBe("Hello");
    });

    it("replySupportTicket includes files in FormData", async () => {
      mockPostFormData.mockResolvedValue({ id: 1 });
      const file = new File(["data"], "test.txt", { type: "text/plain" });

      await replySupportTicket(7, "With attachment", [file]);

      const formData = mockPostFormData.mock.calls[0][1] as FormData;
      const files = formData.getAll("files[]");
      expect(files).toHaveLength(1);
      expect((files[0] as File).name).toBe("test.txt");
    });

    it("updateSupportTicketStatus calls PATCH", async () => {
      mockPatch.mockResolvedValue({ message: "ok" });
      await updateSupportTicketStatus(7, "resolved");
      expect(mockPatch).toHaveBeenCalledWith(
        "/support-tickets/7/status",
        { status: "resolved" }
      );
    });

    it("assignSupportTicket calls POST", async () => {
      mockPost.mockResolvedValue({ message: "ok" });
      await assignSupportTicket(7, 42);
      expect(mockPost).toHaveBeenCalledWith(
        "/support-tickets/7/assign",
        { admin_id: 42 }
      );
    });

    it("getSupportTicketMessages calls GET /support-tickets/:id/messages", async () => {
      mockGet.mockResolvedValue({ messages: [] });
      await getSupportTicketMessages(7);
      expect(mockGet).toHaveBeenCalledWith("/support-tickets/7/messages");
    });

    it("getSupportTicketAttachmentUrl calls GET /support-tickets/attachments/:id/url", async () => {
      mockGet.mockResolvedValue({ url: "https://s3.example.com/file.png" });
      const result = await getSupportTicketAttachmentUrl(99);
      expect(mockGet).toHaveBeenCalledWith("/support-tickets/attachments/99/url");
      expect(result.url).toBe("https://s3.example.com/file.png");
    });
  });

  describe("Subscriptions", () => {
    it("getOrganizationSubscription calls GET /organizations/:id/subscription", async () => {
      mockGet.mockResolvedValue({ id: 1, status: "active" });
      await getOrganizationSubscription(10);
      expect(mockGet).toHaveBeenCalledWith("/organizations/10/subscription");
    });

    it("getSubscriptionPlans calls GET /organizations/:id/subscription/plans", async () => {
      mockGet.mockResolvedValue({ data: [] });
      await getSubscriptionPlans(10);
      expect(mockGet).toHaveBeenCalledWith("/organizations/10/subscription/plans");
    });

    it("createSubscription calls POST with plan_name and months", async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await createSubscription(10, "pro", 6);
      expect(mockPost).toHaveBeenCalledWith(
        "/organizations/10/subscription/create",
        { plan_name: "pro", months: 6 }
      );
    });

    it("createSubscription defaults months to 1", async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await createSubscription(10, "pro");
      expect(mockPost).toHaveBeenCalledWith(
        "/organizations/10/subscription/create",
        { plan_name: "pro", months: 1 }
      );
    });

    it("updateSubscriptionPlan calls PUT", async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await updateSubscriptionPlan(10, "enterprise");
      expect(mockPut).toHaveBeenCalledWith(
        "/organizations/10/subscription/plan",
        { plan_name: "enterprise" }
      );
    });

    it("updateSubscriptionSeats calls PUT", async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await updateSubscriptionSeats(10, 25);
      expect(mockPut).toHaveBeenCalledWith(
        "/organizations/10/subscription/seats",
        { seat_count: 25 }
      );
    });

    it("updateSubscriptionCycle calls PUT", async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await updateSubscriptionCycle(10, "yearly");
      expect(mockPut).toHaveBeenCalledWith(
        "/organizations/10/subscription/cycle",
        { billing_cycle: "yearly" }
      );
    });

    it("freezeSubscription calls POST", async () => {
      mockPost.mockResolvedValue({ id: 1, status: "frozen" });
      await freezeSubscription(10);
      expect(mockPost).toHaveBeenCalledWith("/organizations/10/subscription/freeze");
    });

    it("unfreezeSubscription calls POST", async () => {
      mockPost.mockResolvedValue({ id: 1, status: "active" });
      await unfreezeSubscription(10);
      expect(mockPost).toHaveBeenCalledWith("/organizations/10/subscription/unfreeze");
    });

    it("cancelSubscription calls POST", async () => {
      mockPost.mockResolvedValue({ id: 1, status: "canceled" });
      await cancelSubscription(10);
      expect(mockPost).toHaveBeenCalledWith("/organizations/10/subscription/cancel");
    });

    it("renewSubscription calls POST with months", async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await renewSubscription(10, 12);
      expect(mockPost).toHaveBeenCalledWith(
        "/organizations/10/subscription/renew",
        { months: 12 }
      );
    });

    it("setSubscriptionAutoRenew calls PUT", async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await setSubscriptionAutoRenew(10, false);
      expect(mockPut).toHaveBeenCalledWith(
        "/organizations/10/subscription/auto-renew",
        { auto_renew: false }
      );
    });

    it("setSubscriptionQuota calls PUT", async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await setSubscriptionQuota(10, "max_runners", 50);
      expect(mockPut).toHaveBeenCalledWith(
        "/organizations/10/subscription/quotas",
        { resource: "max_runners", limit: 50 }
      );
    });
  });
});

import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock the apiClient
const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPut = vi.fn();
const mockDelete = vi.fn();

vi.mock("@/lib/api/base", () => ({
  apiClient: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
    delete: (...args: unknown[]) => mockDelete(...args),
  },
}));

import {
  listPromoCodes,
  getPromoCode,
  createPromoCode,
  updatePromoCode,
  activatePromoCode,
  deactivatePromoCode,
  deletePromoCode,
  listPromoCodeRedemptions,
  listRelays,
  getRelayStats,
  getRelay,
  forceUnregisterRelay,
  listSessions,
  migrateSession,
  bulkMigrateSessions,
  listSkillRegistries,
  createSkillRegistry,
  syncSkillRegistry,
  deleteSkillRegistry,
} from "../admin";

describe("Admin API - Promo Codes, Relays & Skill Registries", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Promo Codes", () => {
    it("listPromoCodes converts boolean is_active to string", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listPromoCodes({ is_active: true, type: "media" });
      expect(mockGet).toHaveBeenCalledWith(
        "/promo-codes",
        expect.objectContaining({ is_active: "true", type: "media" })
      );
    });

    it("listPromoCodes omits undefined params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listPromoCodes({ page: 1 });
      const params = mockGet.mock.calls[0][1];
      expect(params.search).toBeUndefined();
      expect(params.type).toBeUndefined();
    });

    it("getPromoCode calls GET /promo-codes/:id", async () => {
      mockGet.mockResolvedValue({ id: 10 });
      await getPromoCode(10);
      expect(mockGet).toHaveBeenCalledWith("/promo-codes/10");
    });

    it("createPromoCode calls POST /promo-codes", async () => {
      const data = {
        code: "TEST50",
        name: "Test",
        type: "campaign" as const,
        plan_name: "pro",
        duration_months: 3,
      };
      mockPost.mockResolvedValue({ id: 1, ...data });
      await createPromoCode(data);
      expect(mockPost).toHaveBeenCalledWith("/promo-codes", data);
    });

    it("activatePromoCode calls POST /promo-codes/:id/activate", async () => {
      mockPost.mockResolvedValue({ message: "ok" });
      await activatePromoCode(5);
      expect(mockPost).toHaveBeenCalledWith("/promo-codes/5/activate");
    });

    it("deactivatePromoCode calls POST /promo-codes/:id/deactivate", async () => {
      mockPost.mockResolvedValue({ message: "ok" });
      await deactivatePromoCode(5);
      expect(mockPost).toHaveBeenCalledWith("/promo-codes/5/deactivate");
    });

    it("deletePromoCode calls DELETE /promo-codes/:id", async () => {
      mockDelete.mockResolvedValue({ message: "ok" });
      await deletePromoCode(5);
      expect(mockDelete).toHaveBeenCalledWith("/promo-codes/5");
    });

    it("updatePromoCode calls PUT /promo-codes/:id", async () => {
      mockPut.mockResolvedValue({ id: 5, name: "Updated" });
      await updatePromoCode(5, { name: "Updated" });
      expect(mockPut).toHaveBeenCalledWith("/promo-codes/5", { name: "Updated" });
    });

    it("listPromoCodeRedemptions calls GET /promo-codes/:id/redemptions", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listPromoCodeRedemptions(5, { page: 1, page_size: 10 });
      expect(mockGet).toHaveBeenCalledWith(
        "/promo-codes/5/redemptions",
        expect.objectContaining({ page: 1, page_size: 10 })
      );
    });

    it("listPromoCodes converts is_active false to string", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listPromoCodes({ is_active: false });
      const params = mockGet.mock.calls[0][1];
      expect(params.is_active).toBe("false");
    });

    it("listPromoCodes with all params set", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listPromoCodes({
        search: "TEST",
        type: "campaign",
        plan_name: "pro",
        is_active: true,
        page: 2,
        page_size: 25,
      });
      expect(mockGet).toHaveBeenCalledWith(
        "/promo-codes",
        expect.objectContaining({
          search: "TEST",
          type: "campaign",
          plan_name: "pro",
          is_active: "true",
          page: 2,
          page_size: 25,
        })
      );
    });
  });

  describe("Relays", () => {
    it("listRelays calls GET /relays", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listRelays();
      expect(mockGet).toHaveBeenCalledWith("/relays");
    });

    it("getRelayStats calls GET /relays/stats", async () => {
      mockGet.mockResolvedValue({ total_relays: 3 });
      await getRelayStats();
      expect(mockGet).toHaveBeenCalledWith("/relays/stats");
    });

    it("forceUnregisterRelay encodes relay ID", async () => {
      mockDelete.mockResolvedValue({ status: "ok" });
      await forceUnregisterRelay("relay/special", true);
      expect(mockDelete).toHaveBeenCalledWith(
        "/relays/relay%2Fspecial",
        { migrate_sessions: true }
      );
    });

    it("migrateSession calls POST /relays/sessions/migrate", async () => {
      mockPost.mockResolvedValue({ status: "ok" });
      await migrateSession("pod-1", "relay-2");
      expect(mockPost).toHaveBeenCalledWith("/relays/sessions/migrate", {
        pod_key: "pod-1",
        target_relay: "relay-2",
      });
    });

    it("bulkMigrateSessions calls POST /relays/sessions/bulk-migrate", async () => {
      mockPost.mockResolvedValue({ status: "ok", total: 5, migrated: 5 });
      await bulkMigrateSessions("relay-1", "relay-2");
      expect(mockPost).toHaveBeenCalledWith(
        "/relays/sessions/bulk-migrate",
        { source_relay: "relay-1", target_relay: "relay-2" }
      );
    });

    it("getRelay encodes relay ID and calls GET", async () => {
      mockGet.mockResolvedValue({ relay: {}, sessions: [] });
      await getRelay("relay/with-slash");
      expect(mockGet).toHaveBeenCalledWith("/relays/relay%2Fwith-slash");
    });

    it("listSessions calls GET /relays/sessions with optional relay_id", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listSessions("relay-1");
      expect(mockGet).toHaveBeenCalledWith(
        "/relays/sessions",
        { relay_id: "relay-1" }
      );
    });

    it("listSessions calls GET /relays/sessions without params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listSessions();
      expect(mockGet).toHaveBeenCalledWith("/relays/sessions", undefined);
    });

    it("forceUnregisterRelay defaults migrateSessions to false", async () => {
      mockDelete.mockResolvedValue({ status: "ok" });
      await forceUnregisterRelay("relay-1");
      expect(mockDelete).toHaveBeenCalledWith(
        "/relays/relay-1",
        { migrate_sessions: false }
      );
    });
  });

  describe("Skill Registries", () => {
    it("listSkillRegistries calls GET /skill-registries", async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await listSkillRegistries();
      expect(mockGet).toHaveBeenCalledWith("/skill-registries");
    });

    it("createSkillRegistry calls POST /skill-registries", async () => {
      const data = { repository_url: "https://github.com/org/repo" };
      mockPost.mockResolvedValue({ id: 1 });
      await createSkillRegistry(data);
      expect(mockPost).toHaveBeenCalledWith("/skill-registries", data);
    });

    it("syncSkillRegistry calls POST /skill-registries/:id/sync", async () => {
      mockPost.mockResolvedValue({ message: "ok" });
      await syncSkillRegistry(3);
      expect(mockPost).toHaveBeenCalledWith("/skill-registries/3/sync");
    });

    it("deleteSkillRegistry calls DELETE /skill-registries/:id", async () => {
      mockDelete.mockResolvedValue(undefined);
      await deleteSkillRegistry(3);
      expect(mockDelete).toHaveBeenCalledWith("/skill-registries/3");
    });
  });
});

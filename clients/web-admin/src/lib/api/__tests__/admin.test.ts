import { describe, it, expect, vi, beforeEach } from "vitest";

// Users / Organizations migrated to Connect-RPC — see adminUsers.test.ts
// and adminOrganizations.test.ts. The rest (Dashboard / Runners /
// Audit Logs / Auth) still routes through REST apiClient.
const mockGet = vi.fn();
const mockPost = vi.fn();
const mockDelete = vi.fn();

vi.mock("../base", () => ({
  apiClient: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => vi.fn()(...args),
    delete: (...args: unknown[]) => mockDelete(...args),
  },
}));

import {
  getDashboardStats,
  listRunners,
  getRunner,
  disableRunner,
  enableRunner,
  deleteRunner,
  listAuditLogs,
  login,
  getCurrentAdmin,
} from "../admin";

describe("Admin API - REST surface", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Dashboard", () => {
    it("getDashboardStats calls GET /dashboard/stats", async () => {
      mockGet.mockResolvedValue({ total_users: 100 });
      const result = await getDashboardStats();
      expect(mockGet).toHaveBeenCalledWith("/dashboard/stats");
      expect(result.total_users).toBe(100);
    });
  });

  describe("Runners", () => {
    it("listRunners calls GET /runners with params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listRunners({ org_id: 5 });
      expect(mockGet).toHaveBeenCalledWith(
        "/runners",
        expect.objectContaining({ org_id: 5 }),
      );
    });

    it("disableRunner calls POST /runners/:id/disable", async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await disableRunner(1);
      expect(mockPost).toHaveBeenCalledWith("/runners/1/disable");
    });

    it("enableRunner calls POST /runners/:id/enable", async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await enableRunner(1);
      expect(mockPost).toHaveBeenCalledWith("/runners/1/enable");
    });

    it("deleteRunner calls DELETE /runners/:id", async () => {
      mockDelete.mockResolvedValue({ message: "ok" });
      await deleteRunner(1);
      expect(mockDelete).toHaveBeenCalledWith("/runners/1");
    });

    it("getRunner calls GET /runners/:id", async () => {
      mockGet.mockResolvedValue({ id: 3, node_id: "node-3" });
      await getRunner(3);
      expect(mockGet).toHaveBeenCalledWith("/runners/3");
    });
  });

  describe("Audit Logs", () => {
    it("listAuditLogs calls GET /audit-logs with params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listAuditLogs({ target_type: "user", page: 1 });
      expect(mockGet).toHaveBeenCalledWith(
        "/audit-logs",
        expect.objectContaining({ target_type: "user", page: 1 }),
      );
    });
  });

  describe("Auth", () => {
    it("login calls POST /auth/login", async () => {
      mockPost.mockResolvedValue({ token: "t", user: {} });
      await login({ email: "admin@test.com", password: "pass" });
      expect(mockPost).toHaveBeenCalledWith("/auth/login", {
        email: "admin@test.com",
        password: "pass",
      });
    });

    it("getCurrentAdmin calls GET /me", async () => {
      mockGet.mockResolvedValue({ id: 1 });
      await getCurrentAdmin();
      expect(mockGet).toHaveBeenCalledWith("/me");
    });
  });
});

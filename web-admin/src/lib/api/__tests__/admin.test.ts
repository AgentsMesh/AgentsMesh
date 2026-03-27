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
  getDashboardStats,
  listUsers,
  getUser,
  updateUser,
  disableUser,
  enableUser,
  grantAdmin,
  revokeAdmin,
  listOrganizations,
  getOrganization,
  getOrganizationMembers,
  deleteOrganization,
  listRunners,
  getRunner,
  disableRunner,
  enableRunner,
  deleteRunner,
  listAuditLogs,
  login,
  getCurrentAdmin,
} from "../admin";

describe("Admin API - Core", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // --- Dashboard ---
  describe("Dashboard", () => {
    it("getDashboardStats calls GET /dashboard/stats", async () => {
      mockGet.mockResolvedValue({ total_users: 100 });
      const result = await getDashboardStats();
      expect(mockGet).toHaveBeenCalledWith("/dashboard/stats");
      expect(result.total_users).toBe(100);
    });
  });

  // --- Users ---
  describe("Users", () => {
    it("listUsers calls GET /users with params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listUsers({ search: "admin", page: 2, page_size: 10 });
      expect(mockGet).toHaveBeenCalledWith(
        "/users",
        expect.objectContaining({ search: "admin", page: 2, page_size: 10 })
      );
    });

    it("getUser calls GET /users/:id", async () => {
      mockGet.mockResolvedValue({ id: 5, email: "a@b.com" });
      const result = await getUser(5);
      expect(mockGet).toHaveBeenCalledWith("/users/5");
      expect(result.id).toBe(5);
    });

    it("disableUser calls POST /users/:id/disable", async () => {
      mockPost.mockResolvedValue({ id: 1, is_active: false });
      await disableUser(1);
      expect(mockPost).toHaveBeenCalledWith("/users/1/disable");
    });

    it("enableUser calls POST /users/:id/enable", async () => {
      mockPost.mockResolvedValue({ id: 1, is_active: true });
      await enableUser(1);
      expect(mockPost).toHaveBeenCalledWith("/users/1/enable");
    });

    it("grantAdmin calls POST /users/:id/grant-admin", async () => {
      mockPost.mockResolvedValue({ id: 1, is_system_admin: true });
      await grantAdmin(1);
      expect(mockPost).toHaveBeenCalledWith("/users/1/grant-admin");
    });

    it("revokeAdmin calls POST /users/:id/revoke-admin", async () => {
      mockPost.mockResolvedValue({ id: 1, is_system_admin: false });
      await revokeAdmin(1);
      expect(mockPost).toHaveBeenCalledWith("/users/1/revoke-admin");
    });

    it("updateUser calls PUT /users/:id", async () => {
      mockPut.mockResolvedValue({ id: 1, name: "Updated" });
      await updateUser(1, { name: "Updated" });
      expect(mockPut).toHaveBeenCalledWith("/users/1", { name: "Updated" });
    });
  });

  // --- Organizations ---
  describe("Organizations", () => {
    it("listOrganizations calls GET /organizations", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listOrganizations({ search: "test" });
      expect(mockGet).toHaveBeenCalledWith(
        "/organizations",
        expect.objectContaining({ search: "test" })
      );
    });

    it("getOrganization calls GET /organizations/:id", async () => {
      mockGet.mockResolvedValue({ id: 1, name: "Org" });
      await getOrganization(1);
      expect(mockGet).toHaveBeenCalledWith("/organizations/1");
    });

    it("deleteOrganization calls DELETE /organizations/:id", async () => {
      mockDelete.mockResolvedValue({ message: "deleted" });
      await deleteOrganization(1);
      expect(mockDelete).toHaveBeenCalledWith("/organizations/1");
    });

    it("getOrganizationMembers calls GET /organizations/:id/members", async () => {
      mockGet.mockResolvedValue({ organization: {}, members: [] });
      await getOrganizationMembers(1);
      expect(mockGet).toHaveBeenCalledWith("/organizations/1/members");
    });
  });

  // --- Runners ---
  describe("Runners", () => {
    it("listRunners calls GET /runners with params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listRunners({ org_id: 5 });
      expect(mockGet).toHaveBeenCalledWith(
        "/runners",
        expect.objectContaining({ org_id: 5 })
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

  // --- Audit Logs ---
  describe("Audit Logs", () => {
    it("listAuditLogs calls GET /audit-logs with params", async () => {
      mockGet.mockResolvedValue({ data: [], total: 0 });
      await listAuditLogs({ target_type: "user", page: 1 });
      expect(mockGet).toHaveBeenCalledWith(
        "/audit-logs",
        expect.objectContaining({ target_type: "user", page: 1 })
      );
    });
  });

  // --- Auth ---
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

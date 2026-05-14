import { describe, it, expect, vi, beforeEach } from "vitest";

const mockCallConnect = vi.fn();
vi.mock("@/lib/connect/transport", () => ({
  callConnect: (...args: unknown[]) => mockCallConnect(...args),
  ConnectError: class ConnectError extends Error {},
}));

// Dashboard still uses REST — mock apiClient too.
const mockGet = vi.fn();
vi.mock("../base", () => ({
  apiClient: { get: (...args: unknown[]) => mockGet(...args) },
}));

import {
  listUsers,
  getUser,
  updateUser,
  disableUser,
  enableUser,
  grantAdmin,
  revokeAdmin,
  verifyUserEmail,
  unverifyUserEmail,
} from "../adminUsers";

function fakeAdminUserResp(id: number, overrides: Record<string, unknown> = {}) {
  return {
    id: BigInt(id),
    email: "",
    username: "",
    isActive: true,
    isSystemAdmin: false,
    isEmailVerified: false,
    createdAt: "",
    updatedAt: "",
    ...overrides,
  };
}

describe("Admin API - Users (Connect-RPC)", () => {
  beforeEach(() => vi.clearAllMocks());

  it("listUsers maps proto response to PaginatedResponse<User>", async () => {
    mockCallConnect.mockResolvedValue({
      items: [fakeAdminUserResp(7, { email: "a@b.com", username: "alice" })],
      total: 1n,
      page: 1,
      pageSize: 10,
      totalPages: 1,
    });
    const result = await listUsers({ search: "alice", page: 1, page_size: 10 });
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "ListUsers",
      expect.anything(),
      expect.anything(),
      expect.objectContaining({ search: "alice", page: 1, pageSize: 10 }),
    );
    expect(result.data[0].id).toBe(7);
    expect(result.data[0].email).toBe("a@b.com");
    expect(result.total).toBe(1);
  });

  it("getUser passes BigInt userId", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminUserResp(5));
    const u = await getUser(5);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "GetUser",
      expect.anything(),
      expect.anything(),
      { userId: 5n },
    );
    expect(u.id).toBe(5);
  });

  it("updateUser forwards optional fields", async () => {
    mockCallConnect.mockResolvedValue(fakeAdminUserResp(1, { name: "New" }));
    await updateUser(1, { name: "New", username: "u" });
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "UpdateUser",
      expect.anything(),
      expect.anything(),
      expect.objectContaining({ userId: 1n, name: "New", username: "u" }),
    );
  });

  it.each([
    ["disableUser", disableUser, "DisableUser"],
    ["enableUser", enableUser, "EnableUser"],
    ["grantAdmin", grantAdmin, "GrantAdmin"],
    ["revokeAdmin", revokeAdmin, "RevokeAdmin"],
    ["verifyUserEmail", verifyUserEmail, "VerifyUserEmail"],
    ["unverifyUserEmail", unverifyUserEmail, "UnverifyUserEmail"],
  ])("%s maps id to userId", async (_name, fn, rpcName) => {
    mockCallConnect.mockResolvedValue(fakeAdminUserResp(42));
    await fn(42);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      rpcName,
      expect.anything(),
      expect.anything(),
      { userId: 42n },
    );
  });
});

import { describe, it, expect, vi, beforeEach } from "vitest";

const mockCallConnect = vi.fn();
vi.mock("@/lib/connect/transport", () => ({
  callConnect: (...args: unknown[]) => mockCallConnect(...args),
  ConnectError: class ConnectError extends Error {},
}));

import {
  listOrganizations,
  getOrganization,
  getOrganizationMembers,
  deleteOrganization,
} from "../adminOrganizations";

function fakeOrgResp(id: number, overrides: Record<string, unknown> = {}) {
  return {
    id: BigInt(id),
    name: "Org",
    slug: "org",
    subscriptionPlan: "based",
    subscriptionStatus: "active",
    createdAt: "",
    updatedAt: "",
    ...overrides,
  };
}

describe("Admin API - Organizations (Connect-RPC)", () => {
  beforeEach(() => vi.clearAllMocks());

  it("listOrganizations maps proto response", async () => {
    mockCallConnect.mockResolvedValue({
      items: [fakeOrgResp(3, { name: "Acme" })],
      total: 1n,
      page: 1,
      pageSize: 10,
      totalPages: 1,
    });
    const result = await listOrganizations({ search: "ac" });
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "ListOrganizations",
      expect.anything(),
      expect.anything(),
      expect.objectContaining({ search: "ac" }),
    );
    expect(result.data[0].name).toBe("Acme");
    expect(result.data[0].id).toBe(3);
    expect(result.total).toBe(1);
  });

  it("getOrganization passes BigInt orgId", async () => {
    mockCallConnect.mockResolvedValue(fakeOrgResp(7, { name: "Beta" }));
    const o = await getOrganization(7);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "GetOrganization",
      expect.anything(),
      expect.anything(),
      { orgId: 7n },
    );
    expect(o.name).toBe("Beta");
    expect(o.id).toBe(7);
  });

  it("getOrganizationMembers maps nested user", async () => {
    mockCallConnect.mockResolvedValue({
      organization: fakeOrgResp(1),
      members: [
        {
          id: 10n,
          userId: 5n,
          orgId: 1n,
          role: "owner",
          joinedAt: "2026-01-01T00:00:00Z",
          user: {
            id: 5n,
            email: "owner@ex.com",
            username: "owner",
            name: "Owner",
            avatarUrl: undefined,
          },
        },
      ],
    });
    const result = await getOrganizationMembers(1);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "GetOrganizationMembers",
      expect.anything(),
      expect.anything(),
      { orgId: 1n },
    );
    expect(result.members[0].user_id).toBe(5);
    expect(result.members[0].role).toBe("owner");
    expect(result.members[0].user?.username).toBe("owner");
  });

  it("deleteOrganization returns message", async () => {
    mockCallConnect.mockResolvedValue({ message: "Organization deleted successfully" });
    const result = await deleteOrganization(1);
    expect(mockCallConnect).toHaveBeenCalledWith(
      "proto.admin.v1.AdminService",
      "DeleteOrganization",
      expect.anything(),
      expect.anything(),
      { orgId: 1n },
    );
    expect(result.message).toContain("deleted");
  });
});

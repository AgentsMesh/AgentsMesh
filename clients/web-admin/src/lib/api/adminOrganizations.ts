// Connect-RPC adapter for proto.admin.v1.AdminService — organization
// management. Migrated from REST `/api/v1/admin/organizations/*`.
//
// Keeps the existing TS return shapes (snake_case + number) so admin pages
// don't need to change.
import {
  AdminOrganizationSchema,
  DeleteOrganizationRequestSchema,
  DeleteOrganizationResponseSchema,
  GetOrganizationMembersRequestSchema,
  GetOrganizationMembersResponseSchema,
  GetOrganizationRequestSchema,
  ListOrganizationsRequestSchema,
  ListOrganizationsResponseSchema,
  type AdminOrganization as ProtoAdminOrganization,
  type AdminOrganizationMember as ProtoAdminMember,
} from "@proto/admin/v1/admin_pb";

import { PaginatedResponse } from "./base";
import { callConnect } from "@/lib/connect/transport";
import type { Organization, OrganizationMember, OrganizationListParams } from "./adminTypes";

const SERVICE = "proto.admin.v1.AdminService";

function organizationFromProto(o: ProtoAdminOrganization): Organization {
  return {
    id: Number(o.id),
    name: o.name,
    slug: o.slug,
    description: null,
    logo_url: o.logoUrl ?? null,
    subscription_plan: o.subscriptionPlan,
    subscription_status: o.subscriptionStatus,
    created_at: o.createdAt,
    updated_at: o.updatedAt,
  };
}

function memberFromProto(m: ProtoAdminMember): OrganizationMember {
  const u = m.user
    ? {
        id: Number(m.user.id),
        email: m.user.email,
        username: m.user.username,
        name: m.user.name ?? null,
        avatar_url: m.user.avatarUrl ?? null,
      }
    : undefined;
  return {
    id: Number(m.id),
    user_id: Number(m.userId),
    org_id: Number(m.orgId),
    role: m.role,
    created_at: m.joinedAt,
    joined_at: m.joinedAt,
    user: u,
  };
}

export async function listOrganizations(
  params?: OrganizationListParams,
): Promise<PaginatedResponse<Organization>> {
  const resp = await callConnect(
    SERVICE,
    "ListOrganizations",
    ListOrganizationsRequestSchema,
    ListOrganizationsResponseSchema,
    {
      search: params?.search ?? "",
      page: params?.page ?? 0,
      pageSize: params?.page_size ?? 0,
    },
  );
  return {
    data: resp.items.map(organizationFromProto),
    total: Number(resp.total),
    page: resp.page,
    page_size: resp.pageSize,
    total_pages: resp.totalPages,
  };
}

export async function getOrganization(id: number): Promise<Organization> {
  const resp = await callConnect(
    SERVICE,
    "GetOrganization",
    GetOrganizationRequestSchema,
    AdminOrganizationSchema,
    { orgId: BigInt(id) },
  );
  return organizationFromProto(resp);
}

export async function getOrganizationMembers(
  id: number,
): Promise<{ organization: Organization; members: OrganizationMember[] }> {
  const resp = await callConnect(
    SERVICE,
    "GetOrganizationMembers",
    GetOrganizationMembersRequestSchema,
    GetOrganizationMembersResponseSchema,
    { orgId: BigInt(id) },
  );
  return {
    organization: organizationFromProto(resp.organization!),
    members: resp.members.map(memberFromProto),
  };
}

export async function deleteOrganization(id: number): Promise<{ message: string }> {
  const resp = await callConnect(
    SERVICE,
    "DeleteOrganization",
    DeleteOrganizationRequestSchema,
    DeleteOrganizationResponseSchema,
    { orgId: BigInt(id) },
  );
  return { message: resp.message };
}

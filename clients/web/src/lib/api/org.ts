// Connect-RPC adapter for proto.org.v1.OrgService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out per conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the snake_case OrganizationData / OrganizationMember shapes (the
// existing web types) so call sites don't have to convert. The proto types
// are camelCase + BigInt; the adapter does the mapping.

import {
  CreateOrgRequestSchema,
  CreatePersonalOrgRequestSchema,
  DeleteOrgRequestSchema,
  DeleteOrgResponseSchema,
  GetOrgRequestSchema,
  InviteMemberRequestSchema,
  InviteMemberResponseSchema,
  ListMembersRequestSchema,
  ListMembersResponseSchema,
  ListMyOrgsRequestSchema,
  ListMyOrgsResponseSchema,
  OrganizationSchema,
  RemoveMemberRequestSchema,
  RemoveMemberResponseSchema,
  UpdateMemberRoleRequestSchema,
  UpdateMemberRoleResponseSchema,
  UpdateOrgRequestSchema,
  type Organization as ProtoOrganization,
  type OrganizationMember as ProtoOrganizationMember,
} from "@proto/org/v1/org_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getOrgApiService } from "@/lib/wasm-core";
import type { OrganizationData, OrganizationMember } from "@/lib/api/organizationTypes";

function fromProtoOrg(o: ProtoOrganization): OrganizationData {
  return {
    id: Number(o.id),
    name: o.name,
    slug: o.slug,
    role: o.role,
    logo_url: o.logoUrl,
    subscription_plan: o.subscriptionPlan,
    subscription_status: o.subscriptionStatus,
    created_at: o.createdAt,
    updated_at: o.updatedAt,
  };
}

function fromProtoMember(m: ProtoOrganizationMember): OrganizationMember {
  return {
    id: Number(m.id),
    user_id: Number(m.userId),
    role: m.role as OrganizationMember["role"],
    joined_at: m.joinedAt,
    user: m.user
      ? {
          id: Number(m.user.id),
          email: m.user.email,
          username: m.user.username,
          name: m.user.name,
          avatar_url: m.user.avatarUrl,
        }
      : undefined,
  };
}

export async function listMyOrgs(): Promise<{
  items: OrganizationData[];
  total: number;
  limit: number;
  offset: number;
}> {
  const req = create(ListMyOrgsRequestSchema, {});
  const bytes = toBinary(ListMyOrgsRequestSchema, req);
  const respBytes = await getOrgApiService().listMyOrgsConnect(bytes);
  const resp = fromBinary(ListMyOrgsResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoOrg),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function createOrg(data: {
  name: string;
  slug: string;
  logoUrl?: string;
}): Promise<OrganizationData> {
  const req = create(CreateOrgRequestSchema, {
    name: data.name,
    slug: data.slug,
    logoUrl: data.logoUrl,
  });
  const bytes = toBinary(CreateOrgRequestSchema, req);
  const respBytes = await getOrgApiService().createOrgConnect(bytes);
  return fromProtoOrg(fromBinary(OrganizationSchema, new Uint8Array(respBytes)));
}

export async function createPersonalOrg(): Promise<OrganizationData> {
  const req = create(CreatePersonalOrgRequestSchema, {});
  const bytes = toBinary(CreatePersonalOrgRequestSchema, req);
  const respBytes = await getOrgApiService().createPersonalOrgConnect(bytes);
  return fromProtoOrg(fromBinary(OrganizationSchema, new Uint8Array(respBytes)));
}

export async function getOrg(orgSlug: string): Promise<OrganizationData> {
  const req = create(GetOrgRequestSchema, { orgSlug });
  const bytes = toBinary(GetOrgRequestSchema, req);
  const respBytes = await getOrgApiService().getOrgConnect(bytes);
  return fromProtoOrg(fromBinary(OrganizationSchema, new Uint8Array(respBytes)));
}

export async function updateOrg(
  orgSlug: string,
  data: { name?: string; logoUrl?: string },
): Promise<OrganizationData> {
  const req = create(UpdateOrgRequestSchema, {
    orgSlug,
    name: data.name,
    logoUrl: data.logoUrl,
  });
  const bytes = toBinary(UpdateOrgRequestSchema, req);
  const respBytes = await getOrgApiService().updateOrgConnect(bytes);
  return fromProtoOrg(fromBinary(OrganizationSchema, new Uint8Array(respBytes)));
}

export async function deleteOrg(orgSlug: string): Promise<void> {
  const req = create(DeleteOrgRequestSchema, { orgSlug });
  const bytes = toBinary(DeleteOrgRequestSchema, req);
  const respBytes = await getOrgApiService().deleteOrgConnect(bytes);
  fromBinary(DeleteOrgResponseSchema, new Uint8Array(respBytes));
}

export async function listMembers(
  orgSlug: string,
  opts: { offset?: number; limit?: number } = {},
): Promise<{
  items: OrganizationMember[];
  total: number;
  limit: number;
  offset: number;
}> {
  const req = create(ListMembersRequestSchema, {
    orgSlug,
    offset: opts.offset,
    limit: opts.limit,
  });
  const bytes = toBinary(ListMembersRequestSchema, req);
  const respBytes = await getOrgApiService().listMembersConnect(bytes);
  const resp = fromBinary(ListMembersResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(fromProtoMember),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function inviteMember(
  orgSlug: string,
  data: { email?: string; userId?: number; role: string },
): Promise<void> {
  const req = create(InviteMemberRequestSchema, {
    orgSlug,
    email: data.email,
    userId: data.userId === undefined ? undefined : BigInt(data.userId),
    role: data.role,
  });
  const bytes = toBinary(InviteMemberRequestSchema, req);
  const respBytes = await getOrgApiService().inviteMemberConnect(bytes);
  fromBinary(InviteMemberResponseSchema, new Uint8Array(respBytes));
}

export async function removeMember(orgSlug: string, userId: number): Promise<void> {
  const req = create(RemoveMemberRequestSchema, { orgSlug, userId: BigInt(userId) });
  const bytes = toBinary(RemoveMemberRequestSchema, req);
  const respBytes = await getOrgApiService().removeMemberConnect(bytes);
  fromBinary(RemoveMemberResponseSchema, new Uint8Array(respBytes));
}

export async function updateMemberRole(
  orgSlug: string,
  userId: number,
  role: string,
): Promise<void> {
  const req = create(UpdateMemberRoleRequestSchema, {
    orgSlug,
    userId: BigInt(userId),
    role,
  });
  const bytes = toBinary(UpdateMemberRoleRequestSchema, req);
  const respBytes = await getOrgApiService().updateMemberRoleConnect(bytes);
  fromBinary(UpdateMemberRoleResponseSchema, new Uint8Array(respBytes));
}

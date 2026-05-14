// Connect-RPC adapter for proto.admin.v1.AdminService — user management.
//
// Migrated from REST `/api/v1/admin/users/*`. Keeps the existing TS
// return shapes (snake_case + number) so admin pages don't need to
// change. Proto types are camelCase + bigint; this module bridges
// the gap.
//
// Auth: handler-level via interceptors.ResolveSystemAdmin (see
// backend/internal/api/connect/admin/server.go). The bearer JWT is
// attached by callConnect from the auth store.
import {
  AdminService,
  AdminUserSchema,
  DisableUserRequestSchema,
  EnableUserRequestSchema,
  GetUserRequestSchema,
  GrantAdminRequestSchema,
  ListUsersRequestSchema,
  ListUsersResponseSchema,
  RevokeAdminRequestSchema,
  UnverifyUserEmailRequestSchema,
  UpdateUserRequestSchema,
  VerifyUserEmailRequestSchema,
  type AdminUser as ProtoAdminUser,
} from "@proto/admin/v1/admin_pb";

import { PaginatedResponse } from "./base";
import { callConnect } from "@/lib/connect/transport";
import type { User, UserListParams, DashboardStats } from "./adminTypes";

const SERVICE = "proto.admin.v1.AdminService";
void AdminService;

function userFromProto(u: ProtoAdminUser): User {
  return {
    id: Number(u.id),
    email: u.email,
    username: u.username,
    name: u.name ?? null,
    avatar_url: u.avatarUrl ?? null,
    is_active: u.isActive,
    is_system_admin: u.isSystemAdmin,
    is_email_verified: u.isEmailVerified,
    last_login_at: u.lastLoginAt ?? null,
    created_at: u.createdAt,
    updated_at: u.updatedAt,
  };
}

// Dashboard stays on REST until a follow-up PR migrates it; we re-export
// the existing REST call here so the import surface in adminUsers.ts
// remains stable for callers.
export { getDashboardStats } from "./adminDashboard";
export type { DashboardStats };

export async function listUsers(params?: UserListParams): Promise<PaginatedResponse<User>> {
  const resp = await callConnect(
    SERVICE,
    "ListUsers",
    ListUsersRequestSchema,
    ListUsersResponseSchema,
    {
      search: params?.search ?? "",
      isActive: params?.is_active,
      isAdmin: params?.is_admin,
      page: params?.page ?? 0,
      pageSize: params?.page_size ?? 0,
    },
  );
  return {
    data: resp.items.map(userFromProto),
    total: Number(resp.total),
    page: resp.page,
    page_size: resp.pageSize,
    total_pages: resp.totalPages,
  };
}

export async function getUser(id: number): Promise<User> {
  const resp = await callConnect(SERVICE, "GetUser", GetUserRequestSchema, AdminUserSchema, {
    userId: BigInt(id),
  });
  return userFromProto(resp);
}

export async function updateUser(
  id: number,
  data: { name?: string; username?: string; email?: string },
): Promise<User> {
  const resp = await callConnect(SERVICE, "UpdateUser", UpdateUserRequestSchema, AdminUserSchema, {
    userId: BigInt(id),
    name: data.name,
    username: data.username,
    email: data.email,
  });
  return userFromProto(resp);
}

export async function disableUser(id: number): Promise<User> {
  const resp = await callConnect(SERVICE, "DisableUser", DisableUserRequestSchema, AdminUserSchema, {
    userId: BigInt(id),
  });
  return userFromProto(resp);
}

export async function enableUser(id: number): Promise<User> {
  const resp = await callConnect(SERVICE, "EnableUser", EnableUserRequestSchema, AdminUserSchema, {
    userId: BigInt(id),
  });
  return userFromProto(resp);
}

export async function grantAdmin(id: number): Promise<User> {
  const resp = await callConnect(SERVICE, "GrantAdmin", GrantAdminRequestSchema, AdminUserSchema, {
    userId: BigInt(id),
  });
  return userFromProto(resp);
}

export async function revokeAdmin(id: number): Promise<User> {
  const resp = await callConnect(SERVICE, "RevokeAdmin", RevokeAdminRequestSchema, AdminUserSchema, {
    userId: BigInt(id),
  });
  return userFromProto(resp);
}

export async function verifyUserEmail(id: number): Promise<User> {
  const resp = await callConnect(
    SERVICE,
    "VerifyUserEmail",
    VerifyUserEmailRequestSchema,
    AdminUserSchema,
    { userId: BigInt(id) },
  );
  return userFromProto(resp);
}

export async function unverifyUserEmail(id: number): Promise<User> {
  const resp = await callConnect(
    SERVICE,
    "UnverifyUserEmail",
    UnverifyUserEmailRequestSchema,
    AdminUserSchema,
    { userId: BigInt(id) },
  );
  return userFromProto(resp);
}

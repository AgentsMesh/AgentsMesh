// Connect-RPC adapter for proto.auth.v1.AuthService + AuthSessionService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out per conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing snake_case web shapes (AuthSession, MessageResponse)
// so the 11 call sites don't have to flip off camelCase + BigInt — same
// dual-track pattern as ssoConnect.ts / podConnect.ts during the migration
// window. The legacy `authApi` JSON methods stay available until call sites
// finish flipping over.
//
// Login/RefreshToken/Logout are routed through this adapter, but
// AuthManager (the stateful auth-flow owner in the Rust auth crate)
// keeps its own REST path for now — its token-store mutations happen
// inside the manager, not at the adapter level. UI call sites that
// previously hit `authApi.register/verify/forgot/reset` migrate here
// 1:1; AuthManager.login()/refresh_token() stay routed through the
// AuthManager surface (`getAuthManager().login(...)`).

import {
  RegisterRequestSchema,
  RegisterResponseSchema,
  VerifyEmailRequestSchema,
  VerifyEmailResponseSchema,
  ResendVerificationRequestSchema,
  ResendVerificationResponseSchema,
  ForgotPasswordRequestSchema,
  ForgotPasswordResponseSchema,
  ResetPasswordRequestSchema,
  ResetPasswordResponseSchema,
  type RegisterResponse as ProtoRegisterResponse,
  type VerifyEmailResponse as ProtoVerifyEmailResponse,
  type User as ProtoUser,
} from "@proto/auth/v1/auth_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getAuthConnectService } from "@/lib/wasm-core";

// ============== Wire conversion (proto -> snake_case web shape) ==============

export interface AuthUserShape {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string;
  is_email_verified?: boolean;
}

export interface AuthSessionShape {
  token: string;
  refresh_token: string;
  expires_in: number;
  user: AuthUserShape;
  message?: string;
}

export interface MessageResponseShape {
  message: string;
}

function fromProtoUser(u: ProtoUser | undefined): AuthUserShape {
  if (!u) return { id: 0, email: "", username: "" };
  return {
    id: Number(u.id),
    email: u.email,
    username: u.username,
    name: u.name,
    avatar_url: u.avatarUrl,
    is_email_verified: u.isEmailVerified,
  };
}

function fromProtoSession(
  p: ProtoRegisterResponse | ProtoVerifyEmailResponse,
): AuthSessionShape {
  return {
    token: p.token,
    refresh_token: p.refreshToken,
    expires_in: Number(p.expiresIn),
    user: fromProtoUser(p.user),
    message: "message" in p ? p.message : undefined,
  };
}

// ============== AuthService — PUBLIC (no auth required) ==============

export interface RegisterInput {
  email: string;
  username: string;
  password: string;
  name?: string;
}

export async function register(input: RegisterInput): Promise<AuthSessionShape> {
  const req = create(RegisterRequestSchema, {
    email: input.email,
    username: input.username,
    password: input.password,
    name: input.name,
  });
  const bytes = toBinary(RegisterRequestSchema, req);
  const respBytes = await getAuthConnectService().registerConnect(bytes);
  return fromProtoSession(
    fromBinary(RegisterResponseSchema, new Uint8Array(respBytes)),
  );
}

export async function verifyEmail(token: string): Promise<AuthSessionShape> {
  const req = create(VerifyEmailRequestSchema, { token });
  const bytes = toBinary(VerifyEmailRequestSchema, req);
  const respBytes = await getAuthConnectService().verifyEmailConnect(bytes);
  return fromProtoSession(
    fromBinary(VerifyEmailResponseSchema, new Uint8Array(respBytes)),
  );
}

export async function resendVerification(
  email: string,
): Promise<MessageResponseShape> {
  const req = create(ResendVerificationRequestSchema, { email });
  const bytes = toBinary(ResendVerificationRequestSchema, req);
  const respBytes = await getAuthConnectService().resendVerificationConnect(bytes);
  return fromBinary(ResendVerificationResponseSchema, new Uint8Array(respBytes));
}

export async function forgotPassword(
  email: string,
): Promise<MessageResponseShape> {
  const req = create(ForgotPasswordRequestSchema, { email });
  const bytes = toBinary(ForgotPasswordRequestSchema, req);
  const respBytes = await getAuthConnectService().forgotPasswordConnect(bytes);
  return fromBinary(ForgotPasswordResponseSchema, new Uint8Array(respBytes));
}

export async function resetPassword(
  token: string,
  newPassword: string,
): Promise<MessageResponseShape> {
  const req = create(ResetPasswordRequestSchema, {
    token,
    newPassword,
  });
  const bytes = toBinary(ResetPasswordRequestSchema, req);
  const respBytes = await getAuthConnectService().resetPasswordConnect(bytes);
  return fromBinary(ResetPasswordResponseSchema, new Uint8Array(respBytes));
}

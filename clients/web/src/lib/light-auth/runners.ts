
import { lightFetch } from "./api-fetch";
import type { RunnerAuthStatus, RunnerData, RunnerListResponse } from "@/lib/api/runnerTypes";

export async function lightGetRunnerAuthStatus(authKey: string): Promise<RunnerAuthStatus> {
  return lightFetch<RunnerAuthStatus>("/api/v1/runners/grpc/auth-status", {
    query: { key: authKey },
  });
}

export interface LightAuthorizeRunnerInput {
  organizationSlug: string;
  authKey: string;
  nodeId?: string;
}

interface AuthorizeRunnerResponse {
  runner_id?: number;
  [key: string]: unknown;
}

export async function lightAuthorizeRunner(
  input: LightAuthorizeRunnerInput,
): Promise<AuthorizeRunnerResponse> {
  const path = `/api/v1/orgs/${encodeURIComponent(input.organizationSlug)}/runners/grpc/authorize`;
  return lightFetch<AuthorizeRunnerResponse>(path, {
    method: "POST",
    body: { auth_key: input.authKey, node_id: input.nodeId || undefined },
    authenticated: true,
  });
}

// Used by the onboarding "Setup local runner" step. Creates a single-use
// registration token bound to the current organization so the user can
// paste it into the `runner register` CLI command.
interface CreateRunnerTokenResponse {
  token?: string;
  [key: string]: unknown;
}

export async function lightCreateRunnerToken(orgSlug: string): Promise<string | null> {
  const path = `/api/v1/orgs/${encodeURIComponent(orgSlug)}/runners/grpc/tokens`;
  const resp = await lightFetch<CreateRunnerTokenResponse>(path, {
    method: "POST",
    body: {},
    authenticated: true,
  });
  return resp?.token ?? null;
}

export async function lightListRunners(orgSlug: string): Promise<RunnerData[]> {
  const path = `/api/v1/orgs/${encodeURIComponent(orgSlug)}/runners`;
  const resp = await lightFetch<RunnerListResponse>(path, { authenticated: true });
  return resp?.runners ?? [];
}

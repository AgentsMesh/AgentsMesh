import { randomUUID } from "crypto";

import { test, expect, orgSlug, apiBase } from "../../fixtures/blockstore.fixture";
import { getApiBaseUrl } from "../../helpers/env";

// Cross-org isolation is the load-bearing security contract of the block
// store. A leak here means org A reads org B's data. Backend has Go
// integration tests for ACL semantics — this spec exercises the REST tier:
// tenant middleware + org-slug routing must reject a non-member's access
// to a private org's blocks endpoint, regardless of which path they probe.

const SECOND_USER = { email: "dev2@agentsmesh.local", password: "devpass123" };

async function login(email: string, password: string): Promise<string> {
  const res = await fetch(`${getApiBaseUrl()}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) throw new Error(`login(${email}): ${res.status}`);
  const json = (await res.json()) as { token: string };
  return json.token;
}

async function bearer(token: string, path: string, init?: RequestInit): Promise<Response> {
  return fetch(`${getApiBaseUrl()}${path}`, {
    ...(init ?? {}),
    headers: {
      ...((init?.headers as Record<string, string>) ?? {}),
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });
}

test("non-member of an org cannot list/read blocks workspaces in that org", async ({ token }) => {
  // Create a fresh sibling org owned by the dev user. dev2 is NOT a member.
  // We use a unique slug per run so previous test debris doesn't 409 us.
  // POST path is /api/v1/orgs (not /organizations — see RegisterOrganizationRoutes
  // in routes_user.go where the group base is "/orgs").
  const slug = `e2e-iso-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const createOrg = await bearer(token, `/api/v1/orgs`, {
    method: "POST",
    body: JSON.stringify({ name: slug, slug }),
  });
  expect([200, 201]).toContain(createOrg.status);

  // Provision a block workspace inside the new org so there's something
  // attractive to leak — empty-org tests would only exercise the tenant
  // middleware, not the workspace lookup.
  const wsRes = await bearer(token, `/api/v1/orgs/${slug}/blocks/workspaces`, {
    method: "POST",
    body: JSON.stringify({ slug: "default", name: "default" }),
  });
  expect([200, 201, 409]).toContain(wsRes.status);

  // Now dev2 (member of dev-org only) probes the new org. Every endpoint
  // must reject — tenant middleware should kick in before any handler runs.
  const dev2Token = await login(SECOND_USER.email, SECOND_USER.password);
  const probeList = await bearer(dev2Token, `/api/v1/orgs/${slug}/blocks/workspaces`);
  expect(probeList.status).toBeGreaterThanOrEqual(400);
  expect(probeList.status).toBeLessThan(500);
  expect([401, 403, 404]).toContain(probeList.status);

  const probeDefault = await bearer(dev2Token, `/api/v1/orgs/${slug}/blocks/workspaces/default`, {
    method: "POST",
  });
  expect([401, 403, 404]).toContain(probeDefault.status);

  // ApplyOps is a write — must also be rejected even with a well-formed body.
  const probeOps = await bearer(dev2Token, `/api/v1/orgs/${slug}/blocks/ops`, {
    method: "POST",
    body: JSON.stringify({
      workspace_id: "00000000-0000-0000-0000-000000000000",
      ops: [],
      idempotency_key: `iso-${Date.now()}`,
    }),
  });
  expect([401, 403, 404]).toContain(probeOps.status);
});

test("blocks endpoint rejects mismatched org-slug header", async ({ token, isolatedWorkspace }) => {
  // Attack vector: caller is a legit member of dev-org but flips the
  // X-Organization-Slug header to a foreign org slug while keeping the
  // dev-org workspace_id in the body. The handler must NOT trust the body
  // workspace id over the URL slug — workspace lookup is scoped to the
  // tenant middleware's org, so a foreign slug should produce 404 / 403,
  // never a successful write into dev-org.
  const fakeSlug = `e2e-fake-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const blockID = randomUUID();
  const res = await fetch(`${apiBase}/api/v1/orgs/${fakeSlug}/blocks/ops`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "X-Organization-Slug": orgSlug,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      workspace_id: isolatedWorkspace.id,
      ops: [
        {
          op: "createBlock",
          payload: { id: blockID, type: "task", data: { title: "leak", status: "todo" } },
        },
      ],
      idempotency_key: `iso-mismatch-${blockID}`,
    }),
  });
  expect([401, 403, 404]).toContain(res.status);
});

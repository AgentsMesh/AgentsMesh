import { randomUUID } from "crypto";

import { test, expect, orgSlug, apiBase } from "../../fixtures/blockstore.fixture";

// Optimistic concurrency: when two clients both read a block at version T0
// and try to update, the second update must reject (HTTP 409, mapped from
// blockstore.ErrStaleUpdate) rather than silently overwrite. The server
// trusts ExpectedUpdatedAt to detect the conflict; without this spec a
// regression that drops the check would silently lose writes — a bug
// pattern that's invisible to load tests because the response status stays
// 200, only the row contents diverge.

async function postOps(
  token: string,
  workspaceID: string,
  body: Record<string, unknown>,
): Promise<Response> {
  return fetch(`${apiBase}/api/v1/orgs/${orgSlug}/blocks/ops`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "X-Organization-Slug": orgSlug,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ workspace_id: workspaceID, ...body }),
  });
}

async function fetchBlock(
  api: { get<T>(path: string): Promise<T> },
  workspaceID: string,
  rootID: string,
  blockID: string,
): Promise<{ id: string; updated_at: string; data: Record<string, unknown> }> {
  const subtree = await api.get<{
    blocks: Array<{ id: string; updated_at: string; data: Record<string, unknown> }>;
  }>(`/api/v1/orgs/${orgSlug}/blocks/workspaces/${workspaceID}/subtree?root=${rootID}`);
  const found = subtree.blocks.find((b) => b.id === blockID);
  if (!found) throw new Error(`block ${blockID} not found in subtree`);
  return found;
}

test("stale ExpectedUpdatedAt → 409, fresh one → 200", async ({
  api,
  token,
  isolatedWorkspace,
}) => {
  const { id: workspaceID, rootID } = isolatedWorkspace;
  const blockID = randomUUID();
  // Seed a task block under the workspace root. We use api.post (which
  // throws on non-2xx) for the setup so any 4xx surfaces immediately
  // instead of being mistaken for the conflict we're trying to test.
  await api.post(`/api/v1/orgs/${orgSlug}/blocks/ops`, {
    workspace_id: workspaceID,
    ops: [
      {
        op: "createBlock",
        payload: { id: blockID, type: "task", data: { title: "stale", status: "todo" }, text: "stale" },
      },
      {
        op: "addRef",
        payload: { from: rootID, to: blockID, rel: "nest", order_key: `s${Date.now().toString(36)}` },
      },
    ],
    idempotency_key: `e2e-stale-setup-${blockID}`,
  });

  const before = await fetchBlock(api, workspaceID, rootID, blockID);

  // First update with the matching expected_updated_at must succeed.
  // ApplyOps returns 201 for fresh writes, 200 for idempotent replays —
  // either is success here.
  const firstUpdate = await postOps(token, workspaceID, {
    ops: [
      {
        op: "updateBlock",
        payload: {
          id: blockID,
          data: { status: "in_progress" },
          expected_updated_at: before.updated_at,
        },
      },
    ],
    idempotency_key: `e2e-stale-first-${blockID}`,
  });
  expect([200, 201]).toContain(firstUpdate.status);

  // Second update with the SAME (now stale) expected_updated_at must 409.
  // No idempotency key reuse — a stale conflict and a replay are different
  // outcomes, and we don't want the idempotency layer masking the conflict.
  const staleUpdate = await postOps(token, workspaceID, {
    ops: [
      {
        op: "updateBlock",
        payload: {
          id: blockID,
          data: { status: "done" },
          expected_updated_at: before.updated_at,
        },
      },
    ],
    idempotency_key: `e2e-stale-second-${blockID}`,
  });
  expect(staleUpdate.status).toBe(409);

  // Confirm the block contents reflect the FIRST update only — the stale
  // attempt did not partially apply. If 409 is returned but the row was
  // mutated, that's worse than no detection at all.
  const after = await fetchBlock(api, workspaceID, rootID, blockID);
  expect(after.data.status).toBe("in_progress");

  // A fresh expected_updated_at (post-first-write) must again succeed.
  const refreshed = await postOps(token, workspaceID, {
    ops: [
      {
        op: "updateBlock",
        payload: {
          id: blockID,
          data: { status: "done" },
          expected_updated_at: after.updated_at,
        },
      },
    ],
    idempotency_key: `e2e-stale-third-${blockID}`,
  });
  expect([200, 201]).toContain(refreshed.status);
});

test("update without expected_updated_at always succeeds (no version check)", async ({
  api,
  isolatedWorkspace,
}) => {
  // The optimistic check is opt-in: a payload without the field must accept
  // even after the row has been mutated by another writer. Documents the
  // last-write-wins escape hatch the field is the explicit signal for.
  const { id: workspaceID, rootID } = isolatedWorkspace;
  const blockID = randomUUID();
  await api.post(`/api/v1/orgs/${orgSlug}/blocks/ops`, {
    workspace_id: workspaceID,
    ops: [
      { op: "createBlock", payload: { id: blockID, type: "task", data: { title: "lww", status: "todo" } } },
      { op: "addRef", payload: { from: rootID, to: blockID, rel: "nest", order_key: `l${Date.now().toString(36)}` } },
    ],
    idempotency_key: `e2e-lww-setup-${blockID}`,
  });

  await api.post(`/api/v1/orgs/${orgSlug}/blocks/ops`, {
    workspace_id: workspaceID,
    ops: [{ op: "updateBlock", payload: { id: blockID, data: { status: "in_progress" } } }],
    idempotency_key: `e2e-lww-first-${blockID}`,
  });
  await api.post(`/api/v1/orgs/${orgSlug}/blocks/ops`, {
    workspace_id: workspaceID,
    ops: [{ op: "updateBlock", payload: { id: blockID, data: { status: "done" } } }],
    idempotency_key: `e2e-lww-second-${blockID}`,
  });
});

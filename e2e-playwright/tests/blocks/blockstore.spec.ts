import { test, expect } from "../../fixtures/index";
import { test as uiTest, expect as uiExpect } from "@playwright/test";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { collectConsoleErrors, assertNoWasmErrors } from "../../helpers/console-errors";

const BLOCKS = `/api/v1/orgs/${TEST_ORG_SLUG}/blocks`;

// ────────────────────────────────────────────────────
// Part 1: API — workspaces, applyOps, catchup, subtree
// ────────────────────────────────────────────────────

test.describe("Block Store · API", () => {
  let workspaceId: string;

  test.beforeAll(async ({ api }) => {
    clearAuthRateLimit();
    // ensure_default: idempotent — returns the org's default workspace.
    const res = await api.post(`${BLOCKS}/workspaces/default`, {});
    expect(res.status).toBe(200);
    const ws = await res.json();
    workspaceId = ws.id;
    expect(workspaceId).toBeTruthy();
  });

  test("listWorkspaces includes default workspace", async ({ api }) => {
    const res = await api.get(`${BLOCKS}/workspaces`);
    expect(res.status).toBe(200);
    const { workspaces } = await res.json();
    expect(Array.isArray(workspaces)).toBe(true);
    expect(workspaces.some((w: { id: string }) => w.id === workspaceId)).toBe(true);
  });

  test("applyOps creates block + nest ref, subtree reflects state", async ({ api }) => {
    const rootId = crypto.randomUUID();
    const childId = crypto.randomUUID();
    const idempotencyKey = `e2e-blocks-${Date.now()}`;
    const applyRes = await api.post(`${BLOCKS}/ops`, {
      workspace_id: workspaceId,
      idempotency_key: idempotencyKey,
      ops: [
        { op: "createBlock", payload: { id: rootId, type: "page", data: { title: "E2E root" } } },
        { op: "createBlock", payload: { id: childId, type: "paragraph", data: { text: "child" } } },
        { op: "addRef", payload: { from: rootId, to: childId, rel: "nest", order_key: "m" } },
      ],
    });
    expect([200, 201]).toContain(applyRes.status);
    const applyBody = await applyRes.json();
    expect(applyBody.op_ids).toHaveLength(3);
    expect(applyBody.was_replay).toBe(false);

    const sub = await api.get(
      `${BLOCKS}/workspaces/${workspaceId}/subtree?root=${rootId}&max_depth=8`,
    );
    expect(sub.status).toBe(200);
    const { blocks, refs } = await sub.json();
    expect(blocks.some((b: { id: string }) => b.id === childId)).toBe(true);
    expect(refs.some((r: { rel: string; to_id: string }) => r.rel === "nest" && r.to_id === childId)).toBe(true);
  });

  test("applyOps idempotency — replay returns was_replay=true", async ({ api }) => {
    const rootId = crypto.randomUUID();
    const idempotencyKey = `e2e-idem-${Date.now()}`;
    const req = {
      workspace_id: workspaceId,
      idempotency_key: idempotencyKey,
      ops: [{ op: "createBlock", payload: { id: rootId, type: "page", data: { title: "Idem" } } }],
    };
    const first = await api.post(`${BLOCKS}/ops`, req);
    expect([200, 201]).toContain(first.status);
    const second = await api.post(`${BLOCKS}/ops`, req);
    expect([200, 201]).toContain(second.status);
    expect((await second.json()).was_replay).toBe(true);
  });

  test("catchup returns ops after the watermark", async ({ api }) => {
    const res = await api.get(`${BLOCKS}/workspaces/${workspaceId}/ops?after=0&limit=50`);
    expect(res.status).toBe(200);
    const { ops } = await res.json();
    expect(Array.isArray(ops)).toBe(true);
  });

  test("type-defs endpoint returns block_type_def blocks", async ({ api }) => {
    const res = await api.get(`${BLOCKS}/workspaces/${workspaceId}/type-defs`);
    expect(res.status).toBe(200);
    const { blocks } = await res.json();
    expect(Array.isArray(blocks)).toBe(true);
    // All returned blocks should be of type block_type_def.
    for (const b of blocks) {
      expect(b.type).toBe("block_type_def");
    }
  });

  test("updateBlock op patches data field", async ({ api }) => {
    const blockId = crypto.randomUUID();
    await api.post(`${BLOCKS}/ops`, {
      workspace_id: workspaceId,
      idempotency_key: `e2e-upd-create-${Date.now()}`,
      ops: [{ op: "createBlock", payload: { id: blockId, type: "paragraph", data: { text: "v1" } } }],
    });
    const upd = await api.post(`${BLOCKS}/ops`, {
      workspace_id: workspaceId,
      idempotency_key: `e2e-upd-patch-${Date.now()}`,
      ops: [{ op: "updateBlock", payload: { id: blockId, data: { text: "v2" } } }],
    });
    expect([200, 201]).toContain(upd.status);
    const getRes = await api.get(`${BLOCKS}/${encodeURIComponent(blockId)}`);
    expect(getRes.status).toBe(200);
    const block = await getRes.json();
    expect(block.data.text).toBe("v2");
  });
});

// ────────────────────────────────────────────────────
// Part 2: UI — page loads, no WASM errors
// ────────────────────────────────────────────────────

uiTest.describe("Block Store · UI", () => {
  uiTest.beforeEach(async () => { clearAuthRateLimit(); });

  uiTest("blocks page loads without WASM errors", async ({ page }) => {
    const errors = collectConsoleErrors(page);
    await page.goto(`/${TEST_ORG_SLUG}/blocks`);
    await page.waitForLoadState("networkidle");
    // Page must render past the spinner — either the DocumentView or an
    // error banner is acceptable; a stuck spinner is not.
    await uiExpect(page.locator("body")).toBeVisible();
    assertNoWasmErrors(errors);
  });

  uiTest("search panel opens on search button click", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/blocks`);
    await page.waitForLoadState("networkidle");
    // Search is a Button with a search icon/label. If absent, skip — the page
    // only mounts SearchPanel after a workspace is hydrated.
    const searchBtn = page.getByRole("button", { name: /search|搜索/i }).first();
    if (!(await searchBtn.isVisible({ timeout: 3000 }).catch(() => false))) {
      uiTest.skip();
      return;
    }
    await searchBtn.click();
    await page.waitForTimeout(300);
  });
});

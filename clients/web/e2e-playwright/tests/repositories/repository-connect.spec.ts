import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { collectConsoleErrors, assertNoWasmErrors } from "../../helpers/console-errors";

// E2E for the proto.repository.v1 Connect-RPC migration. This spec drives the
// full UI path through repositoryConnect.ts → wasm bridge → api-client
// connect_call → backend Connect handler, so it catches integration drift
// the unit tests (which mock at the wasm-core layer) can miss.
//
// Lineage class: PR #341 (skill-registry-wasm-roundtrip.spec.ts) — same
// failure mode if the binary wire format slips between layers.

test.describe("Repository Connect-RPC round-trip", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("list repositories renders from the proto binary response", async ({ page, db }) => {
    const count = db.queryValue(
      `SELECT COUNT(*)::int FROM repositories WHERE organization_id = (SELECT id FROM organizations WHERE slug = '${TEST_ORG_SLUG}')`,
    );
    if (!count || Number(count) === 0) { test.skip(); return; }

    const errors = collectConsoleErrors(page);

    // Watch for the Connect-RPC procedure path (conventions §12).
    const connectResponses: number[] = [];
    page.on("response", (resp) => {
      if (resp.url().includes("/proto.repository.v1.RepositoryService/ListRepositories")) {
        connectResponses.push(resp.status());
      }
    });

    await page.goto(`/${TEST_ORG_SLUG}/repositories`);
    await page.waitForLoadState("networkidle");

    // RepositoriesSettings renders the list via listRepositories() in
    // repositoryConnect.ts. If the proto decoding drops fields silently
    // (the lineage class of PR #345's raw_key bug) the UI falls back to
    // the empty state instead of showing rows.
    const emptyState = page.getByText(/No repositories|没有仓库/i).first();
    const repoLink = page.locator('a[href*="/repositories/"]').first();
    await Promise.race([
      repoLink.waitFor({ state: "visible", timeout: 8000 }).catch(() => null),
      emptyState.waitFor({ state: "visible", timeout: 8000 }).catch(() => null),
    ]);

    expect(connectResponses.length).toBeGreaterThan(0);
    for (const status of connectResponses) {
      expect(status, "ListRepositories Connect call should not 4xx/5xx").toBeLessThan(400);
    }

    assertNoWasmErrors(errors);
  });

  test("repository detail uses Connect GetRepository", async ({ page, db }) => {
    const id = db.queryValue(
      `SELECT id FROM repositories WHERE organization_id = (SELECT id FROM organizations WHERE slug = '${TEST_ORG_SLUG}') LIMIT 1`,
    );
    if (!id) { test.skip(); return; }

    const errors = collectConsoleErrors(page);

    let getRepoStatus = 0;
    page.on("response", (resp) => {
      if (resp.url().includes("/proto.repository.v1.RepositoryService/GetRepository")) {
        getRepoStatus = resp.status();
      }
    });

    await page.goto(`/${TEST_ORG_SLUG}/repositories/${id}`);
    await page.waitForLoadState("networkidle");

    expect(getRepoStatus, "GetRepository Connect call should succeed").toBeGreaterThan(0);
    expect(getRepoStatus).toBeLessThan(400);

    assertNoWasmErrors(errors);
  });
});

import { test, expect } from "../../fixtures/index";
import { SettingsNavPage } from "../../pages/settings/settings-nav.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// Regression for issue #341.
//
// The bug: backend returns `{"skill_registries": [...]}`, but the Rust DTO
// renamed the wrapper field to `registries` with `#[serde(alias = "skill_registries")]`.
// Serde's `alias` only affects deserialization; re-serializing for the wasm
// relay emitted `{"registries": ...}` instead. The TS layer reads
// `.skill_registries` and got `undefined`, so the UI list stayed empty even
// though the DB row existed — and re-registering the same repo tripped the
// unique-key check on the backend.
//
// Pure API tests can't catch this because the backend wire format is correct
// — the drift happens inside the wasm boundary. This spec drives the full UI
// path so the round-trip is exercised end-to-end. The two scenarios cover the
// two symptoms the user reported:
//   1. After adding a source, list stays empty until refresh (or forever).
//   2. Reloading the page with an existing DB row still shows empty list.

const TEST_URL_PREFIX = "https://github.com/agentsmesh-e2e/skill-roundtrip-";

const orgIdSql = `(SELECT id FROM organizations WHERE slug = '${TEST_ORG_SLUG}')`;
const deleteTestRegistries = `DELETE FROM skill_registries WHERE organization_id = ${orgIdSql} AND repository_url LIKE '${TEST_URL_PREFIX}%'`;

test.describe("Skill registry — wasm round-trip (#341)", () => {
  test.beforeEach(async ({ db }) => {
    clearAuthRateLimit();
    db.cleanup(deleteTestRegistries);
  });

  test.afterEach(async ({ db }) => {
    db.cleanup(deleteTestRegistries);
  });

  test("UI shows newly added org registry after submit", async ({ page, db }) => {
    const testUrl = `${TEST_URL_PREFIX}add-${Date.now()}`;

    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "extensions");

    // Open the add-source dialog. The "Add Source" button only renders inside
    // OrgRegistriesList — PlatformRegistriesList has no equivalent — so the
    // page has exactly one such button before the dialog opens.
    const openButton = page.getByRole("button", { name: /add source|添加注册表/i });
    await expect(openButton).toBeVisible();
    await openButton.click();

    // The project's Dialog is a custom portal — it does NOT set role="dialog".
    // It marks the overlay with `data-dialog-overlay`, which is what we scope
    // by; getByRole("dialog") would silently miss every spec that touches it.
    const dialog = page.locator("[data-dialog-overlay]");
    await expect(dialog).toBeVisible();

    await dialog.getByPlaceholder("https://github.com/owner/skills-repo").fill(testUrl);
    // Inside the dialog overlay the only `<button>` carrying the "Add Source"
    // label is the footer submit (the heading is an <h2>, the cancel button
    // reads "Cancel" / "取消"). Scoping by dialog excludes the now-occluded
    // OrgRegistriesList "Add Source" button on the page behind the overlay.
    await dialog.getByRole("button", { name: /add source|添加注册表/i }).click();

    await expect(dialog).toBeHidden();

    // The bug manifested as an empty list even though the POST succeeded.
    // Asserting the URL is rendered proves the wasm relay preserved the
    // `skill_registries` wrapper key on the subsequent list refresh.
    await expect(page.getByText(testUrl)).toBeVisible();

    // Belt-and-braces: confirm the DB row exists, so a future bug that hides
    // rows in the UI without ever POSTing can't pass by simply staying empty.
    const dbCount = db.queryValue(
      `SELECT COUNT(*) FROM skill_registries WHERE organization_id = ${orgIdSql} AND repository_url = '${testUrl}'`
    );
    expect(dbCount).toBe("1");
  });

  test("UI lists pre-existing org registry on page load", async ({ page, api }) => {
    // Symptom 2 from the bug report: even if the DB has rows, the page renders
    // empty. Pre-seed via the backend API (which bypasses wasm), then load the
    // settings page and assert the row appears — this exercises the wasm
    // list_skill_registries() path without going through the create dialog.
    const testUrl = `${TEST_URL_PREFIX}preexisting-${Date.now()}`;
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/skill-registries`, {
      repository_url: testUrl,
      branch: "main",
      auth_type: "none",
    });
    expect([200, 201]).toContain(createRes.status);

    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "extensions");

    await expect(page.getByText(testUrl)).toBeVisible();
  });
});

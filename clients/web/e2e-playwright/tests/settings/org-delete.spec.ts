import { test, expect } from "../../fixtures/index";
import { uniqueSuffix } from "../../helpers/test-data";

/**
 * Regression for the production bug where DELETE /api/v1/orgs/:slug always
 * returned 500 due to `DELETE FROM pod_bindings WHERE channel_id IN (...)` —
 * pod_bindings has no channel_id column (see migration 000001). Every user
 * attempting to delete an organization hit "An internal error occurred".
 *
 * Three layers of guard now cover the fix: repo-level cleanup test in
 * backend/internal/infra/organization_repo_test.go, handler integration test
 * in backend/internal/api/rest/v1/organizations_delete_test.go, and this
 * full-stack flow.
 */
test.describe("Organization Deletion", () => {
  test("owner deletes org via API and the row is gone", async ({ api, db }) => {
    const slug = `e2e-delete-${uniqueSuffix()}`.toLowerCase().replace(/_/g, "-");

    const createRes = await api.post("/api/v1/orgs", {
      name: "E2E Delete Target",
      slug,
    });
    expect(createRes.status, await createRes.text()).toBe(201);

    const delRes = await api.delete(`/api/v1/orgs/${slug}`);
    expect(delRes.status, await delRes.text()).toBe(200);

    const getRes = await api.get(`/api/v1/orgs/${slug}`);
    expect(getRes.status).toBe(404);

    // Belt-and-suspenders teardown in case the test failed mid-way.
    db.cleanup(`DELETE FROM organizations WHERE slug = '${slug}'`);
  });

  test("UI: clicking Delete Organization on settings page deletes and redirects", async ({
    page,
    api,
    db,
  }) => {
    const slug = `e2e-uidel-${uniqueSuffix()}`.toLowerCase().replace(/_/g, "-");

    const createRes = await api.post("/api/v1/orgs", {
      name: "E2E UI Delete",
      slug,
    });
    expect(createRes.status, await createRes.text()).toBe(201);

    try {
      await page.goto(`/${slug}/settings?scope=organization&tab=general`);
      await page.waitForLoadState("networkidle");

      // Danger Zone "Delete Organization" button — case-insensitive to be
      // resilient against locale shifts (the button label is i18n-driven).
      await page.getByRole("button", { name: /delete organization/i }).click();

      // The project's Dialog component (clients/web/src/components/ui/dialog.tsx)
      // does NOT set role="dialog" — it just renders a portal with
      // `data-dialog-overlay`. Use that attribute instead of getByRole.
      const dialog = page.locator("[data-dialog-overlay]");
      await expect(dialog).toBeVisible();
      // Two "Delete Organization" buttons exist once the dialog opens: the
      // Danger Zone trigger and the confirm action. Pick the one inside the
      // overlay to avoid re-clicking the trigger.
      await dialog
        .getByRole("button", { name: /delete organization/i })
        .click();

      // UI feedback: either a success toast or sidebar/URL no longer shows the
      // deleted org. The post-delete redirect can be "/" or another org slug
      // depending on the user's other memberships — assert the deleted slug is
      // gone instead of pinning the destination.
      await expect(page).not.toHaveURL(new RegExp(`/${slug}(/|$|\\?)`), {
        timeout: 10_000,
      });

      const getRes = await api.get(`/api/v1/orgs/${slug}`);
      expect(getRes.status).toBe(404);
    } finally {
      db.cleanup(`DELETE FROM organizations WHERE slug = '${slug}'`);
    }
  });
});

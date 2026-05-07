import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * Regression for: Spawn Pod from a ticket must seed both the prompt
 * (ticket title + description) and the repository (ticket.repository_id).
 *
 * Prior bugs:
 * - SpawnPodButton built a TicketContext that never carried `description`,
 *   so the preset prompt generator only emitted "Work on ticket SLUG: title".
 * - usePrefsAutoFill wrote `lastRepositoryId` into the form before the
 *   ticket-context effect ran; the latter's `!selectedRepository` guard
 *   then no-op'd, so a previously-used repository overrode the ticket's.
 */
test.describe("Ticket Spawn Pod context", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  let createdSlug: string | null = null;

  test.afterEach(async ({ api }) => {
    if (createdSlug) {
      await api.delete(`/api/v1/orgs/${TEST_ORG_SLUG}/tickets/${createdSlug}`);
      createdSlug = null;
    }
  });

  test("modal prefills prompt with ticket body and repo with ticket.repository_id", async ({ page, api }) => {
    const reposRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/repositories`);
    const repositories = (await reposRes.json()).repositories ?? [];
    if (repositories.length < 2) { test.skip(); return; }

    const [ticketRepo, otherRepo] = repositories;

    const description = "Reproduce locally then bisect across last week's commits.";
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/tickets`, {
      title: "E2E Spawn Pod From Ticket",
      description,
      repository_id: ticketRepo.id,
    });
    const created = await createRes.json();
    createdSlug = created.ticket?.slug || created.slug;
    expect(createdSlug).toBeTruthy();

    // Seed the persisted pod-creation prefs with a *different* repository so
    // we can prove ticket context wins over saved prefs.
    await page.addInitScript((repoId: number) => {
      window.localStorage.setItem(
        "agentsmesh-pod-creation",
        JSON.stringify({
          state: {
            lastAgentSlug: null,
            lastRepositoryId: repoId,
            lastCredentialProfileId: null,
            lastBranchName: null,
          },
          version: 0,
        }),
      );
    }, otherRepo.id);

    await page.goto(`/${TEST_ORG_SLUG}/tickets/${createdSlug}`);
    await page.waitForLoadState("networkidle");

    await page.getByRole("button", { name: /spawn pod/i }).first().click();
    await page.locator('[role="dialog"]').first().waitFor({ state: "visible" });

    const promptArea = page.locator('[role="dialog"] textarea').first();
    // Use toHaveValue: textarea content set via React prop lives on `.value`,
    // not `textContent` (which is what toContainText reads).
    await expect(promptArea).toHaveValue(/E2E Spawn Pod From Ticket/);
    await expect(promptArea).toHaveValue(new RegExp(escapeRegex(description)));

    // Repository select lives inside collapsed Advanced Options; expand it.
    const advancedTrigger = page
      .locator('[role="dialog"]')
      .getByRole("button", { name: /advanced/i })
      .first();
    if (await advancedTrigger.isVisible().catch(() => false)) {
      await advancedTrigger.click();
    }

    const repoSelect = page.locator('[role="dialog"] select#repository-select');
    await expect(repoSelect).toHaveValue(String(ticketRepo.id));
  });
});

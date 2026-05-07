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
 * Prior bug: SpawnPodButton built a TicketContext that never carried
 * `description`, so the preset prompt generator only emitted
 * "Work on ticket SLUG: title" and the prompt body was lost.
 *
 * The "prefs do not override ticket repo" half is covered by the unit test
 * usePrefsAutoFill.test.ts (deterministic) — keeping the e2e focused on the
 * UI prefill path so it doesn't fight zustand-persist hydration timing.
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
    if (repositories.length === 0) { test.skip(); return; }

    const ticketRepo = repositories[0];
    const description = "Reproduce locally then bisect across last week's commits.";
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/tickets`, {
      title: "E2E Spawn Pod From Ticket",
      description,
      repository_id: ticketRepo.id,
    });
    const created = await createRes.json();
    createdSlug = created.ticket?.slug || created.slug;
    expect(createdSlug).toBeTruthy();

    page.on("pageerror", (err) => console.error("[pageerror]", err.message));
    page.on("console", (msg) => {
      if (msg.type() === "error") console.error("[console.error]", msg.text());
    });

    await page.goto(`/${TEST_ORG_SLUG}/tickets/${createdSlug}`);
    await page.waitForLoadState("domcontentloaded");

    // Wait for the ticket title to render — proves currentTicket loaded and
    // SpawnPodButton mounted (InlineEditableText renders the title in a
    // <button>, not a heading, so use getByText).
    await expect(page.getByText("E2E Spawn Pod From Ticket").first())
      .toBeVisible({ timeout: 30000 });

    const spawnBtn = page.getByRole("button", { name: /spawn pod/i }).first();
    await expect(spawnBtn).toBeVisible({ timeout: 10000 });
    await spawnBtn.scrollIntoViewIfNeeded();
    await spawnBtn.click();

    // CreatePodModal renders <h2 id="create-pod-title">.
    await expect(page.locator("#create-pod-title")).toBeVisible({ timeout: 15000 });

    const promptArea = page.locator('[role="dialog"] textarea').first();
    await expect(promptArea).toHaveValue(/E2E Spawn Pod From Ticket/);
    await expect(promptArea).toHaveValue(new RegExp(escapeRegex(description)));

    // Repository select sits inside Advanced Options (collapsed by default).
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

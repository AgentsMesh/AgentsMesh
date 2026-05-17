import { test, expect } from "../../fixtures/index";
import { SettingsNavPage } from "../../pages/settings/settings-nav.page";
import { TEST_ORG_SLUG, TEST_USER } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// Regression guard for the May 2026 credentials breakage:
//
//   Bug A — Add Credentials dialog rendered only Name + Description
//           (ENV-derived secret/text fields were swallowed because the
//           renderer accessed `.schema?.credential_fields` on a payload
//           that the wasm bridge already un-wrapped).
//
//   Bug B — The credentials list stayed empty after navigation AND after
//           a successful POST, because the Rust list-response type modelled
//           the grouped `{items:[{agent_slug,profiles}]}` payload as a flat
//           profile list and reverse-serialisation failed silently.
//
// These cases run against the dev stack (`bazel run //deploy/dev:up`) and
// drive the real renderer + real backend through the wasm bridge — the only
// layer where the regression actually surfaced.

const AGENT_SLUG = "claude-code";
const NAME_PREFIX = "E2E Credential";

function unique(label: string): string {
  return `${NAME_PREFIX} ${label} ${Date.now()}`;
}

test.describe("Personal Agent Credentials", () => {
  test.beforeEach(async () => {
    clearAuthRateLimit();
  });

  // Bug A regression.
  test("add-credential dialog renders ENV-derived secret + text fields", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", `agents/${AGENT_SLUG}`);

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();

    // Static fields — present even when the schema is empty. Asserting them
    // first keeps the failure message useful when the dialog itself fails
    // to mount.
    await expect(page.locator("#cred-name")).toBeVisible();
    await expect(page.locator("#cred-desc")).toBeVisible();

    // ENV-derived fields from the built-in claude-code AgentFile. If any of
    // these three goes missing, the dialog is silently degraded to the
    // pre-fix two-field state and the user can't enter their key.
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveAttribute("type", "password");
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toHaveAttribute("type", "password");
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toHaveAttribute("type", "text");
  });

  // Bug B regression — seeded profile must appear in the list.
  test("seeded credential profile renders in the list", async ({ page, api, db }) => {
    const profileName = unique("seeded");
    db.cleanup(
      `DELETE FROM user_agent_credential_profiles WHERE name LIKE '${NAME_PREFIX}%'`
    );

    await api.login(TEST_USER.email, TEST_USER.password);
    const res = await api.post(`/api/v1/users/agent-credentials/agents/${AGENT_SLUG}`, {
      name: profileName,
      description: "seeded by e2e",
      credentials: { ANTHROPIC_API_KEY: "sk-ant-e2e-seeded" },
    });
    expect([200, 201]).toContain(res.status);

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/${AGENT_SLUG}`);

      // The CredentialsSection renders each profile's name in a span;
      // getByText pins to the actual DOM node, not the page-level text dump
      // the legacy spec used.
      await expect(page.getByText(profileName, { exact: true })).toBeVisible({
        timeout: 15_000,
      });
    } finally {
      db.cleanup(
        `DELETE FROM user_agent_credential_profiles WHERE name LIKE '${NAME_PREFIX}%'`
      );
    }
  });

  // Bug A + Bug B end-to-end — full UI create flow.
  test("UI create flow: new credential appears in the list after submit", async ({ page, db }) => {
    const profileName = unique("ui-create");
    db.cleanup(
      `DELETE FROM user_agent_credential_profiles WHERE name LIKE '${NAME_PREFIX}%'`
    );

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/${AGENT_SLUG}`);

      await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
        .first().click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();

      await page.locator("#cred-name").fill(profileName);
      await page.locator("#cred-desc").fill("created via UI");
      await page.locator("#cred-ANTHROPIC_API_KEY").fill("sk-ant-e2e-ui-created");

      await page.getByRole("button", { name: /^(Create|创建)$/ }).click();

      // After submit the dialog closes and loadData() refreshes the list.
      // Pre-fix this is where the regression surfaced — the POST succeeded
      // but the refreshed list was still empty.
      await expect(page.getByText(profileName, { exact: true })).toBeVisible({
        timeout: 15_000,
      });
    } finally {
      db.cleanup(
        `DELETE FROM user_agent_credential_profiles WHERE name LIKE '${NAME_PREFIX}%'`
      );
    }
  });
});

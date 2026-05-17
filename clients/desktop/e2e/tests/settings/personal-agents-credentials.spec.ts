import { test, expect } from "../../fixtures";
import { TEST_ORG_SLUG, TEST_USER } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";

// Desktop counterpart of
// clients/web/e2e-playwright/tests/settings/personal-agents-credentials.spec.ts.
//
// The renderer reuses AgentConfigPage + AgentCredentialsSettings from the
// web tree, and Desktop ships the same Rust Core (via node-bridge). Both
// regressions (Bug A: dialog missing ENV fields; Bug B: list empty after
// load and after a successful POST) must be verified on the Electron
// build too — the data path runs through the native dylib here, not the
// browser wasm artifact, so a fix that only landed in the wasm bridge
// would still leak through.

const AGENT_SLUG = "claude-code";
const NAME_PREFIX = "Desktop E2E Credential";

function unique(label: string): string {
  return `${NAME_PREFIX} ${label} ${Date.now()}`;
}

async function gotoAgentSettings(page: import("@playwright/test").Page): Promise<void> {
  await gotoHash(
    page,
    `/${TEST_ORG_SLUG}/settings?scope=personal&tab=agents/${AGENT_SLUG}`
  );
}

test.describe("Desktop · Personal Agent Credentials", () => {
  // Bug A regression.
  test("add-credential dialog renders ENV-derived secret + text fields", async ({ page }) => {
    await gotoAgentSettings(page);

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();

    await expect(page.locator("#cred-name")).toBeVisible();
    await expect(page.locator("#cred-desc")).toBeVisible();

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
      description: "seeded by desktop e2e",
      credentials: { ANTHROPIC_API_KEY: "sk-ant-desktop-seeded" },
    });
    expect([200, 201]).toContain(res.status);

    try {
      await gotoAgentSettings(page);
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
      await gotoAgentSettings(page);

      await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
        .first().click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();

      await page.locator("#cred-name").fill(profileName);
      await page.locator("#cred-desc").fill("created via desktop UI");
      await page.locator("#cred-ANTHROPIC_API_KEY").fill("sk-ant-desktop-ui-created");

      await page.getByRole("button", { name: /^(Create|创建)$/ }).click();

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

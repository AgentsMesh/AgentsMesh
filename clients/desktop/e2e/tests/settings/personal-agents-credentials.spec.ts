import { test, expect } from "../../fixtures";
import { TEST_ORG_SLUG, TEST_USER } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";

// Desktop counterpart of
// clients/web/e2e-playwright/tests/settings/personal-agents-credentials.spec.ts.
//
// The renderer reuses AgentConfigPage + AgentCredentialsSettings from the
// web tree; Desktop drives the same Rust Core via node-bridge. After the
// credential-form rework the dialog spec lives entirely in the renderer —
// these tests verify the new XOR / custom-env UX still wires through the
// native dylib path.

const NAME_PREFIX = "Desktop E2E Credential";

function unique(label: string): string {
  return `${NAME_PREFIX} ${label} ${Date.now()}`;
}

async function gotoAgentSettings(
  page: import("@playwright/test").Page,
  slug: string
): Promise<void> {
  await gotoHash(
    page,
    `/${TEST_ORG_SLUG}/settings?scope=personal&tab=agents/${slug}`
  );
}

test.describe("Desktop · Claude Code (XOR auth)", () => {
  test("toggling auth method without typing keeps the stored secret", async ({ page, api, db }) => {
    const bundleName = unique("half-switch");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      kind: "credential",
      data: { ANTHROPIC_AUTH_TOKEN: "sk-ant-desktop-keep-toggle" },
    });

    try {
      await gotoAgentSettings(page, "claude-code");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      // Toggle to API Key over the IPC→node-bridge path but type nothing, save.
      await page.getByTestId("oneof-option-anthropic_auth-ANTHROPIC_API_KEY").click();
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-name")).toHaveCount(0);

      // Finding #1: a half-finished switch must not delete the stored token.
      const tokenKept = db.queryValue(
        `SELECT count(*) FROM env_bundles WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_AUTH_TOKEN'`
      );
      expect(tokenKept).toBe("1");
      const keyAbsent = db.queryValue(
        `SELECT count(*) FROM env_bundles WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_API_KEY'`
      );
      expect(keyAbsent).toBe("0");
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("base URL field shows the do-not-embed-secrets hint", async ({ page }) => {
    await gotoAgentSettings(page, "claude-code");
    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toBeVisible();
    // Finding #2: base URL round-trips in plaintext → warn against embedding tokens.
    await expect(page.getByText(/plaintext|明文/).first()).toBeVisible();
  });

  test("editing a corrupt bundle's name degrades instead of 500", async ({ page, api, db }) => {
    const bundleName = unique("corrupt");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      kind: "credential",
      data: { ANTHROPIC_AUTH_TOKEN: "sk-ant-desktop-corrupt" },
    });
    db.exec(
      `UPDATE env_bundles SET data = '{"ANTHROPIC_AUTH_TOKEN":"garbage-not-ciphertext"}' WHERE name = '${bundleName}'`
    );

    const renamed = `${bundleName} renamed`;
    try {
      await gotoAgentSettings(page, "claude-code");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await page.locator("#cred-name").fill(renamed);
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();

      // Finding #3: the update degrades over IPC→backend instead of 500ing on
      // the historical corrupt secret the write never touched.
      await expect(page.locator("#cred-name")).toHaveCount(0);
      await expect(page.getByText(renamed, { exact: true })).toBeVisible({ timeout: 15_000 });
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("dialog shows Base URL first + RadioGroup toggles inputs", async ({ page }) => {
    await gotoAgentSettings(page, "claude-code");

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();

    await expect(page.locator("#cred-name")).toBeVisible();
    await expect(page.locator("#cred-desc")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toHaveCount(0);

    await page.getByTestId("oneof-option-anthropic_auth-ANTHROPIC_AUTH_TOKEN").click();
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveCount(0);
  });

  test("seeded credential bundle renders in the list", async ({ page, api, db }) => {
    const bundleName = unique("seeded");
    db.cleanup(
      `DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`
    );

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      description: "seeded by desktop e2e",
      kind: "credential",
      data: { ANTHROPIC_API_KEY: "sk-ant-desktop-seeded" },
    });

    try {
      await gotoAgentSettings(page, "claude-code");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });
    } finally {
      db.cleanup(
        `DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`
      );
    }
  });

  test("UI create flow: new credential appears in the list after submit", async ({ page, db }) => {
    const bundleName = unique("ui-create");
    db.cleanup(
      `DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`
    );

    try {
      await gotoAgentSettings(page, "claude-code");

      await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
        .first().click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();

      await page.locator("#cred-name").fill(bundleName);
      await page.locator("#cred-desc").fill("created via desktop UI");
      await page.locator("#cred-ANTHROPIC_BASE_URL").fill("https://proxy.anthropic.example");
      await page.locator("#cred-ANTHROPIC_API_KEY").fill("sk-ant-desktop-ui-created");

      await page.getByRole("button", { name: /^(Create|创建)$/ }).click();
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });
    } finally {
      db.cleanup(
        `DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`
      );
    }
  });

  test("edit dialog prefills the non-secret Base URL and keeps the secret blank", async ({ page, api, db }) => {
    const bundleName = unique("edit-prefill");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      kind: "credential",
      data: {
        ANTHROPIC_BASE_URL: "https://proxy.anthropic.example",
        ANTHROPIC_AUTH_TOKEN: "sk-ant-desktop-token",
      },
    });

    try {
      await gotoAgentSettings(page, "claude-code");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();

      // Read fix over the IPC→node-bridge→Rust path: the non-secret Base URL
      // prefills its stored value while the secret stays hidden.
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL"))
        .toHaveValue("https://proxy.anthropic.example");
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toHaveValue("");
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN"))
        .toHaveAttribute("placeholder", /Leave empty to keep existing|留空保持现有值/);
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveCount(0);
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("editing the Base URL preserves the stored Auth Token across a re-edit", async ({ page, api, db }) => {
    const bundleName = unique("edit-preserve");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      kind: "credential",
      data: {
        ANTHROPIC_BASE_URL: "https://old.anthropic.example",
        ANTHROPIC_AUTH_TOKEN: "sk-ant-desktop-keep",
      },
    });

    try {
      await gotoAgentSettings(page, "claude-code");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      // First edit: change ONLY the Base URL, leave the Auth Token blank.
      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL"))
        .toHaveValue("https://old.anthropic.example");
      await page.locator("#cred-ANTHROPIC_BASE_URL").fill("https://new.anthropic.example");
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();

      // The dialog only closes after update + reload are awaited, so the input
      // disappearing means the write already round-tripped through the dylib.
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toHaveCount(0);

      // Write fix: a non-secret-only update must NOT wipe the untouched secret.
      const tokenKept = db.queryValue(
        `SELECT count(*) FROM env_bundles
         WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_AUTH_TOKEN'`
      );
      expect(tokenKept).toBe("1");

      // Re-edit (二次编辑): new Base URL shows, Auth Token radio still selected.
      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL"))
        .toHaveValue("https://new.anthropic.example");
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveCount(0);
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("switching auth method drops the old credential (no XOR leftover)", async ({ page, api, db }) => {
    const bundleName = unique("xor-switch");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      kind: "credential",
      data: { ANTHROPIC_AUTH_TOKEN: "sk-ant-desktop-old" },
    });

    try {
      await gotoAgentSettings(page, "claude-code");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
      // Switch to API Key over the IPC→node-bridge path and fill it.
      await page.getByTestId("oneof-option-anthropic_auth-ANTHROPIC_API_KEY").click();
      await page.locator("#cred-ANTHROPIC_API_KEY").fill("sk-ant-desktop-new");
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveCount(0);

      // The deselected Auth Token must be dropped, not merge-preserved — else
      // the pod would receive both Anthropic auth env vars.
      const tokenGone = db.queryValue(
        `SELECT count(*) FROM env_bundles
         WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_AUTH_TOKEN'`
      );
      expect(tokenGone).toBe("0");
      const keyPresent = db.queryValue(
        `SELECT count(*) FROM env_bundles
         WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_API_KEY'`
      );
      expect(keyPresent).toBe("1");
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });
});

test.describe("Desktop · Loopal (custom env)", () => {
  test("explicit remove deletes one standalone secret, keeps the others", async ({ page, api, db }) => {
    const bundleName = unique("loopal-remove");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "loopal",
      name: bundleName,
      kind: "credential",
      data: { ANTHROPIC_API_KEY: "sk-ant-desktop-keep", OPENAI_API_KEY: "sk-openai-desktop-remove" },
    });

    try {
      await gotoAgentSettings(page, "loopal");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      // Remove only OpenAI over the IPC→node-bridge path.
      await page.getByTestId("remove-secret-OPENAI_API_KEY").click();
      await expect(page.getByTestId("restore-secret-OPENAI_API_KEY")).toBeVisible();
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-name")).toHaveCount(0);

      // Finding #1: the deleted standalone secret is dropped; the untouched one kept.
      const openaiGone = db.queryValue(
        `SELECT count(*) FROM env_bundles WHERE name = '${bundleName}' AND data ? 'OPENAI_API_KEY'`
      );
      expect(openaiGone).toBe("0");
      const anthropicKept = db.queryValue(
        `SELECT count(*) FROM env_bundles WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_API_KEY'`
      );
      expect(anthropicKept).toBe("1");
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("dialog renders three distinct provider keys + add-variable button", async ({ page }) => {
    await gotoAgentSettings(page, "loopal");

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();

    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-OPENAI_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-GOOGLE_API_KEY")).toBeVisible();
    await expect(page.getByRole("button", { name: /Add Variable|添加环境变量/ })).toBeVisible();
  });
});

test.describe("Desktop · Codex (custom env)", () => {
  test("removing a custom env row deletes that key, keeps the declared secret", async ({ page, api, db }) => {
    const bundleName = unique("codex-custom-remove");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "codex-cli",
      name: bundleName,
      kind: "credential",
      data: { OPENAI_API_KEY: "sk-openai-desktop-keep", HTTP_PROXY: "http://old-proxy" },
    });

    try {
      await gotoAgentSettings(page, "codex-cli");
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await expect(page.getByLabel(/ENV_NAME|环境变量名/).first()).toHaveValue("HTTP_PROXY");
      // The custom row's trash title is the bare "Remove" (^Remove$ excludes the
      // declared secret's "Remove this credential").
      await page.getByTitle(/^(Remove|移除)$/).first().click();
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-name")).toHaveCount(0);

      // Finding #1 (custom-env path): removed row → empty-string delete signal.
      const proxyGone = db.queryValue(
        `SELECT count(*) FROM env_bundles WHERE name = '${bundleName}' AND data ? 'HTTP_PROXY'`
      );
      expect(proxyGone).toBe("0");
      const keyKept = db.queryValue(
        `SELECT count(*) FROM env_bundles WHERE name = '${bundleName}' AND data ? 'OPENAI_API_KEY'`
      );
      expect(keyKept).toBe("1");
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("declared OPENAI_API_KEY + add-variable button", async ({ page }) => {
    await gotoAgentSettings(page, "codex-cli");

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();
    await expect(page.locator("#cred-OPENAI_API_KEY")).toBeVisible();
    await expect(page.getByRole("button", { name: /Add Variable|添加环境变量/ })).toBeVisible();
  });
});

import { test, expect } from "../../fixtures/index";
import { SettingsNavPage } from "../../pages/settings/settings-nav.page";
import { TEST_ORG_SLUG, TEST_USER } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// Regression coverage for the credential dialog after the EnvBundle refactor:
//
//   - "Credential" is now one kind of EnvBundle (kind='credential'); requests
//     reach EnvBundleService over Connect-RPC with a unified {kind, data} payload.
//   - Backend stays a pure KV store; the per-agent form spec lives on the
//     frontend (declared ENVs + custom ENV section).
//
// Three flows are covered:
//   * Claude Code — Base URL on top, RadioGroup XOR for API Key vs Auth
//     Token (only one input visible at a time).
//   * Loopal — three distinct provider labels + a custom-ENV section
//     that round-trips (e.g. XAI_API_KEY).
//   * Codex CLI — single declared field + custom-ENV section so users can
//     add OPENAI_BASE_URL etc. for proxy setups.

const NAME_PREFIX = "E2E Bundle";

function unique(label: string): string {
  return `${NAME_PREFIX} ${label} ${Date.now()}`;
}

test.describe("Personal Agent Credentials — Claude Code (XOR auth)", () => {
  test.beforeEach(async () => {
    clearAuthRateLimit();
  });

  test("toggling auth method without typing keeps the stored secret", async ({ page, api, db }) => {
    const bundleName = unique("half-switch");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      kind: "credential",
      data: { ANTHROPIC_AUTH_TOKEN: "sk-ant-keep-on-toggle" },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      // Stored auth is the token → its radio is preselected. Toggle to API Key
      // but type nothing, then save (as if the user only meant to rename).
      await page.getByTestId("oneof-option-anthropic_auth-ANTHROPIC_API_KEY").click();
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-name")).toHaveCount(0);

      // Finding #1: a half-finished switch (radio toggled, no value typed) must
      // NOT delete the stored token and leave the bundle with no credential.
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
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", `agents/claude-code`);
    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toBeVisible();
    // Finding #2: the base URL round-trips in plaintext, so the form warns the
    // user against putting a token in it.
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
      data: { ANTHROPIC_AUTH_TOKEN: "sk-ant-corrupt-me" },
    });
    // Corrupt the stored ciphertext directly (simulating a key rotation): the
    // value is no longer valid ciphertext, so any decrypt of it fails.
    db.exec(
      `UPDATE env_bundles SET data = '{"ANTHROPIC_AUTH_TOKEN":"garbage-not-ciphertext"}' WHERE name = '${bundleName}'`
    );

    const renamed = `${bundleName} renamed`;
    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await page.locator("#cred-name").fill(renamed);
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();

      // Finding #3: the update used to 500 because building its response
      // re-decrypted the historical corrupt secret. It now degrades — the write
      // commits, the dialog closes, and the new name shows.
      await expect(page.locator("#cred-name")).toHaveCount(0);
      await expect(page.getByText(renamed, { exact: true })).toBeVisible({ timeout: 15_000 });
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("dialog shows Base URL first + RadioGroup that toggles inputs", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", `agents/claude-code`);

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();

    await expect(page.locator("#cred-name")).toBeVisible();
    await expect(page.locator("#cred-desc")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toHaveAttribute("type", "text");

    // RadioGroup defaults to API Key — Auth Token input must not exist yet.
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveAttribute("type", "password");
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toHaveCount(0);

    // Switching the radio swaps the active input.
    await page.getByTestId("oneof-option-anthropic_auth-ANTHROPIC_AUTH_TOKEN").click();
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
    await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toHaveAttribute("type", "password");
    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveCount(0);
  });

  test("seeded credential bundle renders in the list", async ({ page, api, db }) => {
    const bundleName = unique("seeded");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "claude-code",
      name: bundleName,
      description: "seeded by e2e",
      kind: "credential",
      data: { ANTHROPIC_API_KEY: "sk-ant-e2e-seeded" },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });

  test("UI create flow: new credential appears in the list after submit", async ({ page, db }) => {
    const bundleName = unique("ui-create");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);

      await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
        .first().click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();

      await page.locator("#cred-name").fill(bundleName);
      await page.locator("#cred-desc").fill("created via UI");
      await page.locator("#cred-ANTHROPIC_BASE_URL").fill("https://proxy.anthropic.example");
      await page.locator("#cred-ANTHROPIC_API_KEY").fill("sk-ant-e2e-ui-created");

      await page.getByRole("button", { name: /^(Create|创建)$/ }).click();

      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      // Create drops the deselected XOR sibling: only the API Key is stored, no
      // blank ANTHROPIC_AUTH_TOKEN ghost key (the form submits it as "").
      const tokenAbsent = db.queryValue(
        `SELECT count(*) FROM env_bundles
         WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_AUTH_TOKEN'`
      );
      expect(tokenAbsent).toBe("0");
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
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
        ANTHROPIC_AUTH_TOKEN: "sk-ant-e2e-token",
      },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();

      // Read fix: the non-secret Base URL prefills its stored value. Before the
      // per-key split the backend hid every credential value, so this was blank.
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL"))
        .toHaveValue("https://proxy.anthropic.example");

      // The stored secret was the Auth Token: its radio is preselected, the
      // input stays blank with the "keep existing" placeholder, and the API
      // Key field is not rendered (XOR).
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
        ANTHROPIC_AUTH_TOKEN: "sk-ant-keep-me",
      },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      // First edit: change ONLY the Base URL, leave the Auth Token blank.
      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL"))
        .toHaveValue("https://old.anthropic.example");
      await page.locator("#cred-ANTHROPIC_BASE_URL").fill("https://new.anthropic.example");
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();

      // handleSaveProfile awaits update + list reload before closing the
      // dialog, so the input disappearing means the write already landed.
      await expect(page.locator("#cred-ANTHROPIC_BASE_URL")).toHaveCount(0);

      // Write fix: a non-secret-only update must NOT wipe the untouched secret.
      const tokenKept = db.queryValue(
        `SELECT count(*) FROM env_bundles
         WHERE name = '${bundleName}' AND data ? 'ANTHROPIC_AUTH_TOKEN'`
      );
      expect(tokenKept).toBe("1");

      // Re-edit (二次编辑): the new Base URL shows and the Auth Token radio is
      // still preselected — proof configured_fields still lists the token.
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
      data: { ANTHROPIC_AUTH_TOKEN: "sk-ant-old-token" },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/claude-code`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      // Stored auth was the token → its radio is preselected.
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toBeVisible();
      // Switch to API Key and fill it.
      await page.getByTestId("oneof-option-anthropic_auth-ANTHROPIC_API_KEY").click();
      await page.locator("#cred-ANTHROPIC_API_KEY").fill("sk-ant-new-key");
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toHaveCount(0);

      // The old Auth Token must be gone — switching auth must not leave both
      // set, else the pod gets two conflicting Anthropic auth env vars.
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

      // Re-edit: API Key preselected, the token is gone.
      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();
      await expect(page.locator("#cred-ANTHROPIC_AUTH_TOKEN")).toHaveCount(0);
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });
});

test.describe("Personal Agent Credentials — Loopal (custom env)", () => {
  test.beforeEach(async () => {
    clearAuthRateLimit();
  });

  test("explicit remove deletes one standalone secret, keeps the others", async ({ page, api, db }) => {
    const bundleName = unique("loopal-remove");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "loopal",
      name: bundleName,
      kind: "credential",
      data: { ANTHROPIC_API_KEY: "sk-ant-keep", OPENAI_API_KEY: "sk-openai-remove" },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/loopal`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      // Both configured secrets expose a remove button; remove only OpenAI.
      await page.getByTestId("remove-secret-OPENAI_API_KEY").click();
      await expect(page.getByTestId("restore-secret-OPENAI_API_KEY")).toBeVisible();
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-name")).toHaveCount(0);

      // Finding #1: a standalone secret could not be deleted before — the merge
      // re-preserved any blank-but-omitted secret. The explicit remove now sends
      // an empty-string delete signal; the untouched sibling is preserved.
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

  test("dialog renders three distinct provider keys + custom env section", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", `agents/loopal`);

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();

    await expect(page.locator("#cred-ANTHROPIC_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-OPENAI_API_KEY")).toBeVisible();
    await expect(page.locator("#cred-GOOGLE_API_KEY")).toBeVisible();
    await expect(page.getByRole("button", { name: /Add Variable|添加环境变量/ })).toBeVisible();
  });

  test("custom env round-trips through the backend", async ({ page, db }) => {
    const bundleName = unique("loopal-custom");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/loopal`);

      await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
        .first().click();
      await page.locator("#cred-name").fill(bundleName);
      await page.locator("#cred-OPENAI_API_KEY").fill("sk-openai-loopal");
      await page.getByRole("button", { name: /Add Variable|添加环境变量/ }).click();

      const keyInputs = page.getByLabel(/ENV_NAME|环境变量名/);
      await keyInputs.last().fill("XAI_API_KEY");
      const valueInputs = page.getByLabel(/^Value$|^值$/);
      await valueInputs.last().fill("xai-secret-value");

      await page.getByRole("button", { name: /^(Create|创建)$/ }).click();
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      // The bundle should have landed in env_bundles with kind=credential and
      // agent_slug=loopal. We only verify presence; the encrypted blob lives
      // in `data` JSONB which we don't probe from e2e.
      const count = db.queryValue(
        `SELECT count(*) FROM env_bundles
         WHERE name = '${bundleName}' AND agent_slug = 'loopal' AND kind = 'credential'`
      );
      expect(count).toBe("1");
    } finally {
      db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);
    }
  });
});

test.describe("Personal Agent Credentials — Codex (custom env)", () => {
  test.beforeEach(async () => {
    clearAuthRateLimit();
  });

  test("removing a custom env row deletes that key, keeps the declared secret", async ({ page, api, db }) => {
    const bundleName = unique("codex-custom-remove");
    db.cleanup(`DELETE FROM env_bundles WHERE name LIKE '${NAME_PREFIX}%'`);

    await api.login(TEST_USER.email, TEST_USER.password);
    const cc = await api.connect();
    await cc.envBundle.createEnvBundle({
      agentSlug: "codex-cli",
      name: bundleName,
      kind: "credential",
      data: { OPENAI_API_KEY: "sk-openai-keep", HTTP_PROXY: "http://old-proxy" },
    });

    try {
      const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
      await nav.goto("personal", `agents/codex-cli`);
      await expect(page.getByText(bundleName, { exact: true })).toBeVisible({ timeout: 15_000 });

      await page.getByTitle(/^(Edit|编辑)$/).first().click();
      // The custom HTTP_PROXY row is rebuilt from the stored keys; its key input
      // shows even though the (secret) value does not round-trip.
      await expect(page.getByLabel(/ENV_NAME|环境变量名/).first()).toHaveValue("HTTP_PROXY");
      // Remove the custom row. Its trash title is the bare "Remove"; the declared
      // secret's remove button reads "Remove this credential", so ^Remove$ is exact.
      await page.getByTitle(/^(Remove|移除)$/).first().click();
      await page.getByRole("button", { name: /^(Save|保存)$/ }).click();
      await expect(page.locator("#cred-name")).toHaveCount(0);

      // Finding #1 (custom-env path): the removed row becomes an empty-string
      // delete signal; the declared secret left blank is preserved.
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

  test("declared OPENAI_API_KEY + custom env button", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", `agents/codex-cli`);

    await page.getByRole("button", { name: /Add Custom Credentials|添加自定义凭据/i })
      .first().click();
    await expect(page.locator("#cred-OPENAI_API_KEY")).toBeVisible();
    await expect(page.getByRole("button", { name: /Add Variable|添加环境变量/ })).toBeVisible();
  });
});

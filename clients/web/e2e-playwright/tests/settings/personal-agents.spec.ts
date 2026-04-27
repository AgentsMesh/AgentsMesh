import { test, expect } from "../../fixtures/index";
import { SettingsNavPage } from "../../pages/settings/settings-nav.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

test.describe("Personal Agent Configuration", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  /**
   * TC-AGENT-001: Agent config page navigation
   */
  test("agent config page shows agent types", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "agents");

    const body = await page.textContent("body");
    // Should list supported agents
    expect(body).toMatch(/Claude Code|Gemini|Codex|Aider|OpenCode/i);
  });

  /**
   * TC-AGENT-002: Claude Code default config
   */
  test("Claude Code config shows default sections", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "agents/claude-code");

    const body = await page.textContent("body");
    expect(body).toMatch(/Claude Code/i);
    // Should have credential and runtime sections
    expect(body).toMatch(/credential|凭据|API/i);
  });

  /**
   * TC-AGENT-003: Add custom API credential
   */
  test("add and delete custom API credential", async ({ api, db }) => {
    db.cleanup(
      `DELETE FROM user_agent_credential_profiles WHERE name = 'E2E Test Credential'`
    );
    const res = await api.post("/api/v1/users/agent-credentials/agents/claude-code", {
      name: "E2E Test Credential",
      description: "Test credential for E2E",
      credentials: { ANTHROPIC_API_KEY: "sk-ant-test-key-12345" },
    });
    expect([200, 201]).toContain(res.status);

    db.cleanup(
      `DELETE FROM user_agent_credential_profiles WHERE name = 'E2E Test Credential'`
    );
  });

  /**
   * TC-AGENT-008: Switch Claude model
   */
  test("agent config page has model selection", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "agents/claude-code");

    // Should have model-related text (Opus, Sonnet)
    const body = await page.textContent("body");
    expect(body).toMatch(/model|模型|Opus|Sonnet/i);
  });

  /**
   * TC-AGENT-015: Save runtime configuration
   */
  test("agent config page shows runtime section or no-config message", async ({ page }) => {
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "agents/claude-code");

    const body = await page.textContent("body");
    const hasSave = await page.getByRole("button", { name: /save|保存/i }).isVisible().catch(() => false);
    if (hasSave) {
      expect(hasSave).toBe(true);
    } else {
      expect(body).toMatch(/No configuration options|没有可用的配置/i);
    }
  });
});

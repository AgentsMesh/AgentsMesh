import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import { createMockAgentPod, loopalConsoleUrlForPod } from "../../helpers/mock-agent";

// Browser end-to-end coverage of the redesigned Loopal control console:
//   browser ↔ relay ↔ runner ↔ e2e-mock-agent (--scenario=loopal_panels)
// The mock emits the full _loopal/* signal set (incl. mode/thinking/model) on
// prompt; the console folds them into the top status bar, the bottom data dock,
// and the React Flow topology.
test.describe("Loopal console (ACP mode)", () => {
  test.beforeEach(() => {
    clearAuthRateLimit();
  });
  test.afterEach(async () => {
    await terminateAllPods();
  });

  test("renders status-bar truth, data dock, and topology from mock signals", async ({ page, api, monitor }) => {
    const pod = await createMockAgentPod(api, { mode: "acp", scenario: "loopal_panels", prompt: "go" });
    await page.goto(loopalConsoleUrlForPod(pod.podKey));
    await page.waitForLoadState("load");

    // Goal indicator confirms the _loopal/* pipeline reaches the top bar.
    await expect(page.getByTestId("loopal-goal-indicator")).toContainText("ship it", { timeout: 15_000 });
    // Status bar shows real mode/thinking/model (new _loopal/mode|thinking|model).
    await expect(page.getByTestId("loopal-mode-badge")).toHaveText("Plan");
    await expect(page.getByTestId("loopal-thinking")).toContainText("High");
    await expect(page.getByTestId("loopal-model")).toContainText("claude-opus-4-7");

    // Data dock: tabs appear only for non-empty classes; expand each.
    await page.getByTestId("loopal-dock-tab-bg").click();
    await expect(page.getByText("npm test")).toBeVisible();
    await page.getByTestId("loopal-dock-tab-cron").click();
    await expect(page.getByText("0 9 * * *")).toBeVisible();
    await page.getByTestId("loopal-dock-tab-tasks").click();
    await expect(page.getByText("build")).toBeVisible();
    await page.getByTestId("loopal-dock-tab-mcp").click();
    await expect(page.getByText("fs")).toBeVisible();
    // Topology renders the spawned agent as a React Flow node.
    await page.getByTestId("loopal-dock-tab-agents").click();
    await expect(
      page.locator('[data-testid="loopal-agent-node"][data-agent-name="worker"]'),
    ).toBeVisible();

    expect(monitor.errors()).toHaveLength(0);
  });

  test("slash command parses to a control_request, not a chat prompt", async ({ page, api, monitor }) => {
    const pod = await createMockAgentPod(api, { mode: "acp", scenario: "loopal_panels", prompt: "go" });
    await page.goto(loopalConsoleUrlForPod(pod.podKey));
    await page.waitForLoadState("load");
    await expect(page.getByTestId("loopal-mode-badge")).toBeVisible({ timeout: 15_000 });

    const input = page.getByTestId("loopal-prompt-input");
    // Typing "/" opens the slash menu — proves the composer recognizes commands.
    await input.fill("/");
    await expect(page.getByTestId("loopal-slash-dropdown")).toBeVisible();

    // Submitting "/compact" runs the command: execute() clears the input and
    // closes the menu, so it was dispatched as a loopal.* control_request rather
    // than sent verbatim to the agent as a chat prompt (which would not clear
    // the same way and would echo into the activity stream).
    await input.fill("/compact");
    await input.press("Enter");
    await expect(input).toHaveValue("");
    await expect(page.getByTestId("loopal-slash-dropdown")).toBeHidden();

    expect(monitor.errors()).toHaveLength(0);
  });
});

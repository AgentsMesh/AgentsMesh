import { expect } from "../fixtures";
import type { ElectronApplication, Locator, Page } from "@playwright/test";
import type { ApiFixture } from "../../../web/e2e-playwright/fixtures/api.fixture";
import { TEST_ORG_SLUG } from "./env";
import { invokeIpc } from "./ipc";
import { gotoHash } from "./nav";
import {
  TERMINAL_RENDER_COMMAND,
  TERMINAL_RENDER_DONE,
  TERMINAL_RENDER_READY,
  TERMINAL_ALT_BUFFER_PROBE,
  TERMINAL_DCS_PAYLOAD,
  TERMINAL_OSC_PAYLOAD,
  TERMINAL_SELECTION_ROW,
  TERMINAL_SELECTION_TEXT,
  TERMINAL_UNICODE_GLYPHS,
  TERMINAL_UNICODE_ROW,
  TERMINAL_UNICODE_SELECTION_TARGETS,
  type TerminalRenderMetrics,
  readTerminalRenderMetrics,
  readTerminalRenderRows,
  readTerminalUnicodeMetrics,
  readTerminalUnicodeSelectionMetrics,
  selectTerminalMarker,
  waitForTerminalRender,
} from "../../../web/e2e-playwright/helpers/terminal-ui";

export async function createDesktopPtyPod(page: Page, agentfileLayer?: string): Promise<string> {
  // Each spec launches an isolated Electron profile cloned from global setup.
  // Re-bootstrap the main-process Rust auth state immediately before the first
  // privileged IPC call; renderer hydration can otherwise win the cold-start
  // race and issue ListRunners before the cloned bearer is loaded.
  await invokeIpc(page, "authBootstrap");
  const runners = await invokeIpc<string>(page, "runnerFetchRunners");
  const runnerList = JSON.parse(runners) as { runners?: { id: number; status: string }[] } | { id: number; status: string }[];
  const onlineRunner = (Array.isArray(runnerList) ? runnerList : runnerList.runners ?? [])
    .find((runner) => runner.status === "online");
  expect(onlineRunner, "dev env must have an online runner").toBeTruthy();

  const created = await invokeIpc<string>(page, "podCreatePod", JSON.stringify({
    agent_slug: "e2e-echo",
    runner_id: onlineRunner!.id,
    agentfile_layer: agentfileLayer,
    cols: 120,
    rows: 32,
  }));
  const podKey = (JSON.parse(created) as { pod: { pod_key: string } }).pod.pod_key;
  expect(podKey, "podCreatePod returned a pod_key").toBeTruthy();
  return podKey;
}
export async function waitForPodRelayReady(api: ApiFixture, podKey: string): Promise<void> {
  const cc = await api.connect();
  await expect.poll(async () => {
    const pod = await cc.pod.getPod({
      orgSlug: TEST_ORG_SLUG,
      podKey,
    }) as { status?: string };
    return pod.status;
  }, {
    message: "workspace hydration must observe the terminal-ready pod state",
    timeout: 30_000,
  }).toBe("running");

  // GetPodConnection sends the runner's subscribe command as a side effect.
  // Invoke it only after the status barrier instead of once per poll attempt.
  const connection = await cc.pod.getPodConnection({
    orgSlug: TEST_ORG_SLUG,
    podKey,
  }) as { relayUrl?: string };
  expect(connection.relayUrl, "pod must publish its relay endpoint before workspace hydration")
    .toBeTruthy();
}

export async function openDesktopPtyPod(page: Page, podKey: string, viaDeepLink = false): Promise<Locator> {
  await gotoHash(page, `/${TEST_ORG_SLUG}/workspace`);
  await page.reload();
  await page.waitForLoadState("domcontentloaded");
  await invokeIpc(page, "authBootstrap");
  await gotoHash(
    page,
    viaDeepLink
      ? `/${TEST_ORG_SLUG}/workspace?pod=${encodeURIComponent(podKey)}`
      : `/${TEST_ORG_SLUG}/workspace`,
  );

  if (!viaDeepLink) {
    const sidebarPod = page.locator(
      `[data-testid="pod-list-item"][data-pod-key="${podKey}"]`,
    );
    await expect(sidebarPod, "new pod must appear in sidebar").toBeVisible({ timeout: 30_000 });
    await sidebarPod.click();
  }

  const terminal = page.locator(".xterm");
  await expect(terminal).toBeVisible({ timeout: 30_000 });
  return terminal;
}

export async function startDesktopTerminalRenderFixture(
  page: Page,
  api: ApiFixture,
): Promise<{ podKey: string; terminal: Locator }> {
  const podKey = await createDesktopPtyPod(
    page,
    'CONFIG scenario = "terminal_render"\n',
  );
  await waitForPodRelayReady(api, podKey);

  const terminal = await openDesktopPtyPod(page, podKey);
  await expect(terminal, "render fixture must wait until Electron is subscribed").toContainText(
    TERMINAL_RENDER_READY,
    { timeout: 45_000 },
  );

  const input = terminal.locator(".xterm-helper-textarea");
  await input.focus();
  await page.keyboard.type(TERMINAL_RENDER_COMMAND);
  await page.keyboard.press("Enter");

  await expect(terminal, "alternate-buffer probe must render before returning to normal mode").toContainText(
    TERMINAL_ALT_BUFFER_PROBE,
    { timeout: 15_000 },
  );
  await expect(terminal).toContainText(TERMINAL_RENDER_DONE, { timeout: 30_000 });
  return { podKey, terminal };
}

export async function resizePrimaryDesktopWindow(
  electronApp: ElectronApplication,
): Promise<{ width: number; height: number }> {
  return electronApp.evaluate(({ BrowserWindow }) => {
    const win = BrowserWindow.getAllWindows().find((candidate) =>
      !candidate.isDestroyed() && !candidate.webContents.getURL().includes("/popout/"),
    );
    if (!win) throw new Error("no primary BrowserWindow");

    const before = win.getContentBounds();
    const width = before.width >= 1100 ? before.width - 180 : before.width + 180;
    const height = before.height >= 720 ? before.height - 100 : before.height + 100;
    win.setContentSize(width, height);
    return { width, height };
  });
}

export async function readRendererViewport(page: Page): Promise<{ width: number; height: number }> {
  return page.evaluate(() => ({ width: window.innerWidth, height: window.innerHeight }));
}

export async function expectRenderSurface(terminal: Locator): Promise<TerminalRenderMetrics> {
  await waitForTerminalRender(terminal);
  const rows = await readTerminalRenderRows(terminal);
  expect(rows).toContain(TERMINAL_SELECTION_ROW);
  expect(rows).toContain(TERMINAL_UNICODE_ROW);
  expect(rows).toContain("CURSOR:OKxxx");
  expect(rows).toContain(TERMINAL_RENDER_DONE);
  expect(rows.join("\n")).not.toContain(TERMINAL_DCS_PAYLOAD);
  expect(rows.join("\n")).not.toContain(TERMINAL_OSC_PAYLOAD);

  const metrics = await readTerminalRenderMetrics(terminal);
  expect(metrics.cellWidth).toBeGreaterThan(4);
  expect(metrics.rowHeight).toBeGreaterThan(8);
  expect(metrics.wideColumns).toBeCloseTo(2, 1);
  expect(metrics.targetStartColumn).toBeCloseTo(10, 1);
  expect(metrics.targetWidthColumns).toBeCloseTo(TERMINAL_SELECTION_TEXT.length, 1);
  const unicodeMetrics = await readTerminalUnicodeMetrics(terminal, metrics.cellWidth);
  for (const [index, actual] of unicodeMetrics.entries()) {
    const expected = TERMINAL_UNICODE_GLYPHS[index];
    expect(actual.text).toBe(expected.text);
    expect(actual.visualColumns, `${expected.name} must occupy the expected visual width`)
      .toBeCloseTo(expected.visualColumns, 1);
    expect(actual.visualStartColumn, `${expected.name} must start at the expected visual column`)
      .toBeCloseTo(expected.visualStartColumn, 1);
  }
  return metrics;
}

export async function expectMarkerSelection(page: Page, terminal: Locator): Promise<void> {
  const metrics = await expectRenderSurface(terminal);
  const selection = await selectTerminalMarker(page, terminal, metrics);
  const tolerance = metrics.cellWidth * 0.4;
  expect(selection.handled, "xterm copy handler must consume the copy event").toBe(true);
  expect(selection.text, "mouse drag must select exactly the intended terminal cells").toBe(TERMINAL_SELECTION_TEXT);
  expect(Math.abs(selection.overlayLeft - metrics.targetLeft)).toBeLessThan(tolerance);
  expect(Math.abs(selection.overlayWidth - metrics.targetWidth)).toBeLessThan(tolerance);

  const unicodeTargets = await readTerminalUnicodeSelectionMetrics(terminal, metrics.cellWidth);
  for (const [index, target] of unicodeTargets.entries()) {
    const expected = TERMINAL_UNICODE_SELECTION_TARGETS[index];
    expect(target.text).toBe(expected.text);
    expect(target.bufferStartColumn).toBe(expected.bufferStartColumn);
    expect(target.visualStartColumn, `${expected.name} visual position must match the buffer cell`)
      .toBeCloseTo(expected.visualStartColumn, 1);
    expect(target.visualWidthColumns).toBeCloseTo(expected.text.length, 1);
    expect(
      (target.bufferLeft - target.targetLeft) / metrics.cellWidth,
      `${expected.name} buffer and shaped DOM coordinates must share one contract`,
    ).toBeCloseTo(0, 1);

    const unicodeSelection = await selectTerminalMarker(page, terminal, target);
    expect(unicodeSelection.handled, `${expected.name} copy must use xterm's production handler`).toBe(true);
    expect(unicodeSelection.text, `${expected.name} must copy exactly when dragged over its visible DOM bounds`)
      .toBe(expected.text);
    expect(
      Math.abs(unicodeSelection.overlayLeft - target.targetLeft),
      `${expected.name} selection overlay must align with the visible DOM target`,
    ).toBeLessThan(tolerance);
    expect(
      Math.abs(unicodeSelection.overlayWidth - target.targetWidth),
      `${expected.name} selection width must align with the visible DOM target`,
    ).toBeLessThan(tolerance);
  }
}

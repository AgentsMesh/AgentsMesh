import { test, expect } from "../../fixtures";
import type { ElectronApplication, Locator, Page } from "@playwright/test";
import { invokeIpc } from "../../helpers/ipc";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";
import type { ApiFixture } from "../../../../web/e2e-playwright/fixtures/api.fixture";
import {
  TERMINAL_RENDER_COMMAND,
  TERMINAL_RENDER_DONE,
  TERMINAL_RENDER_READY,
  TERMINAL_ALT_BUFFER_PROBE,
  TERMINAL_ALT_SNAPSHOT_ACTIVE,
  TERMINAL_ALT_SNAPSHOT_ENTER_COMMAND,
  TERMINAL_ALT_SNAPSHOT_EXIT_COMMAND,
  TERMINAL_ALT_SNAPSHOT_EXITED,
  TERMINAL_ALT_SNAPSHOT_NORMAL_HISTORY_TOP,
  TERMINAL_ALT_SNAPSHOT_READY,
  TERMINAL_ALT_SNAPSHOT_SURFACE,
  TERMINAL_DCS_PAYLOAD,
  TERMINAL_NORMAL_BUFFER_SENTINEL,
  TERMINAL_OSC_PAYLOAD,
  TERMINAL_SELECTION_ROW,
  TERMINAL_SELECTION_TEXT,
  TERMINAL_UNICODE_GLYPHS,
  TERMINAL_UNICODE_ROW,
  TERMINAL_UNICODE_SELECTION_TARGETS,
  type TerminalRenderMetrics,
  readLatestTerminalPtySize,
  readTerminalRenderMetrics,
  readTerminalRenderRows,
  readTerminalUnicodeMetrics,
  readTerminalUnicodeSelectionMetrics,
  scrollTerminalToBottom,
  scrollTerminalToTop,
  selectTerminalMarker,
  waitForTerminalRender,
} from "../../../../web/e2e-playwright/helpers/terminal-ui";

async function createDesktopPtyPod(page: Page, agentfileLayer?: string): Promise<string> {
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

async function waitForPodRelayReady(api: ApiFixture, podKey: string): Promise<void> {
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

async function openDesktopPtyPod(page: Page, podKey: string, viaDeepLink = false): Promise<Locator> {
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

async function startDesktopTerminalRenderFixture(
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

async function resizePrimaryDesktopWindow(
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

async function readRendererViewport(page: Page): Promise<{ width: number; height: number }> {
  return page.evaluate(() => ({ width: window.innerWidth, height: window.innerHeight }));
}

async function expectRenderSurface(terminal: Locator): Promise<TerminalRenderMetrics> {
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

async function expectMarkerSelection(page: Page, terminal: Locator): Promise<void> {
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

/**
 * Desktop terminal DATA PLANE round-trip after the relay-SSOT migration.
 *
 * Desktop's relay path differs from web: the Rust RelayConnectionPool runs in
 * the MAIN process (node-bridge), PTY bytes retain their subscription identity
 * through the `relay:*` IPC bridge, and ElectronRelayManager dispatches each
 * stream to its addressed xterm subscriber. This proves that whole chain:
 *   xterm ↔ relayConnection adapter ↔ ElectronRelayManager ↔ IPC ↔ main Rust
 *   pool ↔ relay WS ↔ runner PTY ↔ e2e-echo (pty mode).
 *
 * pty_runtime.go writes "ready" on spawn, then echoes each stdin line as
 * "got: <line>". The e2e-echo agent (migration 000151) defaults to pty mode.
 */
test.describe("Desktop terminal round-trip (relay SSOT)", () => {
  test("attaches, streams pty output, and round-trips typed input via the main-process pool", async ({ page, api }) => {
    // pty is the e2e-echo default mode — no agentfile layer needed.
    const podKey = await createDesktopPtyPod(page);

    // The workspace's first status fetch must observe the relay-ready pod;
    // usePodStatus reuses that cached value and intentionally does not subscribe
    // a terminal for a stale "initializing" snapshot.
    await waitForPodRelayReady(api, podKey);

    try {
      // OUTPUT: the terminal pane self-fetches pod status (usePodStatus) and
      // subscribes once running; the daemon replays the buffered "ready" on
      // attach. Generous window covers fetch + realtime status flip + subscribe.
      const term = await openDesktopPtyPod(page, podKey);
      await expect(term, "pty 'ready' must stream through the main-process pool to xterm").toContainText(
        "ready",
        { timeout: 45_000 },
      );

      // INPUT: typed line → ElectronRelayManager → IPC → main pool → relay → PTY.
      await page.locator(".xterm-helper-textarea").focus();
      await page.keyboard.type("relay-roundtrip");
      await page.keyboard.press("Enter");

      await expect(term, "pty echo must round-trip back through the bridge to xterm").toContainText(
        "got: relay-roundtrip",
        { timeout: 20_000 },
      );
    } finally {
      await invokeIpc<void>(page, "podTerminatePod", podKey).catch(() => undefined);
    }
  });

  test("rebinds a fresh N-API driver generation after raw driver disconnect", async ({
    page,
    api,
  }, testInfo) => {
    const podKey = await createDesktopPtyPod(page);
    await waitForPodRelayReady(api, podKey);

    try {
      const liveTerminal = await openDesktopPtyPod(page, podKey);
      await expect(liveTerminal, "initial driver must deliver the PTY baseline")
        .toContainText("ready", { timeout: 45_000 });
      await expect.poll(
        () => invokeIpc<string>(page, "relayGetStatus", podKey),
        { message: "initial Rust relay driver must be connected", timeout: 15_000 },
      ).toBe("connected");

      const liveInput = liveTerminal.locator(".xterm-helper-textarea");
      await liveInput.focus();
      await page.keyboard.type("driver-generation-before");
      await page.keyboard.press("Enter");
      await expect(liveTerminal).toContainText("got: driver-generation-before", {
        timeout: 20_000,
      });

      // Deliberately invoke the raw AppState N-API method. Unlike the renderer
      // manager's `relay:disconnect` channel, this does not first remove the
      // ElectronRelayManager output route or main's subscription identity.
      // The old Rust driver must retire cleanly while those owners still exist.
      await invokeIpc<void>(page, "relayDisconnect", podKey);
      await expect.poll(
        () => invokeIpc<string>(page, "relayGetStatus", podKey),
        { message: "raw N-API disconnect must retire the old driver", timeout: 15_000 },
      ).toBe("disconnected");
      await expect(liveTerminal, "raw driver teardown must not destroy the renderer terminal")
        .toBeVisible();

      // Reload creates a new renderer subscription while main still has to
      // reconcile the retired generation. A new Rust generation must publish
      // its snapshot baseline and accept fresh input/output afterwards.
      const reboundTerminal = await openDesktopPtyPod(page, podKey, true);
      await expect(reboundTerminal, "replacement generation must replay prior terminal output")
        .toContainText("got: driver-generation-before", { timeout: 45_000 });
      await expect.poll(
        () => invokeIpc<string>(page, "relayGetStatus", podKey),
        { message: "replacement Rust relay driver must become connected", timeout: 20_000 },
      ).toBe("connected");

      const reboundInput = reboundTerminal.locator(".xterm-helper-textarea");
      await reboundInput.focus();
      await page.keyboard.type("driver-generation-after");
      await page.keyboard.press("Enter");
      await expect(reboundTerminal, "replacement generation must deliver fresh PTY output")
        .toContainText("got: driver-generation-after", { timeout: 20_000 });

      await testInfo.attach("desktop-terminal-driver-rebound", {
        body: await reboundTerminal.screenshot(),
        contentType: "image/png",
      });
    } finally {
      await invokeIpc<void>(page, "podTerminatePod", podKey).catch(() => undefined);
    }
  });

  test("preserves fragmented Unicode layout and mouse selection across snapshot replay", async ({ page, api }, testInfo) => {
    const { podKey, terminal: liveTerminal } = await startDesktopTerminalRenderFixture(page, api);

    try {
      await expectRenderSurface(liveTerminal);
      await scrollTerminalToTop(page, liveTerminal);
      await expect(liveTerminal).toContainText("SCROLL-00|abcdefghijklmnopqrstuvwxyz|");
      await expect(liveTerminal, "normal-buffer contents must survive alternate-buffer exit")
        .toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);
      await expect(liveTerminal, "alternate-buffer contents must not leak into normal scrollback")
        .not.toContainText(TERMINAL_ALT_BUFFER_PROBE);
      await expect(liveTerminal, "OSC payload must remain non-printing")
        .not.toContainText(TERMINAL_OSC_PAYLOAD);
      await expect(liveTerminal, "DCS payload must remain non-printing")
        .not.toContainText(TERMINAL_DCS_PAYLOAD);
      await scrollTerminalToBottom(page, liveTerminal);
      await expectMarkerSelection(page, liveTerminal);

      // Reopening the workspace replaces the Electron listener and xterm
      // instance. The restored surface now comes from Runner snapshot replay,
      // not from the original live PTY writes.
      const restoredTerminal = await openDesktopPtyPod(page, podKey, true);
      await expect(restoredTerminal, "snapshot replay must restore the final terminal surface").toContainText(
        TERMINAL_RENDER_DONE,
        { timeout: 45_000 },
      );
      await scrollTerminalToTop(page, restoredTerminal);
      await expect(restoredTerminal, "snapshot replay must retain the oldest normal-buffer scrollback")
        .toContainText("SCROLL-00|abcdefghijklmnopqrstuvwxyz|");
      await expect(restoredTerminal, "snapshot replay must retain normal-buffer contents")
        .toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);
      await expect(restoredTerminal, "alternate-buffer output must not pollute restored normal scrollback")
        .not.toContainText(TERMINAL_ALT_BUFFER_PROBE);
      await scrollTerminalToBottom(page, restoredTerminal);
      await expectMarkerSelection(page, restoredTerminal);

      await testInfo.attach("terminal-render-restored", {
        body: await restoredTerminal.screenshot(),
        contentType: "image/png",
      });
    } finally {
      await invokeIpc<void>(page, "podTerminatePod", podKey).catch(() => undefined);
    }
  });

  test("replays an active alternate buffer and restores hidden normal history on exit", async ({
    page,
    api,
  }, testInfo) => {
    const podKey = await createDesktopPtyPod(
      page,
      'CONFIG scenario = "terminal_alt_snapshot"\n',
    );
    await waitForPodRelayReady(api, podKey);

    try {
      const liveTerminal = await openDesktopPtyPod(page, podKey);
      await expect(liveTerminal).toContainText(TERMINAL_ALT_SNAPSHOT_READY, {
        timeout: 45_000,
      });
      const liveInput = liveTerminal.locator(".xterm-helper-textarea");
      await liveInput.focus();
      await page.keyboard.type(TERMINAL_ALT_SNAPSHOT_ENTER_COMMAND);
      await page.keyboard.press("Enter");
      await expect(liveTerminal, "fixture must remain inside the alternate buffer")
        .toContainText(TERMINAL_ALT_SNAPSHOT_ACTIVE, { timeout: 30_000 });
      await expect(liveTerminal, "fixture must expose the active alternate surface")
        .toContainText(TERMINAL_ALT_SNAPSHOT_SURFACE);
      await expect(liveTerminal, "active alternate buffer must hide the normal surface")
        .not.toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);

      const restoredTerminal = await openDesktopPtyPod(page, podKey, true);
      await expect(restoredTerminal, "snapshot must restore the active alternate surface")
        .toContainText(TERMINAL_ALT_SNAPSHOT_ACTIVE, { timeout: 45_000 });
      await expect(restoredTerminal, "snapshot must replay alternate-buffer contents")
        .toContainText(TERMINAL_ALT_SNAPSHOT_SURFACE);
      await expect(restoredTerminal, "normal history must stay hidden while DECSET 1049 is active")
        .not.toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);

      const restoredInput = restoredTerminal.locator(".xterm-helper-textarea");
      await restoredInput.focus();
      await page.keyboard.type(TERMINAL_ALT_SNAPSHOT_EXIT_COMMAND);
      await page.keyboard.press("Enter");
      await expect(restoredTerminal, "DECRST 1049 must return to the hidden normal buffer")
        .toContainText(TERMINAL_ALT_SNAPSHOT_EXITED, { timeout: 30_000 });
      await expect(restoredTerminal, "alternate surface must disappear after DECRST 1049")
        .not.toContainText(TERMINAL_ALT_SNAPSHOT_ACTIVE);

      await scrollTerminalToTop(page, restoredTerminal);
      await expect(restoredTerminal, "hidden normal scrollback must survive replay")
        .toContainText(TERMINAL_ALT_SNAPSHOT_NORMAL_HISTORY_TOP);
      await expect(restoredTerminal, "hidden normal sentinel must survive replay")
        .toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);
      await expect(restoredTerminal, "alternate content must not leak into normal history")
        .not.toContainText(TERMINAL_ALT_SNAPSHOT_ACTIVE);
      await scrollTerminalToBottom(page, restoredTerminal, TERMINAL_ALT_SNAPSHOT_EXITED);

      await testInfo.attach("desktop-terminal-active-alt-restored-normal-history", {
        body: await restoredTerminal.screenshot(),
        contentType: "image/png",
      });
    } finally {
      await invokeIpc<void>(page, "podTerminatePod", podKey).catch(() => undefined);
    }
  });

  test("propagates PTY SIGWINCH dimensions through Electron resize and snapshot replay", async ({
    page,
    electronApp,
    api,
  }, testInfo) => {
    const { podKey, terminal: liveTerminal } = await startDesktopTerminalRenderFixture(page, api);

    try {
      await expectRenderSurface(liveTerminal);

      // Recreate both the renderer listener and xterm first, so the resize is
      // exercised after a Runner baseline instead of only after live writes.
      const restoredTerminal = await openDesktopPtyPod(page, podKey, true);
      await expect(restoredTerminal, "snapshot replay must restore the final terminal surface").toContainText(
        TERMINAL_RENDER_DONE,
        { timeout: 45_000 },
      );

      await expect.poll(
        async () => (await readLatestTerminalPtySize(restoredTerminal))?.text ?? "",
        { message: "terminal fixture must report its baseline PTY size", timeout: 15_000 },
      ).toContain("E2E_PTY_SIZE:");
      const initialPtySize = await readLatestTerminalPtySize(restoredTerminal);
      expect(initialPtySize, "terminal fixture must expose the baseline PTY dimensions").toBeTruthy();

      const viewportBeforeResize = await readRendererViewport(page);
      expect(viewportBeforeResize.width).toBeGreaterThan(0);
      expect(viewportBeforeResize.height).toBeGreaterThan(0);
      const rowsBeforeResize = await restoredTerminal.locator(".xterm-rows > div").count();
      const requestedSize = await resizePrimaryDesktopWindow(electronApp);

      await expect.poll(
        async () => {
          const viewport = await readRendererViewport(page);
          return viewport.width !== viewportBeforeResize.width || viewport.height !== viewportBeforeResize.height
            ? `${viewport.width}x${viewport.height}`
            : "";
        },
        { message: `Electron BrowserWindow must resize its renderer toward ${requestedSize.width}x${requestedSize.height}` },
      ).not.toBe("");
      await expect.poll(
        () => restoredTerminal.locator(".xterm-rows > div").count(),
        { message: "xterm must refit after the Electron content viewport changes" },
      ).not.toBe(rowsBeforeResize);
      await expect.poll(
        async () => {
          const marker = await readLatestTerminalPtySize(restoredTerminal);
          if (!marker || !initialPtySize) return "";
          return marker.sequence > initialPtySize.sequence &&
            (marker.cols !== initialPtySize.cols || marker.rows !== initialPtySize.rows)
            ? marker.text
            : "";
        },
        { message: "SIGWINCH must reach the real PTY with a newer, different size", timeout: 20_000 },
      ).toContain("E2E_PTY_SIZE:");
      await page.waitForTimeout(300);
      const resizedPtySize = await readLatestTerminalPtySize(restoredTerminal);
      expect(resizedPtySize, "terminal fixture must expose the resized PTY dimensions").toBeTruthy();
      expect(resizedPtySize!.sequence).toBeGreaterThan(initialPtySize!.sequence);
      expect(
        resizedPtySize!.cols !== initialPtySize!.cols || resizedPtySize!.rows !== initialPtySize!.rows,
        "resized PTY dimensions must differ from the baseline",
      ).toBe(true);

      const resizedReplayTerminal = await openDesktopPtyPod(page, podKey, true);
      await expect(resizedReplayTerminal, "snapshot after resize must retain the terminal surface").toContainText(
        TERMINAL_RENDER_DONE,
        { timeout: 45_000 },
      );
      await expect(resizedReplayTerminal, "Runner snapshot must include output emitted after SIGWINCH")
        .toContainText(resizedPtySize!.text);

      await testInfo.attach("terminal-render-resized-replay", {
        body: await resizedReplayTerminal.screenshot(),
        contentType: "image/png",
      });
    } finally {
      await invokeIpc<void>(page, "podTerminatePod", podKey).catch(() => undefined);
    }
  });
});

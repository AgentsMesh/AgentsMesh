import { test, expect } from "../../fixtures";
import { invokeIpc } from "../../helpers/ipc";
import {
  TERMINAL_RENDER_DONE,
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
  readLatestTerminalPtySize,
  scrollTerminalToBottom,
  scrollTerminalToTop,
} from "../../../../web/e2e-playwright/helpers/terminal-ui";
import {
  createDesktopPtyPod,
  expectMarkerSelection,
  expectRenderSurface,
  openDesktopPtyPod,
  readRendererViewport,
  resizePrimaryDesktopWindow,
  startDesktopTerminalRenderFixture,
  waitForPodRelayReady,
} from "../../helpers/terminal-render-fixture";

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

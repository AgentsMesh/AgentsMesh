import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import { createMockAgentPod, workspaceUrlForPod } from "../../helpers/mock-agent";
import {
  TERMINAL_RENDER_DONE,
  TERMINAL_ALT_SNAPSHOT_ACTIVE,
  TERMINAL_ALT_SNAPSHOT_ENTER_COMMAND,
  TERMINAL_ALT_SNAPSHOT_EXIT_COMMAND,
  TERMINAL_ALT_SNAPSHOT_EXITED,
  TERMINAL_ALT_SNAPSHOT_NORMAL_HISTORY_TOP,
  TERMINAL_ALT_SNAPSHOT_READY,
  TERMINAL_ALT_SNAPSHOT_SURFACE,
  TERMINAL_NORMAL_BUFFER_SENTINEL,
  readLatestTerminalPtySize,
  scrollTerminalToBottom,
  scrollTerminalToTop,
} from "../../helpers/terminal-ui";
import {
  expectMarkerSelection,
  expectNormalRenderHistory,
  expectRenderSurface,
  startTerminalRenderFixture,
  startWebSocketBlackhole,
  waitForPodRunningThenRelayReady,
} from "../../helpers/terminal-roundtrip-fixture";

// End-to-end coverage of the terminal DATA PLANE after the relay-SSOT
// migration: browser xterm ↔ relayConnection adapter ↔ WasmRelayManager ↔
// Rust RelayConnectionPool ↔ relay WS ↔ runner PTY ↔ e2e-echo (pty mode).
//
// This is the only spec that exercises the relay OUTPUT byte path through the
// adapter (acp-ui-echo covers the ACP-message path). The Rust pool owns
// reconnect/dedup/debounce/codec/snapshot replay; the surviving JS adapter
// must still wire real bytes both ways, verified by the echo round-trip: a
// typed line travels IN (xterm → adapter → … → PTY) and the PTY's reply
// travels back OUT (PTY → … → adapter → xterm).
// pty_runtime.go: writes "ready" on spawn, then echoes each stdin line as
// "got: <line>". We assert the echo (delivered live once subscribed), not the
// one-shot spawn banner — see the round-trip note in the test body.
test.describe("Terminal data-plane round-trip (relay SSOT)", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("propagates cancelled WASM readiness as a rejected JavaScript promise", async ({
    page,
    monitor,
  }) => {
    const blackhole = await startWebSocketBlackhole();
    try {
      await page.goto(`/${TEST_ORG_SLUG}/workspace`);
      await page.waitForLoadState("load");
      await expect.poll(
        () => page.evaluate(() => typeof window.__agentsmeshE2ERelayReadiness?.cancelPendingSubscribe),
        { message: "WASM E2E readiness probe must be installed after platform bootstrap" },
      ).toBe("function");
      await expect.poll(
        () => page.evaluate(() => window.__agentsmeshE2ERelayReadiness?.managerConstructorName),
        { message: "readiness probe must capture the browser's real wasm-bindgen relay manager" },
      ).toBe("WasmRelayManager");

      await page.evaluate((relayUrl) => {
        window.__agentsmeshE2ERelayReadiness!.beginPendingSubscribe(relayUrl);
      }, blackhole.url);
      await blackhole.waitForUpgrade(5_000);

      const outcome = await page.evaluate(async () => {
        try {
          await window.__agentsmeshE2ERelayReadiness!.cancelPendingSubscribe();
          return { rejected: false, message: "" };
        } catch (error) {
          return {
            rejected: true,
            message: error instanceof Error ? `${error.name}: ${error.message}` : String(error),
          };
        }
      });

      expect(outcome.rejected, "wasm-bindgen subscribe Promise must reject after cancellation").toBe(true);
      expect(outcome.message).toMatch(/cancel|closed|removed|readiness|subscription/i);

      // cancelPendingSubscribe does not return until get_status confirms that
      // the Rust driver handle (and therefore its callback owner) is gone. Send
      // real frames only after that barrier: an implementation that drops the
      // wasm Closure without clearing WebSocket.onmessage/onclose will now call
      // the stale trampoline and surface a pageerror.
      await blackhole.sendLateFrameAndClose(5_000);
      await page.evaluate(() => new Promise<void>((resolve) => {
        requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
      }));
      monitor.assertClean();
    } finally {
      await blackhole.close();
    }
  });

  test("attaches, streams pty output, and round-trips typed input through the relay", async ({ page, api, monitor }) => {
    // Realtime EventsService streams through the Next dev-server proxy in local
    // e2e; that proxy intermittently 502s long-lived gRPC streams. It only
    // affects the control-plane event feed (pod-status push), never the relay
    // data plane this spec asserts — acp-ui-echo proves the relay path over the
    // same adapter. Wait-for-running below removes the dependency on the event.
    monitor.allow(/EventsService\/Subscribe.*502|Subscribe:0:0.*502/);

    const pod = await createMockAgentPod(api, { mode: "pty", scenario: "echo" });

    // Gate navigation on the real running state. An initializing pod is active
    // enough for GetPodConnection, but the workspace intentionally will not
    // subscribe a terminal for that stale initial status snapshot.
    await waitForPodRunningThenRelayReady(api, pod.podKey);

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("load");

    // xterm uses the DOM renderer (fit/weblinks/search addons only — no
    // webgl/canvas), so rendered rows are queryable text. Wait for the
    // terminal to mount + its hidden input to attach before driving I/O.
    const term = page.locator(".xterm");
    await expect(term).toBeVisible({ timeout: 30_000 });
    const input = page.locator(".xterm-helper-textarea");
    await expect(input).toBeAttached({ timeout: 30_000 });

    // Assert the round-trip on the INPUT echo, delivered LIVE once the browser
    // subscribes: typed line → onData → relayPool.send → WasmRelayManager →
    // Rust pool → relay → runner PTY → "got: <line>" → back through the relay
    // → xterm. This exercises the surviving adapter's byte path BOTH ways.
    //
    // We deliberately do NOT gate on the one-shot "ready" spawn banner: it is
    // written before the browser subscribes, so catching it E2E hinges on the
    // runner's early-output replay landing ahead of the relay snapshot — a race
    // covered at the daemon layer by TestEarlyOutputReplayedOnAttach and too
    // timing-sensitive to gate this live-infra spec on. Retrying type+assert
    // absorbs the subscription-establishment window (input typed before the
    // subscription is wired is dropped at the PTY, not buffered).
    await expect(async () => {
      await input.focus();
      await page.keyboard.type("relay-roundtrip");
      await page.keyboard.press("Enter");
      await expect(term, "pty echo of typed input must round-trip back to xterm").toContainText(
        "got: relay-roundtrip",
        { timeout: 8_000 },
      );
    }).toPass({ timeout: 60_000 });
  });

  test("preserves fragmented Unicode layout and mouse selection across snapshot replay", async ({ page, api, monitor }, testInfo) => {
    const liveTerminal = await startTerminalRenderFixture(page, api, monitor);

    await expectRenderSurface(liveTerminal);
    await expectNormalRenderHistory(page, liveTerminal);
    await expectMarkerSelection(page, liveTerminal);

    await page.reload();
    await page.waitForLoadState("load");
    const restoredTerminal = page.locator(".xterm");
    await expect(restoredTerminal, "snapshot replay must restore the final terminal surface").toContainText(
      TERMINAL_RENDER_DONE,
      { timeout: 45_000 },
    );
    await expectNormalRenderHistory(page, restoredTerminal);
    await expectMarkerSelection(page, restoredTerminal);

    await testInfo.attach("terminal-render-restored", {
      body: await restoredTerminal.screenshot(),
      contentType: "image/png",
    });
  });

  test("replays an active DECSET 1049 buffer and restores hidden normal history on exit", async ({
    page,
    api,
    monitor,
  }, testInfo) => {
    monitor.allow(/EventsService\/Subscribe.*502|Subscribe:0:0.*502/);

    const pod = await createMockAgentPod(api, {
      mode: "pty",
      scenario: "terminal_alt_snapshot",
    });
    await waitForPodRunningThenRelayReady(api, pod.podKey);

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("load");
    const liveTerminal = page.locator(".xterm");
    await expect(liveTerminal).toContainText(TERMINAL_ALT_SNAPSHOT_READY, {
      timeout: 30_000,
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

    await page.reload();
    await page.waitForLoadState("load");
    const restoredTerminal = page.locator(".xterm");
    await expect(restoredTerminal, "snapshot must restore the active alternate surface")
      .toContainText(TERMINAL_ALT_SNAPSHOT_ACTIVE, { timeout: 45_000 });
    await expect(restoredTerminal, "snapshot must replay alternate-buffer contents")
      .toContainText(TERMINAL_ALT_SNAPSHOT_SURFACE);
    await expect(restoredTerminal, "snapshot must keep the normal surface hidden while 1049 is active")
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
    await expect(restoredTerminal, "hidden normal scrollback must survive active-alt snapshot replay")
      .toContainText(TERMINAL_ALT_SNAPSHOT_NORMAL_HISTORY_TOP);
    await expect(restoredTerminal, "hidden normal sentinel must survive active-alt snapshot replay")
      .toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);
    await expect(restoredTerminal, "alternate content must not leak into restored normal history")
      .not.toContainText(TERMINAL_ALT_SNAPSHOT_ACTIVE);
    await scrollTerminalToBottom(page, restoredTerminal, TERMINAL_ALT_SNAPSHOT_EXITED);

    await testInfo.attach("terminal-active-alt-restored-normal-history", {
      body: await restoredTerminal.screenshot(),
      contentType: "image/png",
    });
  });

  test("propagates PTY SIGWINCH dimensions through resize and snapshot replay", async ({ page, api, monitor }, testInfo) => {
    const liveTerminal = await startTerminalRenderFixture(page, api, monitor);
    await expectRenderSurface(liveTerminal);

    // Establish the baseline from Runner snapshot replay, rather than relying
    // on only the original live stream, before exercising browser-driven fit.
    await page.reload();
    await page.waitForLoadState("load");
    const restoredTerminal = page.locator(".xterm");
    await expect(restoredTerminal, "snapshot replay must restore the final terminal surface").toContainText(
      TERMINAL_RENDER_DONE,
      { timeout: 45_000 },
    );

    // A successful first replay is insufficient if resize destroys the
    // Runner's recovery source after delivering that baseline. Change the real
    // browser viewport, wait for xterm's ResizeObserver path, then reload again.
    const viewport = page.viewportSize();
    expect(viewport, "Chromium E2E must use a fixed viewport").toBeTruthy();
    await expect.poll(
      async () => (await readLatestTerminalPtySize(restoredTerminal))?.text ?? "",
      { message: "terminal fixture must report its baseline PTY size", timeout: 15_000 },
    ).toContain("E2E_PTY_SIZE:");
    const initialPtySize = await readLatestTerminalPtySize(restoredTerminal);
    expect(initialPtySize).toBeTruthy();
    const rowsBeforeResize = await restoredTerminal.locator(".xterm-rows > div").count();
    await page.setViewportSize({
      width: Math.max(800, viewport!.width - 160),
      height: Math.max(600, viewport!.height - 80),
    });
    await expect.poll(
      () => restoredTerminal.locator(".xterm-rows > div").count(),
      { message: "xterm must refit after the browser viewport changes" },
    ).not.toBe(rowsBeforeResize);
    await expect.poll(
      async () => {
        const marker = await readLatestTerminalPtySize(restoredTerminal);
        if (!marker || !initialPtySize) return "";
        return marker.cols !== initialPtySize.cols || marker.rows !== initialPtySize.rows
          ? marker.text
          : "";
      },
      { message: "SIGWINCH must reach the PTY with different dimensions", timeout: 20_000 },
    ).toContain("E2E_PTY_SIZE:");
    await page.waitForTimeout(300);
    const resizedPtySize = await readLatestTerminalPtySize(restoredTerminal);
    expect(resizedPtySize, "terminal fixture must expose the resized PTY dimensions").toBeTruthy();
    expect(
      resizedPtySize!.cols !== initialPtySize!.cols || resizedPtySize!.rows !== initialPtySize!.rows,
      "resized PTY dimensions must differ from the baseline",
    ).toBe(true);

    await page.reload();
    await page.waitForLoadState("load");
    const resizedReplayTerminal = page.locator(".xterm");
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
  });
});

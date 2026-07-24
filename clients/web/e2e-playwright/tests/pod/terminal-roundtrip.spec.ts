import { createHash } from "node:crypto";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import type { Duplex } from "node:stream";

import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import { createMockAgentPod, workspaceUrlForPod } from "../../helpers/mock-agent";
import type { ApiFixture } from "../../fixtures/api.fixture";
import type { ConsoleMonitor } from "../../helpers/console-monitor";
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
} from "../../helpers/terminal-ui";
import type { Locator, Page } from "@playwright/test";

const websocketAcceptGuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";

async function withTimeout<T>(
  promise: Promise<T>,
  timeoutMs: number,
  message: string,
): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<never>((_resolve, reject) => {
        timer = setTimeout(() => reject(new Error(message)), timeoutMs);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

function serverWebSocketFrame(opcode: number, payload: Buffer): Buffer {
  if (payload.length > 125) {
    throw new Error("E2E WebSocket control fixture only supports short frames");
  }
  return Buffer.concat([Buffer.from([0x80 | opcode, payload.length]), payload]);
}

async function startWebSocketBlackhole(): Promise<{
  url: string;
  waitForUpgrade(timeoutMs: number): Promise<void>;
  sendLateFrameAndClose(timeoutMs: number): Promise<void>;
  close(): Promise<void>;
}> {
  const sockets = new Set<Duplex>();
  const server = createServer();
  let activeSocket: Duplex | undefined;
  let inbound = Buffer.alloc(0);
  let resolveUpgrade!: () => void;
  let rejectUpgrade!: (error: Error) => void;
  let upgradeSettled = false;
  const upgraded = new Promise<void>((resolve, reject) => {
    resolveUpgrade = resolve;
    rejectUpgrade = reject;
  });
  let resolvePeerClose!: () => void;
  let rejectPeerClose!: (error: Error) => void;
  const peerCloseAcknowledged = new Promise<void>((resolve, reject) => {
    resolvePeerClose = resolve;
    rejectPeerClose = reject;
  });
  // The promise is awaited after the server sends its close frame. Register a
  // handler now so an early socket failure cannot become an unhandled rejection.
  void peerCloseAcknowledged.catch(() => undefined);
  let resolveSocketClosed!: () => void;
  const socketClosed = new Promise<void>((resolve) => { resolveSocketClosed = resolve; });

  const consumeClientFrames = (chunk: Buffer) => {
    inbound = Buffer.concat([inbound, chunk]);
    while (inbound.length >= 2) {
      const opcode = inbound[0] & 0x0f;
      const masked = (inbound[1] & 0x80) !== 0;
      let payloadLength = inbound[1] & 0x7f;
      let offset = 2;
      if (payloadLength === 126) {
        if (inbound.length < 4) return;
        payloadLength = inbound.readUInt16BE(2);
        offset = 4;
      } else if (payloadLength === 127) {
        if (inbound.length < 10) return;
        const wideLength = inbound.readBigUInt64BE(2);
        if (wideLength > BigInt(Number.MAX_SAFE_INTEGER)) {
          rejectPeerClose(new Error("WebSocket client frame exceeded the safe fixture length"));
          return;
        }
        payloadLength = Number(wideLength);
        offset = 10;
      }
      const maskLength = masked ? 4 : 0;
      const frameLength = offset + maskLength + payloadLength;
      if (inbound.length < frameLength) return;
      inbound = inbound.subarray(frameLength);

      if (opcode === 0x8) {
        if (!masked) {
          rejectPeerClose(new Error("browser close acknowledgement was not masked"));
          return;
        }
        resolvePeerClose();
      }
    }
  };

  server.on("upgrade", (request, socket) => {
    const key = request.headers["sec-websocket-key"];
    if (typeof key !== "string") {
      upgradeSettled = true;
      rejectUpgrade(new Error("WebSocket upgrade omitted Sec-WebSocket-Key"));
      socket.destroy();
      return;
    }
    if (activeSocket) {
      rejectPeerClose(new Error("WASM readiness probe unexpectedly opened a second socket"));
      socket.destroy();
      return;
    }
    const accept = createHash("sha1").update(key + websocketAcceptGuid).digest("base64");
    socket.write([
      "HTTP/1.1 101 Switching Protocols",
      "Upgrade: websocket",
      "Connection: Upgrade",
      `Sec-WebSocket-Accept: ${accept}`,
      "",
      "",
    ].join("\r\n"));
    activeSocket = socket;
    sockets.add(socket);
    socket.on("data", consumeClientFrames);
    socket.once("close", () => {
      sockets.delete(socket);
      resolveSocketClosed();
    });
    socket.once("error", (error) => {
      sockets.delete(socket);
      rejectPeerClose(error);
    });
    if (!upgradeSettled) {
      upgradeSettled = true;
      resolveUpgrade();
    }
  });
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      server.off("error", reject);
      resolve();
    });
  });
  const address = server.address() as AddressInfo;
  return {
    url: `ws://127.0.0.1:${address.port}`,
    async waitForUpgrade(timeoutMs: number): Promise<void> {
      await withTimeout(
        upgraded,
        timeoutMs,
        `WebSocket upgrade not observed within ${timeoutMs}ms`,
      );
    },
    async sendLateFrameAndClose(timeoutMs: number): Promise<void> {
      const socket = activeSocket;
      if (!socket || socket.destroyed) {
        throw new Error("WebSocket fixture has no live upgraded socket");
      }

      const closeReason = Buffer.from("teardown-probe", "utf8");
      const closePayload = Buffer.allocUnsafe(2 + closeReason.length);
      closePayload.writeUInt16BE(1000, 0);
      closeReason.copy(closePayload, 2);
      const lateBinary = serverWebSocketFrame(0x2, Buffer.from([0xde, 0xad, 0xbe, 0xef]));
      const closeFrame = serverWebSocketFrame(0x8, closePayload);
      await new Promise<void>((resolve, reject) => {
        const onError = (error: Error) => reject(error);
        socket.once("error", onError);
        socket.write(Buffer.concat([lateBinary, closeFrame]), () => {
          socket.off("error", onError);
          resolve();
        });
      });

      await withTimeout(
        peerCloseAcknowledged,
        timeoutMs,
        `browser close acknowledgement not observed within ${timeoutMs}ms`,
      );
      if (!socket.destroyed) socket.end();
      await withTimeout(
        socketClosed,
        timeoutMs,
        `WebSocket TCP close not observed within ${timeoutMs}ms`,
      );
    },
    async close(): Promise<void> {
      for (const socket of sockets) socket.destroy();
      await new Promise<void>((resolve, reject) => {
        server.close((error) => error ? reject(error) : resolve());
      });
    },
  };
}

async function waitForPodRunningThenRelayReady(api: ApiFixture, podKey: string): Promise<void> {
  const cc = await api.connect();
  await expect.poll(async () => {
    const pod = await cc.pod.getPod({ orgSlug: TEST_ORG_SLUG, podKey }) as {
      status?: string;
    };
    return pod.status;
  }, {
    message: "workspace hydration must observe the terminal-ready pod state",
    timeout: 30_000,
  }).toBe("running");

  // GetPodConnection sends the runner's subscribe command as a side effect,
  // so call it once after status convergence instead of inside the poll loop.
  const info = await cc.pod.getPodConnection({
    orgSlug: TEST_ORG_SLUG,
    podKey,
  }) as { relayUrl?: string };
  expect(info.relayUrl, "running pod must publish a relay endpoint").toBeTruthy();
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

async function expectNormalRenderHistory(page: Page, terminal: Locator): Promise<void> {
  await scrollTerminalToTop(page, terminal);
  await expect(terminal).toContainText("SCROLL-00|abcdefghijklmnopqrstuvwxyz|");
  await expect(terminal, "normal-buffer contents must survive alternate-buffer exit")
    .toContainText(TERMINAL_NORMAL_BUFFER_SENTINEL);
  await expect(terminal, "alternate-buffer contents must not leak into normal scrollback")
    .not.toContainText(TERMINAL_ALT_BUFFER_PROBE);
  await expect(terminal, "OSC payload must remain non-printing")
    .not.toContainText(TERMINAL_OSC_PAYLOAD);
  await expect(terminal, "DCS payload must remain non-printing")
    .not.toContainText(TERMINAL_DCS_PAYLOAD);
  await scrollTerminalToBottom(page, terminal);
}

async function startTerminalRenderFixture(
  page: Page,
  api: ApiFixture,
  monitor: ConsoleMonitor,
): Promise<Locator> {
  monitor.allow(/EventsService\/Subscribe.*502|Subscribe:0:0.*502/);

  const pod = await createMockAgentPod(api, { mode: "pty", scenario: "terminal_render" });
  await waitForPodRunningThenRelayReady(api, pod.podKey);

  await page.goto(workspaceUrlForPod(pod.podKey));
  await page.waitForLoadState("load");

  const terminal = page.locator(".xterm");
  await expect(terminal).toContainText(TERMINAL_RENDER_READY, { timeout: 30_000 });
  const input = terminal.locator(".xterm-helper-textarea");
  await input.focus();
  await page.keyboard.type(TERMINAL_RENDER_COMMAND);
  await page.keyboard.press("Enter");

  await expect(terminal, "alternate-buffer probe must render before returning to normal mode").toContainText(
    TERMINAL_ALT_BUFFER_PROBE,
    { timeout: 15_000 },
  );
  await expect(terminal).toContainText(TERMINAL_RENDER_DONE, { timeout: 30_000 });
  return terminal;
}

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

import { createHash } from "node:crypto";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import type { Duplex } from "node:stream";

import { expect } from "../fixtures/index";
import type { ApiFixture } from "../fixtures/api.fixture";
import type { ConsoleMonitor } from "./console-monitor";
import { TEST_ORG_SLUG } from "./env";
import { createMockAgentPod, workspaceUrlForPod } from "./mock-agent";
import {
  TERMINAL_RENDER_COMMAND,
  TERMINAL_RENDER_DONE,
  TERMINAL_RENDER_READY,
  TERMINAL_ALT_BUFFER_PROBE,
  TERMINAL_DCS_PAYLOAD,
  TERMINAL_NORMAL_BUFFER_SENTINEL,
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
  scrollTerminalToBottom,
  scrollTerminalToTop,
  selectTerminalMarker,
  waitForTerminalRender,
} from "./terminal-ui";
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

export async function startWebSocketBlackhole(): Promise<{
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

export async function waitForPodRunningThenRelayReady(api: ApiFixture, podKey: string): Promise<void> {
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

export async function expectNormalRenderHistory(page: Page, terminal: Locator): Promise<void> {
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

export async function startTerminalRenderFixture(
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

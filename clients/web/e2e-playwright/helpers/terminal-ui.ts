import type { Locator, Page } from "@playwright/test";

export const TERMINAL_RENDER_COMMAND = "render-probe";
export const TERMINAL_RENDER_READY = "terminal-render-ready";
export const TERMINAL_RENDER_DONE = "E2E_RENDER_DONE";
export const TERMINAL_NORMAL_BUFFER_SENTINEL = "NORMAL-BUFFER-SENTINEL";
export const TERMINAL_ALT_BUFFER_PROBE = "ALT-BUFFER-PROBE";
export const TERMINAL_ALT_SNAPSHOT_ENTER_COMMAND = "enter-alt-buffer";
export const TERMINAL_ALT_SNAPSHOT_EXIT_COMMAND = "exit-alt-buffer";
export const TERMINAL_ALT_SNAPSHOT_READY = "terminal-alt-snapshot-ready";
export const TERMINAL_ALT_SNAPSHOT_ACTIVE = "ALT-BUFFER-ACTIVE";
export const TERMINAL_ALT_SNAPSHOT_SURFACE = "ACTIVE-ALT-SURFACE-MUST-REPLAY";
export const TERMINAL_ALT_SNAPSHOT_EXITED = "ALT-BUFFER-EXITED";
export const TERMINAL_ALT_SNAPSHOT_NORMAL_HISTORY_TOP =
  "ALT-NORMAL-SCROLL-00|abcdefghijklmnopqrstuvwxyz|";
export const TERMINAL_OSC_PAYLOAD = "AgentsMesh terminal E2E";
export const TERMINAL_DCS_PAYLOAD = "ignored-dcs-payload";
export const TERMINAL_SELECTION_ROW = "OFFSET:☰|SELECT-ME|TAIL";
export const TERMINAL_SELECTION_TEXT = "SELECT-ME";
export const TERMINAL_UNICODE_ROW = "COMB:Ae\u0301B|CJK:A界B|EMOJI:A🫠B|ZWJ:A👩‍💻B|VS:A♥️B|END";
export const TERMINAL_PTY_SIZE_PREFIX = "E2E_PTY_SIZE";
export const TERMINAL_UNICODE_GLYPHS = [
  { name: "combining mark", text: "e\u0301", bufferColumns: 1, bufferStartColumn: 6, visualColumns: 1, visualStartColumn: 6 },
  { name: "CJK wide glyph", text: "界", bufferColumns: 2, bufferStartColumn: 14, visualColumns: 2, visualStartColumn: 14 },
  { name: "emoji", text: "🫠", bufferColumns: 2, bufferStartColumn: 25, visualColumns: 2, visualStartColumn: 25 },
  { name: "ZWJ emoji", text: "👩‍💻", bufferColumns: 2, bufferStartColumn: 34, visualColumns: 2, visualStartColumn: 34 },
  { name: "variation-selector sequence", text: "♥️", bufferColumns: 2, bufferStartColumn: 42, visualColumns: 2, visualStartColumn: 42 },
] as const;
export const TERMINAL_UNICODE_SELECTION_TARGETS = [
  { name: "text after ZWJ", text: "B|VS:A", bufferStartColumn: 36, visualStartColumn: 36 },
  { name: "text after VS16", text: "B|END", bufferStartColumn: 44, visualStartColumn: 44 },
] as const;

const OFFSET_TEXT = "OFFSET:";
const WIDE_TEXT = "☰";
const WIDE_TEXT_COLUMNS = 2;
const TARGET_START_COLUMN = OFFSET_TEXT.length + WIDE_TEXT_COLUMNS + 1;

export interface TerminalSelectionMetrics {
  targetLeft: number;
  targetWidth: number;
  mouseY: number;
  mouseStartX: number;
  mouseEndX: number;
}

export interface TerminalRenderMetrics extends TerminalSelectionMetrics {
  cellWidth: number;
  rowHeight: number;
  wideColumns: number;
  targetStartColumn: number;
  targetWidthColumns: number;
}

export interface TerminalSelectionResult {
  text: string;
  handled: boolean;
  overlayLeft: number;
  overlayWidth: number;
}

export interface TerminalUnicodeMetric {
  name: string;
  text: string;
  visualColumns: number;
  visualStartColumn: number;
}

export interface TerminalUnicodeSelectionMetrics extends TerminalSelectionMetrics {
  name: string;
  text: string;
  bufferStartColumn: number;
  bufferLeft: number;
  visualStartColumn: number;
  visualWidthColumns: number;
}

export interface TerminalPtySizeMarker {
  text: string;
  sequence: number;
  cols: number;
  rows: number;
}

function renderedRow(terminal: Locator, text: string): Locator {
  return terminal.locator(".xterm-rows > div").filter({ hasText: text }).last();
}

export async function waitForTerminalRender(terminal: Locator): Promise<void> {
  await terminal.waitFor({ state: "visible", timeout: 30_000 });
  await renderedRow(terminal, TERMINAL_SELECTION_ROW).waitFor({ state: "visible", timeout: 30_000 });
  await renderedRow(terminal, TERMINAL_UNICODE_ROW).waitFor({ state: "visible", timeout: 30_000 });
  await renderedRow(terminal, TERMINAL_RENDER_DONE).waitFor({ state: "visible", timeout: 30_000 });
}

export async function readTerminalRenderRows(terminal: Locator): Promise<string[]> {
  return terminal.locator(".xterm-rows > div").evaluateAll((rows) =>
    rows.map((row) => row.textContent ?? ""),
  );
}

export async function readTerminalRenderMetrics(terminal: Locator): Promise<TerminalRenderMetrics> {
  const row = renderedRow(terminal, TERMINAL_SELECTION_ROW);
  return row.evaluate(
    (element, expected) => {
      const spans = Array.from(element.querySelectorAll("span"));
      const findSpan = (text: string) => spans.find((span) => span.textContent === text);
      const offsetSpan = findSpan(expected.offsetText);
      const wideSpan = findSpan(expected.wideText);
      const targetSpan = findSpan(expected.targetText);
      const screen = element.closest(".xterm")?.querySelector<HTMLElement>(".xterm-screen");
      if (!offsetSpan || !wideSpan || !targetSpan || !screen) {
        throw new Error(
          `terminal fixture spans missing: row=${JSON.stringify(element.textContent)} ` +
          `spans=${JSON.stringify(spans.map((span) => span.textContent))}`,
        );
      }

      const rowRect = element.getBoundingClientRect();
      const offsetRect = offsetSpan.getBoundingClientRect();
      const wideRect = wideSpan.getBoundingClientRect();
      const targetRect = targetSpan.getBoundingClientRect();
      const screenRect = screen.getBoundingClientRect();
      const paddingLeft = Number.parseFloat(getComputedStyle(screen).paddingLeft) || 0;
      const cellWidth = offsetRect.width / expected.offsetText.length;
      const originX = screenRect.left + paddingLeft;

      return {
        cellWidth,
        rowHeight: rowRect.height,
        wideColumns: wideRect.width / cellWidth,
        targetStartColumn: (targetRect.left - originX) / cellWidth,
        targetWidthColumns: targetRect.width / cellWidth,
        targetLeft: targetRect.left,
        targetWidth: targetRect.width,
        mouseY: rowRect.top + rowRect.height / 2,
        mouseStartX: originX + (expected.targetStartColumn + 0.15) * cellWidth,
        mouseEndX: originX + (expected.targetStartColumn + expected.targetLength - 0.15) * cellWidth,
      };
    },
    {
      offsetText: OFFSET_TEXT,
      wideText: WIDE_TEXT,
      targetText: TERMINAL_SELECTION_TEXT,
      targetStartColumn: TARGET_START_COLUMN,
      targetLength: TERMINAL_SELECTION_TEXT.length,
    },
  );
}

export async function readTerminalUnicodeMetrics(
  terminal: Locator,
  cellWidth: number,
): Promise<TerminalUnicodeMetric[]> {
  const row = renderedRow(terminal, TERMINAL_UNICODE_ROW);
  return row.evaluate(
    (element, expected) => {
      const spans = Array.from(element.querySelectorAll("span"));
      const screen = element.closest(".xterm")?.querySelector<HTMLElement>(".xterm-screen");
      if (!screen) throw new Error("terminal Unicode row has no xterm screen");

      const screenRect = screen.getBoundingClientRect();
      const paddingLeft = Number.parseFloat(getComputedStyle(screen).paddingLeft) || 0;
      const originX = screenRect.left + paddingLeft;

      return expected.glyphs.map((glyph) => {
        const span = spans.find((candidate) => candidate.textContent === glyph.text);
        if (!span) {
          throw new Error(
            `terminal Unicode span missing: glyph=${JSON.stringify(glyph.text)} ` +
            `row=${JSON.stringify(element.textContent)} ` +
            `spans=${JSON.stringify(spans.map((candidate) => candidate.textContent))}`,
          );
        }
        const rect = span.getBoundingClientRect();
        return {
          name: glyph.name,
          text: glyph.text,
          visualColumns: rect.width / expected.cellWidth,
          visualStartColumn: (rect.left - originX) / expected.cellWidth,
        };
      });
    },
    { cellWidth, glyphs: TERMINAL_UNICODE_GLYPHS },
  );
}

export async function readTerminalUnicodeSelectionMetrics(
  terminal: Locator,
  cellWidth: number,
): Promise<TerminalUnicodeSelectionMetrics[]> {
  const row = renderedRow(terminal, TERMINAL_UNICODE_ROW);
  return row.evaluate(
    (element, expected) => {
      const spans = Array.from(element.querySelectorAll("span"));
      const screen = element.closest(".xterm")?.querySelector<HTMLElement>(".xterm-screen");
      if (!screen) throw new Error("terminal Unicode selection row has no xterm screen");

      const rowRect = element.getBoundingClientRect();
      const screenRect = screen.getBoundingClientRect();
      const paddingLeft = Number.parseFloat(getComputedStyle(screen).paddingLeft) || 0;
      const originX = screenRect.left + paddingLeft;
      return expected.targets.map((definition) => {
        const target = spans.find((span) => span.textContent === definition.text);
        if (!target) {
          throw new Error(
            `terminal Unicode selection target missing: target=${JSON.stringify(definition.text)} ` +
            `row=${JSON.stringify(element.textContent)}`,
          );
        }
        const targetRect = target.getBoundingClientRect();
        const bufferLeft = originX + definition.bufferStartColumn * expected.cellWidth;
        return {
          name: definition.name,
          text: definition.text,
          bufferStartColumn: definition.bufferStartColumn,
          bufferLeft,
          targetLeft: targetRect.left,
          targetWidth: targetRect.width,
          visualStartColumn: (targetRect.left - originX) / expected.cellWidth,
          visualWidthColumns: targetRect.width / expected.cellWidth,
          mouseY: rowRect.top + rowRect.height / 2,
          mouseStartX: targetRect.left + 0.15 * expected.cellWidth,
          mouseEndX: targetRect.right - 0.15 * expected.cellWidth,
        };
      });
    },
    {
      cellWidth,
      targets: TERMINAL_UNICODE_SELECTION_TARGETS,
    },
  );
}

export async function readLatestTerminalPtySize(
  terminal: Locator,
): Promise<TerminalPtySizeMarker | null> {
  const rows = await readTerminalRenderRows(terminal);
  const pattern = new RegExp(`${TERMINAL_PTY_SIZE_PREFIX}:(\\d+):(\\d+)x(\\d+)`, "g");
  let latest: TerminalPtySizeMarker | null = null;
  for (const row of rows) {
    for (const match of row.matchAll(pattern)) {
      const marker = {
        text: match[0],
        sequence: Number(match[1]),
        cols: Number(match[2]),
        rows: Number(match[3]),
      };
      if (!latest || marker.sequence > latest.sequence) latest = marker;
    }
  }
  return latest;
}

export async function selectTerminalMarker(
  page: Page,
  terminal: Locator,
  metrics: TerminalSelectionMetrics,
): Promise<TerminalSelectionResult> {
  await page.mouse.move(metrics.mouseStartX, metrics.mouseY);
  await page.mouse.down({ button: "left", clickCount: 1 });
  await page.waitForTimeout(40);
  await page.mouse.move(metrics.mouseEndX, metrics.mouseY, { steps: 12 });
  await page.waitForTimeout(40);
  await page.mouse.up({ button: "left", clickCount: 1 });

  const selectionLayer = terminal.locator(".xterm-selection");
  try {
    await selectionLayer.locator(":scope > div").first().waitFor({ state: "visible", timeout: 5_000 });
  } catch (error) {
    const diagnostic = await terminal.evaluate((element, points) => {
      const describeHit = (x: number, y: number) => {
        const hit = document.elementFromPoint(x, y);
        return hit instanceof HTMLElement
          ? { tag: hit.tagName, className: hit.className }
          : null;
      };
      return {
        terminalClass: element.className,
        activeElementClass: document.activeElement instanceof HTMLElement
          ? document.activeElement.className
          : null,
        startHit: describeHit(points.startX, points.y),
        endHit: describeHit(points.endX, points.y),
        selectionChildren: Array.from(element.querySelectorAll(".xterm-selection > div"))
          .map((child) => child.getAttribute("style")),
      };
    }, {
      startX: metrics.mouseStartX,
      endX: metrics.mouseEndX,
      y: metrics.mouseY,
    });
    throw new Error(`xterm mouse drag produced no visible selection: ${JSON.stringify(diagnostic)}; ${String(error)}`);
  }

  const overlay = await selectionLayer.evaluate((element) => {
    const rects = Array.from(element.children)
      .map((child) => child.getBoundingClientRect())
      .filter((rect) => rect.width > 0.5 && rect.height > 0.5);
    if (rects.length === 0) {
      throw new Error("xterm selection layer has no visible rectangle");
    }
    return { left: rects[0].left, width: rects[0].width };
  });

  const copied = await terminal.evaluate((element) => {
    const clipboardData = new DataTransfer();
    const event = new ClipboardEvent("copy", {
      clipboardData,
      bubbles: true,
      cancelable: true,
    });
    element.dispatchEvent(event);
    return {
      text: clipboardData.getData("text/plain"),
      handled: event.defaultPrevented,
    };
  });

  return {
    ...copied,
    overlayLeft: overlay.left,
    overlayWidth: overlay.width,
  };
}

export async function scrollTerminalToTop(page: Page, terminal: Locator): Promise<void> {
  const screen = terminal.locator(".xterm-screen");
  const box = await screen.boundingBox();
  if (!box) throw new Error("terminal screen has no bounding box");
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);

  // xterm 6's custom scrollbar deliberately caps each wheel gesture and
  // animates it. Repeated physical wheel input is required for long buffers;
  // one huge delta does not mean "jump to top".
  for (let attempt = 0; attempt < 20; attempt++) {
    await page.mouse.wheel(0, -1_200);
    await page.waitForTimeout(60);
    if (await renderedRow(terminal, TERMINAL_NORMAL_BUFFER_SENTINEL).isVisible()) {
      await page.waitForTimeout(250);
      return;
    }
  }

  const diagnostic = await terminal.evaluate((element, point) => {
    const rows = Array.from(element.querySelectorAll(".xterm-rows > div"))
      .map((row) => row.textContent ?? "");
    const scrollable = element.querySelector<HTMLElement>(".xterm-scrollable-element");
    const slider = scrollable?.querySelector<HTMLElement>(".scrollbar > .slider");
    const hit = document.elementFromPoint(point.x, point.y);
    return {
      visibleRows: rows,
      scrollable: scrollable && {
        scrollTop: scrollable.scrollTop,
        scrollHeight: scrollable.scrollHeight,
        clientHeight: scrollable.clientHeight,
      },
      sliderStyle: slider?.getAttribute("style"),
      hitClass: hit instanceof HTMLElement ? hit.className : null,
    };
  }, { x: box.x + box.width / 2, y: box.y + box.height / 2 });
  throw new Error(`terminal did not expose the normal-buffer sentinel after real wheel input: ${JSON.stringify(diagnostic)}`);
}

export async function scrollTerminalToBottom(
  page: Page,
  terminal: Locator,
  visibleMarker = TERMINAL_RENDER_DONE,
): Promise<void> {
  const screen = terminal.locator(".xterm-screen");
  const box = await screen.boundingBox();
  if (!box) throw new Error("terminal screen has no bounding box");
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  for (let attempt = 0; attempt < 20; attempt++) {
    await page.mouse.wheel(0, 1_200);
    await page.waitForTimeout(60);
    if (await renderedRow(terminal, visibleMarker).isVisible()) {
      await page.waitForTimeout(250);
      return;
    }
  }
  throw new Error(
    `terminal did not expose ${JSON.stringify(visibleMarker)} after real wheel input`,
  );
}

import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { MutableRefObject } from "react";
import { terminalRegistry } from "@/stores/workspace";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";

export { setupConnection, setupDataHandlers } from "./useTerminalConnection";
export type { TerminalConnection } from "./useTerminalConnection";
export { setupIME, setupImagePaste } from "./useTerminalInputHandlers";

export const TERMINAL_THEME = {
  background: "#1e1e1e",
  foreground: "#d4d4d4",
  cursor: "#d4d4d4",
  cursorAccent: "#1e1e1e",
  selectionBackground: "#264f78",
  black: "#000000",
  red: "#cd3131",
  green: "#0dbc79",
  yellow: "#e5e510",
  blue: "#2472c8",
  magenta: "#bc3fbc",
  cyan: "#11a8cd",
  white: "#e5e5e5",
  brightBlack: "#666666",
  brightRed: "#f14c4c",
  brightGreen: "#23d18b",
  brightYellow: "#f5f543",
  brightBlue: "#3b8eea",
  brightMagenta: "#d670d6",
  brightCyan: "#29b8db",
  brightWhite: "#e5e5e5",
};

export function safeFit(fitAddon: FitAddon): { cols: number; rows: number } | null {
  const dims = fitAddon.proposeDimensions();
  if (!dims || !Number.isFinite(dims.cols) || !Number.isFinite(dims.rows) || dims.cols <= 0 || dims.rows <= 0) {
    return null;
  }
  fitAddon.fit();
  return { cols: dims.cols, rows: dims.rows };
}

export interface SetupTerminalResult {
  term: XTerm;
  fitAddon: FitAddon;
  scheduler: TerminalWriteScheduler;
  deferredFitRafId: number;
}

export function setupTerminal(
  container: HTMLDivElement,
  podKey: string,
  fontSize: number,
  lastSyncedSizeRef: MutableRefObject<{ cols: number; rows: number } | null>,
): SetupTerminalResult {
  const term = new XTerm({
    cursorBlink: true,
    cursorStyle: "block",
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    fontSize,
    lineHeight: 1.2,
    theme: TERMINAL_THEME,
    allowProposedApi: true,
  });

  const fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon());
  term.loadAddon(new SearchAddon());

  term.open(container);

  const deferredFitRafId = requestAnimationFrame(() => {
    const dims = safeFit(fitAddon);
    if (dims) {
      lastSyncedSizeRef.current = dims;
    }
  });

  const scheduler = new TerminalWriteScheduler();
  scheduler.attach(term);

  terminalRegistry.register(podKey, term);

  return { term, fitAddon, scheduler, deferredFitRafId };
}

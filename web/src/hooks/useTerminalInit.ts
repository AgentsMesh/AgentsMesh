import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { MutableRefObject } from "react";
import { terminalPool, terminalRegistry } from "@/stores/workspace";
import type { ConnectionStatus } from "@/stores/terminalConnection";
import { TerminalWriteScheduler } from "@/lib/terminalScheduler";
import { uploadImage } from "@/lib/api/file";
import { toast } from "sonner";

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

/**
 * Safely fit terminal only when container has valid dimensions.
 * Uses FitAddon.proposeDimensions() to check before fitting.
 *
 * @see https://github.com/xtermjs/xterm.js/issues/3029
 */
export function safeFit(fitAddon: FitAddon): { cols: number; rows: number } | null {
  const dims = fitAddon.proposeDimensions();
  if (!dims || !Number.isFinite(dims.cols) || !Number.isFinite(dims.rows) || dims.cols <= 0 || dims.rows <= 0) {
    return null;
  }
  fitAddon.fit();
  return { cols: dims.cols, rows: dims.rows };
}

interface SetupTerminalResult {
  term: XTerm;
  fitAddon: FitAddon;
  scheduler: TerminalWriteScheduler;
  /** Initial dims captured after first rAF layout. */
  initialDims: { value: { cols: number; rows: number } | null };
  /** rAF ID for the deferred fit — caller must cancel on cleanup. */
  deferredFitRafId: number;
}

/**
 * Creates an XTerm instance with addons, opens it in the container,
 * schedules a deferred fit, and registers in the terminal registry.
 */
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

  const initialDims: { value: { cols: number; rows: number } | null } = { value: null };
  const deferredFitRafId = requestAnimationFrame(() => {
    const dims = safeFit(fitAddon);
    if (dims) {
      initialDims.value = dims;
      lastSyncedSizeRef.current = dims;
    }
  });

  const scheduler = new TerminalWriteScheduler();
  scheduler.attach(term);

  terminalRegistry.register(podKey, term);

  return { term, fitAddon, scheduler, initialDims, deferredFitRafId };
}

interface TerminalConnection {
  send: (data: string) => void;
  unsubscribe: () => void;
  disconnect: () => void;
}

/**
 * Subscribes to the terminal pool WebSocket, wiring incoming data
 * through the scheduler. Returns an AbortController for cleanup.
 */
export function setupConnection(
  podKey: string,
  scheduler: TerminalWriteScheduler,
  initialDims: { value: { cols: number; rows: number } | null },
  connectionRef: MutableRefObject<TerminalConnection | null>,
  setConnectionStatus: (status: ConnectionStatus) => void,
  setIsRunnerDisconnected: (v: boolean) => void,
): { abort: AbortController; unsubscribeStatus: () => void } {
  const handleMessage = (data: Uint8Array | string) => {
    if (data instanceof Uint8Array) {
      scheduler.schedule(data);
    } else {
      scheduler.schedule(new TextEncoder().encode(data));
    }
  };

  const subscriptionId = `terminal-${podKey}`;
  const abort = new AbortController();

  (async () => {
    try {
      const handle = await terminalPool.subscribe(podKey, subscriptionId, handleMessage);
      if (abort.signal.aborted) return;
      connectionRef.current = handle;
      if (initialDims.value) {
        terminalPool.forceResize(podKey, initialDims.value.cols, initialDims.value.rows);
      }
    } catch (error) {
      if (abort.signal.aborted) return;
      console.error("Failed to connect terminal:", error);
      setConnectionStatus("error");
    }
  })();

  const unsubscribeStatus = terminalPool.onStatusChange(podKey, (info) => {
    if (info.status !== "none") {
      setConnectionStatus(info.status);
    }
    setIsRunnerDisconnected(info.runnerDisconnected);
  });

  return { abort, unsubscribeStatus };
}

/**
 * Sets up IME composition tracking and mobile cursor position sync
 * on the xterm helper textarea.
 */
export function setupIME(
  container: HTMLDivElement,
  term: XTerm,
  disposables: IDisposable[],
): { isComposing: { current: boolean } } {
  const isComposing = { current: false };

  const textarea = container.querySelector('.xterm-helper-textarea') as HTMLTextAreaElement;
  if (!textarea) return { isComposing };

  const handleCompositionStart = () => { isComposing.current = true; };
  const handleCompositionEnd = () => { isComposing.current = false; };

  textarea.addEventListener('compositionstart', handleCompositionStart);
  textarea.addEventListener('compositionend', handleCompositionEnd);

  disposables.push({
    dispose: () => {
      textarea.removeEventListener('compositionstart', handleCompositionStart);
      textarea.removeEventListener('compositionend', handleCompositionEnd);
    },
  });

  // Mobile cursor position sync
  const syncTextareaPosition = () => {
    const cursorX = term.buffer.active.cursorX;
    const cursorY = term.buffer.active.cursorY - term.buffer.active.viewportY;
    const cellWidth = term.options.fontSize! * 0.6;
    const cellHeight = term.options.fontSize! * (term.options.lineHeight || 1.2);
    textarea.style.left = `${Math.max(0, cursorX * cellWidth)}px`;
    textarea.style.top = `${Math.max(0, cursorY * cellHeight)}px`;
  };

  const cursorDisposable = term.onCursorMove(syncTextareaPosition);
  const writeDisposable = term.onWriteParsed(syncTextareaPosition);
  const initialSyncRafId = requestAnimationFrame(syncTextareaPosition);

  disposables.push(
    cursorDisposable,
    writeDisposable,
    { dispose: () => cancelAnimationFrame(initialSyncRafId) },
  );

  return { isComposing };
}

/**
 * Intercepts paste events containing images — uploads them and sends
 * the resulting URL to the terminal connection.
 */
export function setupImagePaste(
  container: HTMLDivElement,
  connectionRef: MutableRefObject<TerminalConnection | null>,
  disposables: IDisposable[],
): void {
  const handlePaste = (e: ClipboardEvent) => {
    const items = e.clipboardData?.items;
    if (!items) return;

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.type.startsWith('image/')) {
        e.preventDefault();
        e.stopPropagation();
        const blob = item.getAsFile();
        if (!blob) continue;

        // Check connection before starting upload to fail fast
        if (!connectionRef.current) {
          toast.error('Terminal not connected');
          return;
        }

        const toastId = toast.loading('Uploading image...');
        uploadImage(blob)
          .then((url) => {
            // Re-check connection — it may have dropped during upload
            if (!connectionRef.current) {
              toast.error('Terminal disconnected during upload', { id: toastId });
              return;
            }
            connectionRef.current.send(url);
            toast.success('Image uploaded', { id: toastId });
          })
          .catch((err) => {
            toast.error(err instanceof Error ? err.message : 'Failed to upload image', { id: toastId });
          });
        return;
      }
    }
  };

  container.addEventListener('paste', handlePaste, true);
  disposables.push({ dispose: () => container.removeEventListener('paste', handlePaste, true) });
}

/**
 * Wires onData (user input) and onResize handlers on the xterm instance.
 */
export function setupDataHandlers(
  term: XTerm,
  podKey: string,
  connectionRef: MutableRefObject<TerminalConnection | null>,
  isComposing: { current: boolean },
  disposables: IDisposable[],
): void {
  const dataDisposable = term.onData((data) => {
    if (isComposing.current) return;
    connectionRef.current?.send(data);
  });

  const resizeDisposable = term.onResize(({ rows, cols }) => {
    terminalPool.sendResize(podKey, cols, rows);
  });

  disposables.push(dataDisposable, resizeDisposable);
}

import { describe, it, expect, vi, beforeEach } from "vitest";
import { IDisposable } from "@xterm/xterm";
import {
  safeFit,
  TERMINAL_THEME,
  setupIME,
  setupImagePaste,
  setupDataHandlers,
} from "../useTerminalInit";

// Mock dependencies
vi.mock("@/stores/workspace", () => ({
  relayPool: {
    subscribe: vi.fn(),
    forceResize: vi.fn(),
    sendResize: vi.fn(),
    onStatusChange: vi.fn(() => vi.fn()),
  },
  terminalRegistry: {
    register: vi.fn(),
    unregister: vi.fn(),
  },
}));

vi.mock("@/lib/terminalScheduler", () => ({
  TerminalWriteScheduler: vi.fn().mockImplementation(() => ({
    attach: vi.fn(),
    schedule: vi.fn(),
    dispose: vi.fn(),
  })),
}));

vi.mock("@/lib/api/file", () => ({
  uploadImage: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: {
    loading: vi.fn(() => "toast-id"),
    success: vi.fn(),
    error: vi.fn(),
  },
}));

const mockIsTouchPrimaryInput = vi.fn(() => false);
vi.mock("@/lib/platform", () => ({
  isTouchPrimaryInput: () => mockIsTouchPrimaryInput(),
}));

describe("TERMINAL_THEME", () => {
  it("exports a theme with standard terminal colors", () => {
    expect(TERMINAL_THEME.background).toBe("#1e1e1e");
    expect(TERMINAL_THEME.foreground).toBe("#d4d4d4");
    expect(TERMINAL_THEME.red).toBe("#cd3131");
  });
});

describe("safeFit", () => {
  it("returns null when proposeDimensions returns null", () => {
    const fitAddon = { proposeDimensions: vi.fn(() => null), fit: vi.fn() };
    // @ts-expect-error - partial mock
    expect(safeFit(fitAddon)).toBeNull();
    expect(fitAddon.fit).not.toHaveBeenCalled();
  });

  it("returns null when dimensions have non-finite cols", () => {
    const fitAddon = { proposeDimensions: vi.fn(() => ({ cols: Infinity, rows: 24 })), fit: vi.fn() };
    // @ts-expect-error - partial mock
    expect(safeFit(fitAddon)).toBeNull();
  });

  it("returns null when dimensions have zero rows", () => {
    const fitAddon = { proposeDimensions: vi.fn(() => ({ cols: 80, rows: 0 })), fit: vi.fn() };
    // @ts-expect-error - partial mock
    expect(safeFit(fitAddon)).toBeNull();
  });

  it("returns null when dimensions have negative cols", () => {
    const fitAddon = { proposeDimensions: vi.fn(() => ({ cols: -1, rows: 24 })), fit: vi.fn() };
    // @ts-expect-error - partial mock
    expect(safeFit(fitAddon)).toBeNull();
  });

  it("calls fit() and returns dimensions when valid", () => {
    const fitAddon = { proposeDimensions: vi.fn(() => ({ cols: 80, rows: 24 })), fit: vi.fn() };
    // @ts-expect-error - partial mock
    const result = safeFit(fitAddon);
    expect(fitAddon.fit).toHaveBeenCalled();
    expect(result).toEqual({ cols: 80, rows: 24 });
  });
});

describe("setupIME", () => {
  beforeEach(() => {
    mockIsTouchPrimaryInput.mockReturnValue(false);
  });

  it("returns isComposing { current: false } when no textarea found", () => {
    const container = document.createElement("div");
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    const { isComposing } = setupIME(container, term, disposables);

    expect(isComposing.current).toBe(false);
    expect(disposables).toHaveLength(0);
  });

  it("tracks composition start/end events", () => {
    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    const { isComposing } = setupIME(container, term, disposables);

    expect(isComposing.current).toBe(false);

    textarea.dispatchEvent(new Event("compositionstart"));
    expect(isComposing.current).toBe(true);

    textarea.dispatchEvent(new Event("compositionend"));
    expect(isComposing.current).toBe(false);
  });

  it("on desktop, only adds composition cleanup disposable (no cursor/write sync)", () => {
    mockIsTouchPrimaryInput.mockReturnValue(false);

    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    setupIME(container, term, disposables);

    // Desktop: only composition event listener cleanup
    expect(disposables).toHaveLength(1);
    expect(term.onCursorMove).not.toHaveBeenCalled();
    expect(term.onWriteParsed).not.toHaveBeenCalled();
  });

  it("on touch device, adds composition + cursor sync disposables", () => {
    mockIsTouchPrimaryInput.mockReturnValue(true);

    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    setupIME(container, term, disposables);

    // Touch: composition cleanup + cursorMove disposable + rAF cancel
    expect(disposables).toHaveLength(3);
    expect(term.onCursorMove).toHaveBeenCalled();
    // onWriteParsed must NEVER be bound (output→input decoupling)
    expect(term.onWriteParsed).not.toHaveBeenCalled();
  });

  it("on touch device, cursor move callback updates textarea position", () => {
    mockIsTouchPrimaryInput.mockReturnValue(true);

    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    setupIME(container, term, disposables);

    // Capture the callback passed to onCursorMove
    const onCursorMoveMock = term.onCursorMove as ReturnType<typeof vi.fn>;
    const cursorMoveCallback = onCursorMoveMock.mock.calls[0][0] as () => void;

    // Simulate cursor at position (10, 2)
    (term.buffer.active as { cursorX: number }).cursorX = 10;
    (term.buffer.active as { cursorY: number }).cursorY = 2;
    (term.buffer.active as { viewportY: number }).viewportY = 0;
    cursorMoveCallback();

    // cellWidth = 14 * 0.6 = 8.4, cellHeight = 14 * 1.2 = 16.8
    expect(textarea.style.left).toBe(`${10 * 8.4}px`);
    expect(textarea.style.top).toBe(`${2 * 16.8}px`);
  });

  it("on touch device, clamps textarea position to zero when cursor is above viewport", () => {
    mockIsTouchPrimaryInput.mockReturnValue(true);

    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    setupIME(container, term, disposables);

    const onCursorMoveMock = term.onCursorMove as ReturnType<typeof vi.fn>;
    const cursorMoveCallback = onCursorMoveMock.mock.calls[0][0] as () => void;

    // viewportY > cursorY → negative relative position → clamped to 0
    (term.buffer.active as { cursorX: number }).cursorX = 0;
    (term.buffer.active as { cursorY: number }).cursorY = 2;
    (term.buffer.active as { viewportY: number }).viewportY = 5;
    cursorMoveCallback();

    expect(textarea.style.left).toBe("0px");
    expect(textarea.style.top).toBe("0px");
  });

  it("cleans up event listeners on dispose (desktop)", () => {
    mockIsTouchPrimaryInput.mockReturnValue(false);

    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    const { isComposing } = setupIME(container, term, disposables);

    // Dispose all
    disposables.forEach((d) => d.dispose());

    // After cleanup, composition events should not affect isComposing
    textarea.dispatchEvent(new Event("compositionstart"));
    expect(isComposing.current).toBe(false);
  });

  it("cleans up all disposables on dispose (touch device)", () => {
    mockIsTouchPrimaryInput.mockReturnValue(true);

    const container = document.createElement("div");
    const textarea = document.createElement("textarea");
    textarea.className = "xterm-helper-textarea";
    container.appendChild(textarea);
    const term = createMockTerm();
    const disposables: IDisposable[] = [];

    const { isComposing } = setupIME(container, term, disposables);

    expect(disposables).toHaveLength(3);

    // Capture cursorMove disposable mock to verify it was disposed
    const onCursorMoveMock = term.onCursorMove as ReturnType<typeof vi.fn>;
    const cursorMoveDispose = onCursorMoveMock.mock.results[0].value.dispose;

    // Dispose all
    disposables.forEach((d) => d.dispose());

    // Composition cleanup works
    textarea.dispatchEvent(new Event("compositionstart"));
    expect(isComposing.current).toBe(false);

    // CursorMove disposable was disposed
    expect(cursorMoveDispose).toHaveBeenCalled();
  });
});

describe("setupImagePaste", () => {
  it("adds paste event listener to container", () => {
    const container = document.createElement("div");
    const connectionRef = { current: null };
    const disposables: IDisposable[] = [];
    const addSpy = vi.spyOn(container, "addEventListener");

    setupImagePaste(container, connectionRef, disposables);

    expect(addSpy).toHaveBeenCalledWith("paste", expect.any(Function), true);
    expect(disposables).toHaveLength(1);
  });

  it("removes paste event listener on dispose", () => {
    const container = document.createElement("div");
    const connectionRef = { current: null };
    const disposables: IDisposable[] = [];
    const removeSpy = vi.spyOn(container, "removeEventListener");

    setupImagePaste(container, connectionRef, disposables);
    disposables[0].dispose();

    expect(removeSpy).toHaveBeenCalledWith("paste", expect.any(Function), true);
  });
});

describe("setupDataHandlers", () => {
  it("adds data and resize disposables", () => {
    const term = createMockTerm();
    const connectionRef = { current: { send: vi.fn(), unsubscribe: vi.fn(), disconnect: vi.fn() } };
    const isComposing = { current: false };
    const disposables: IDisposable[] = [];

    setupDataHandlers(term, "pod-1", connectionRef, isComposing, disposables);

    expect(disposables).toHaveLength(2);
  });

  it("sends data to connection when not composing", () => {
    const term = createMockTerm();
    const mockSend = vi.fn();
    const connectionRef = { current: { send: mockSend, unsubscribe: vi.fn(), disconnect: vi.fn() } };
    const isComposing = { current: false };
    const disposables: IDisposable[] = [];

    setupDataHandlers(term, "pod-1", connectionRef, isComposing, disposables);

    // Trigger the onData callback
    const dataCallback = term._onDataCallback;
    dataCallback?.("hello");

    expect(mockSend).toHaveBeenCalledWith("hello");
  });

  it("skips sending data when composing", () => {
    const term = createMockTerm();
    const mockSend = vi.fn();
    const connectionRef = { current: { send: mockSend, unsubscribe: vi.fn(), disconnect: vi.fn() } };
    const isComposing = { current: true };
    const disposables: IDisposable[] = [];

    setupDataHandlers(term, "pod-1", connectionRef, isComposing, disposables);

    const dataCallback = term._onDataCallback;
    dataCallback?.("hello");

    expect(mockSend).not.toHaveBeenCalled();
  });

  it("calls relayPool.sendResize on terminal resize", async () => {
    const { relayPool } = await import("@/stores/workspace");
    const term = createMockTerm();
    const connectionRef = { current: null };
    const isComposing = { current: false };
    const disposables: IDisposable[] = [];

    setupDataHandlers(term, "pod-1", connectionRef, isComposing, disposables);

    const resizeCallback = term._onResizeCallback;
    resizeCallback?.({ rows: 24, cols: 80 });

    expect(relayPool.sendResize).toHaveBeenCalledWith("pod-1", 80, 24);
  });
});

// Helper: create a minimal mock XTerm
function createMockTerm() {
  const term = {
    _onDataCallback: null as ((data: string) => void) | null,
    _onResizeCallback: null as ((size: { rows: number; cols: number }) => void) | null,
    buffer: {
      active: { cursorX: 0, cursorY: 0, viewportY: 0 },
    },
    options: { fontSize: 14, lineHeight: 1.2 },
    onData: vi.fn((cb: (data: string) => void) => {
      term._onDataCallback = cb;
      return { dispose: vi.fn() };
    }),
    onResize: vi.fn((cb: (size: { rows: number; cols: number }) => void) => {
      term._onResizeCallback = cb;
      return { dispose: vi.fn() };
    }),
    onCursorMove: vi.fn(() => ({ dispose: vi.fn() })),
    onWriteParsed: vi.fn(() => ({ dispose: vi.fn() })),
    loadAddon: vi.fn(),
    open: vi.fn(),
    dispose: vi.fn(),
    cols: 80,
    rows: 24,
  };

  return term as unknown as import("@xterm/xterm").Terminal & {
    _onDataCallback: ((data: string) => void) | null;
    _onResizeCallback: ((size: { rows: number; cols: number }) => void) | null;
  };
}

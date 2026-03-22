import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalResize } from "../useTerminalResize";

// Mock dependencies
const mockForceResize = vi.fn();

vi.mock("@/stores/workspace", () => ({
  relayPool: {
    forceResize: (...args: unknown[]) => mockForceResize(...args),
  },
}));

vi.mock("../useTerminalInit", () => ({
  safeFit: vi.fn((fitAddon: { _mockDims: { cols: number; rows: number } | null }) => {
    if (!fitAddon) return null;
    const dims = fitAddon._mockDims;
    return dims;
  }),
}));

function createMockRefs(options: {
  dims?: { cols: number; rows: number } | null;
  termCols?: number;
  termRows?: number;
} = {}) {
  const { dims = { cols: 80, rows: 24 }, termCols = 80, termRows = 24 } = options;
  return {
    fitAddonRef: { current: { _mockDims: dims } as unknown as import("@xterm/addon-fit").FitAddon },
    xtermRef: {
      current: {
        cols: termCols,
        rows: termRows,
        focus: vi.fn(),
        options: { fontSize: 14 },
      } as unknown as import("@xterm/xterm").Terminal,
    },
    containerRef: { current: document.createElement("div") },
  };
}

describe("useTerminalResize", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("syncSize", () => {
    it("calls forceResize with current terminal dimensions", () => {
      const refs = createMockRefs();
      const { result } = renderHook(() =>
        useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, 14)
      );

      act(() => {
        result.current.syncSize();
      });

      expect(mockForceResize).toHaveBeenCalledWith("pod-1", 80, 24);
    });

    it("does not call forceResize when terminal cols is 0", () => {
      const refs = createMockRefs({ termCols: 0 });
      const { result } = renderHook(() =>
        useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, 14)
      );

      act(() => {
        result.current.syncSize();
      });

      expect(mockForceResize).not.toHaveBeenCalled();
    });

    it("does not call forceResize when xtermRef is null", () => {
      const refs = createMockRefs();
      refs.xtermRef.current = null as unknown as import("@xterm/xterm").Terminal;
      const { result } = renderHook(() =>
        useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, 14)
      );

      act(() => {
        result.current.syncSize();
      });

      expect(mockForceResize).not.toHaveBeenCalled();
    });

    it("skips duplicate syncSize calls with same dimensions", () => {
      const refs = createMockRefs();
      const { result } = renderHook(() =>
        useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, 14)
      );

      act(() => {
        result.current.syncSize();
      });
      act(() => {
        result.current.syncSize();
      });

      expect(mockForceResize).toHaveBeenCalledTimes(1);
    });
  });

  describe("font size update", () => {
    it("updates term.options.fontSize when fontSize changes", () => {
      const refs = createMockRefs();
      const { rerender } = renderHook(
        ({ fontSize }) =>
          useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, fontSize),
        { initialProps: { fontSize: 14 } }
      );

      rerender({ fontSize: 18 });

      expect(refs.xtermRef.current!.options.fontSize).toBe(18);
    });
  });

  describe("active pane focus", () => {
    it("focuses terminal when pane becomes active", () => {
      const refs = createMockRefs();
      const focusMock = refs.xtermRef.current!.focus as ReturnType<typeof vi.fn>;

      renderHook(
        ({ isActive }) =>
          useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, isActive, 14),
        { initialProps: { isActive: true } }
      );

      expect(focusMock).toHaveBeenCalled();
    });

    it("does not focus when pane is not active", () => {
      const refs = createMockRefs();
      const focusMock = refs.xtermRef.current!.focus as ReturnType<typeof vi.fn>;

      renderHook(
        ({ isActive }) =>
          useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, isActive, 14),
        { initialProps: { isActive: false } }
      );

      expect(focusMock).not.toHaveBeenCalled();
    });
  });

  describe("null refs guard", () => {
    it("does not throw when fitAddonRef is null", () => {
      const refs = createMockRefs();
      refs.fitAddonRef.current = null as unknown as import("@xterm/addon-fit").FitAddon;

      expect(() => {
        renderHook(() =>
          useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, 14)
        );
      }).not.toThrow();
    });

    it("does not throw when containerRef is null", () => {
      const refs = createMockRefs();
      refs.containerRef.current = null as unknown as HTMLDivElement;

      expect(() => {
        renderHook(() =>
          useTerminalResize("pod-1", refs.fitAddonRef, refs.xtermRef, refs.containerRef, true, 14)
        );
      }).not.toThrow();
    });
  });
});

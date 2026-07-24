import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setupTerminal } from "../useTerminalInit";
import { UnicodeGraphemesAddon } from "@xterm/addon-unicode-graphemes";
import { RUNNER_UNICODE_VERSION } from "@/lib/terminal/runnerUnicodeGraphemes";

const mocks = vi.hoisted(() => ({
  events: [] as string[],
  deferredFit: undefined as FrameRequestCallback | undefined,
  proposedDimensions: { cols: 80, rows: 24 } as { cols: number; rows: number } | null,
}));

vi.mock("@xterm/xterm", () => ({
  Terminal: class Terminal {
    private readonly addons: Array<{ activate?: (terminal: Terminal) => void; dispose: () => void }> = [];
    readonly unicode: {
      register: (provider: { version: string }) => void;
      activeVersion: string;
    };

    constructor() {
      const versions = new Set(["6"]);
      let activeVersion = "6";
      this.unicode = {
        register: (provider) => versions.add(provider.version),
        get activeVersion() {
          return activeVersion;
        },
        set activeVersion(version: string) {
          if (!versions.has(version)) throw new Error(`unknown Unicode version "${version}"`);
          activeVersion = version;
          mocks.events.push(`unicode:${version}`);
        },
      };
    }

    loadAddon(addon: { activate?: (terminal: Terminal) => void; dispose: () => void }) {
      mocks.events.push(`load:${addon.constructor.name}`);
      this.addons.push(addon);
      addon.activate?.(this);
    }

    open() {
      mocks.events.push(`open:${this.unicode.activeVersion}`);
    }

    dispose() {
      for (const addon of this.addons.reverse()) addon.dispose();
    }
  },
}));

vi.mock("@xterm/addon-fit", () => ({
  FitAddon: class FitAddon {
    activate() {}
    dispose() {}
    fit() {}
    proposeDimensions() {
      return mocks.proposedDimensions;
    }
  },
}));

vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: class WebLinksAddon {
    activate() {}
    dispose() {}
  },
}));

vi.mock("@xterm/addon-search", () => ({
  SearchAddon: class SearchAddon {
    activate() {}
    dispose() {}
  },
}));

vi.mock("@/lib/terminalScheduler", () => ({
  TerminalWriteScheduler: class TerminalWriteScheduler {
    attach() {}
    dispose() {}
  },
}));

vi.mock("@/stores/workspace", () => ({
  terminalRegistry: {
    register: vi.fn(),
    unregister: vi.fn(),
  },
}));

describe("setupTerminal Unicode width provider", () => {
  beforeEach(() => {
    mocks.events.length = 0;
    mocks.deferredFit = undefined;
    mocks.proposedDimensions = { cols: 80, rows: 24 };
    vi.stubGlobal("requestAnimationFrame", vi.fn((callback: FrameRequestCallback) => {
      mocks.deferredFit = callback;
      return 1;
    }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("activates the Runner Unicode profile before open and disposes its addon", () => {
    const container = document.createElement("div");
    const lastSyncedSizeRef = { current: null };
    const dispose = vi.spyOn(UnicodeGraphemesAddon.prototype, "dispose");

    const { term } = setupTerminal(container, "pod-1", 14, lastSyncedSizeRef);

    expect(mocks.events).toContain("load:RunnerUnicodeGraphemesAddon");
    expect(mocks.events).toContain(`unicode:${RUNNER_UNICODE_VERSION}`);
    expect(mocks.events).toContain(`open:${RUNNER_UNICODE_VERSION}`);
    expect(mocks.events.indexOf("load:RunnerUnicodeGraphemesAddon")).toBeLessThan(
      mocks.events.indexOf(`unicode:${RUNNER_UNICODE_VERSION}`),
    );
    expect(mocks.events.indexOf(`unicode:${RUNNER_UNICODE_VERSION}`)).toBeLessThan(
      mocks.events.indexOf(`open:${RUNNER_UNICODE_VERSION}`),
    );

    term.dispose();

    expect(dispose).toHaveBeenCalledOnce();
    expect(mocks.events.at(-1)).toBe("unicode:6");
    dispose.mockRestore();
  });

  it("publishes only a valid deferred fit to the caller", () => {
    const container = document.createElement("div");
    const lastSyncedSizeRef: { current: { cols: number; rows: number } | null } = {
      current: null,
    };

    const result = setupTerminal(container, "pod-fit", 14, lastSyncedSizeRef);
    expect(result.deferredFitRafId).toBe(1);
    mocks.deferredFit?.(0);
    expect(lastSyncedSizeRef.current).toEqual({ cols: 80, rows: 24 });
    result.term.dispose();

    mocks.proposedDimensions = null;
    lastSyncedSizeRef.current = null;
    const invalid = setupTerminal(container, "pod-invalid-fit", 14, lastSyncedSizeRef);
    mocks.deferredFit?.(0);
    expect(lastSyncedSizeRef.current).toBeNull();
    invalid.term.dispose();
  });
});

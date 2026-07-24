import { beforeEach, describe, expect, it, vi } from "vitest";
import type { IUnicodeHandling, IUnicodeVersionProvider, Terminal } from "@xterm/xterm";

const upstream = vi.hoisted(() => ({
  activate: vi.fn(),
  dispose: vi.fn(),
}));

vi.mock("@xterm/addon-unicode-graphemes", () => ({
  UnicodeGraphemesAddon: class UnicodeGraphemesAddon {
    activate(terminal: Terminal) {
      return upstream.activate(terminal);
    }

    dispose() {
      upstream.dispose();
    }
  },
}));

import {
  RUNNER_UNICODE_VERSION,
  RUNEWIDTH_UNICODE_VERSION,
  RUNEWIDTH_VERSION,
  runnerRuneWidth,
  RunnerUnicodeGraphemeProvider,
  RunnerUnicodeGraphemesAddon,
} from "./runnerUnicodeGraphemes";

describe("runnerRuneWidth", () => {
  it.each([Number.NaN, 1.5, -1, 0x110000])("rejects invalid codepoint %s", (codepoint) => {
    expect(runnerRuneWidth(codepoint)).toBe(0);
  });

  it("matches zero, single, and double-width Runner codepoints", () => {
    expect(runnerRuneWidth(0x0301)).toBe(0);
    expect(runnerRuneWidth("A".codePointAt(0)!)).toBe(1);
    expect(runnerRuneWidth("😀".codePointAt(0)!)).toBe(2);
    expect(RUNNER_UNICODE_VERSION).toContain(RUNEWIDTH_UNICODE_VERSION);
    expect(RUNEWIDTH_VERSION).toMatch(/^v/);
  });
});

describe("RunnerUnicodeGraphemeProvider", () => {
  it("delegates segmentation while replacing standalone cell widths", () => {
    const charProperties = vi.fn().mockReturnValue(0b100000);
    const provider = new RunnerUnicodeGraphemeProvider({
      version: "15-graphemes",
      wcwidth: vi.fn(() => 1),
      charProperties,
    });

    expect(provider.version).toBe(RUNNER_UNICODE_VERSION);
    expect(provider.wcwidth(0x0301)).toBe(0);
    expect(provider.wcwidth("😀".codePointAt(0)!)).toBe(2);
    expect((provider.charProperties(0xFE0F, 0) >> 1) & 0b11).toBe(2);
    expect((provider.charProperties("A".codePointAt(0)!, 0) >> 1) & 0b11).toBe(1);
    expect(provider.charProperties("A".codePointAt(0)!, 0) & 0b100000).toBe(0b100000);
  });

  it("keeps the widest encoded or Runner width for joined graphemes", () => {
    const charProperties = vi.fn()
      .mockReturnValueOnce(0b000011)
      .mockReturnValueOnce(0b000011);
    const provider = new RunnerUnicodeGraphemeProvider({
      version: "15-graphemes",
      wcwidth: vi.fn(() => 1),
      charProperties,
    });

    const precedingWide = 0b000100;
    expect((provider.charProperties("a".codePointAt(0)!, precedingWide) >> 1) & 0b11).toBe(2);
    expect((provider.charProperties("😀".codePointAt(0)!, 0b000010) >> 1) & 0b11).toBe(2);
  });
});

type UnicodeFixture = {
  unicode: IUnicodeHandling;
  registered: IUnicodeVersionProvider[];
  getActive: () => string;
  method: () => boolean;
  label: string;
};

function terminalFixture(): UnicodeFixture {
  const registered: IUnicodeVersionProvider[] = [];
  let activeVersion = "6";
  const fixture = {
    registered,
    label: "terminal-value",
    method() {
      return this === fixture;
    },
    unicode: {
      get versions() {
        return ["6", ...registered.map((provider) => provider.version)];
      },
      get activeVersion() {
        return activeVersion;
      },
      set activeVersion(version: string) {
        activeVersion = version;
      },
      register(provider: IUnicodeVersionProvider) {
        registered.push(provider);
      },
    },
    getActive: () => activeVersion,
  };
  return fixture;
}

describe("RunnerUnicodeGraphemesAddon", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("decorates only the upstream provider and maps activation to the Runner version", () => {
    const terminal = terminalFixture();
    const graphemeProvider: IUnicodeVersionProvider = {
      version: "15-graphemes",
      wcwidth: () => 1,
      charProperties: () => 2,
    };
    const customProvider: IUnicodeVersionProvider = {
      version: "custom",
      wcwidth: () => 1,
      charProperties: () => 2,
    };
    let facade!: Terminal;
    upstream.activate.mockImplementationOnce((received: Terminal) => {
      facade = received;
      expect(received.unicode.versions).toEqual(["6"]);
      expect((received as unknown as UnicodeFixture).label).toBe("terminal-value");
      expect((received as unknown as UnicodeFixture).method()).toBe(true);
      received.unicode.register(graphemeProvider);
      received.unicode.register(customProvider);
      received.unicode.activeVersion = "15-graphemes";
    });
    const addon = new RunnerUnicodeGraphemesAddon();

    addon.activate(terminal as unknown as Terminal);

    expect(terminal.registered[0]).toBeInstanceOf(RunnerUnicodeGraphemeProvider);
    expect(terminal.registered[1]).toBe(customProvider);
    expect(terminal.getActive()).toBe(RUNNER_UNICODE_VERSION);
    expect(facade.unicode.activeVersion).toBe(RUNNER_UNICODE_VERSION);

    // Mapping is scoped to the upstream addon's activate call. Later callers
    // retain explicit control over the upstream version name.
    facade.unicode.activeVersion = "15-graphemes";
    expect(terminal.getActive()).toBe("15-graphemes");

    addon.dispose();
    expect(upstream.dispose).toHaveBeenCalledOnce();
  });

  it("always leaves activation mode after an upstream failure", () => {
    const terminal = terminalFixture();
    let facade!: Terminal;
    upstream.activate.mockImplementationOnce((received: Terminal) => {
      facade = received;
      throw new Error("upstream activation failed");
    });
    const addon = new RunnerUnicodeGraphemesAddon();

    expect(() => addon.activate(terminal as unknown as Terminal)).toThrow(
      "upstream activation failed",
    );
    facade.unicode.activeVersion = "15-graphemes";
    expect(terminal.getActive()).toBe("15-graphemes");
  });
});

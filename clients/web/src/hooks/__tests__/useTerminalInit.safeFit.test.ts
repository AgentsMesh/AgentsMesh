import { describe, expect, it, vi } from "vitest";

import { safeFit } from "../useTerminalInit";

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

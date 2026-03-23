import { describe, it, expect, vi, afterEach } from "vitest";
import { isTouchPrimaryInput } from "../platform";

describe("isTouchPrimaryInput", () => {
  const originalMatchMedia = window.matchMedia;

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
  });

  it("returns true when primary pointer is coarse (touch device)", () => {
    window.matchMedia = vi.fn((query: string) => ({
      matches: query === "(pointer: coarse)",
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    expect(isTouchPrimaryInput()).toBe(true);
    expect(window.matchMedia).toHaveBeenCalledWith("(pointer: coarse)");
  });

  it("returns false when primary pointer is fine (desktop)", () => {
    window.matchMedia = vi.fn(() => ({
      matches: false,
      media: "(pointer: coarse)",
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    expect(isTouchPrimaryInput()).toBe(false);
  });
});

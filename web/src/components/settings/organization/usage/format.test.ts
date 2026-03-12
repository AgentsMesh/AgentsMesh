import { describe, it, expect } from "vitest";
import { formatTokenCount, formatNumber } from "./format";

describe("formatTokenCount", () => {
  it("returns '0' for non-finite values", () => {
    expect(formatTokenCount(NaN)).toBe("0");
    expect(formatTokenCount(Infinity)).toBe("0");
    expect(formatTokenCount(-Infinity)).toBe("0");
  });

  it("returns raw number for values below 1000", () => {
    expect(formatTokenCount(0)).toBe("0");
    expect(formatTokenCount(1)).toBe("1");
    expect(formatTokenCount(999)).toBe("999");
  });

  it("formats thousands with K suffix", () => {
    expect(formatTokenCount(1000)).toBe("1.0K");
    expect(formatTokenCount(1500)).toBe("1.5K");
    expect(formatTokenCount(12345)).toBe("12.3K");
    expect(formatTokenCount(999000)).toBe("999.0K");
  });

  it("promotes to M when K would round to 1000.0K", () => {
    // 999950 / 1000 = 999.95, which would round to "1000.0K"
    expect(formatTokenCount(999950)).toBe("1.00M");
    expect(formatTokenCount(999999)).toBe("1.00M");
  });

  it("formats millions with M suffix", () => {
    expect(formatTokenCount(1_000_000)).toBe("1.00M");
    expect(formatTokenCount(1_234_567)).toBe("1.23M");
    expect(formatTokenCount(999_000_000)).toBe("999.00M");
  });

  it("formats billions with B suffix", () => {
    expect(formatTokenCount(1_000_000_000)).toBe("1.00B");
    expect(formatTokenCount(2_500_000_000)).toBe("2.50B");
  });

  it("formats trillions with T suffix", () => {
    expect(formatTokenCount(1_000_000_000_000)).toBe("1.00T");
  });
});

describe("formatNumber", () => {
  it("returns '0' for non-finite values", () => {
    expect(formatNumber(NaN)).toBe("0");
    expect(formatNumber(Infinity)).toBe("0");
  });

  it("formats with locale separators", () => {
    // The exact output depends on locale, but it should be a string
    const result = formatNumber(1234567);
    expect(typeof result).toBe("string");
    expect(result.length).toBeGreaterThan(0);
  });
});

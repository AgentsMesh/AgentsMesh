import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cn, formatDate, formatRelativeTime } from "../utils";

describe("cn", () => {
  it("should merge class names", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("should handle conditional classes", () => {
    expect(cn("base", false && "hidden", "extra")).toBe("base extra");
  });

  it("should merge conflicting tailwind classes", () => {
    expect(cn("p-4", "p-6")).toBe("p-6");
  });

  it("should handle empty inputs", () => {
    expect(cn()).toBe("");
  });

  it("should handle undefined and null", () => {
    expect(cn("base", undefined, null, "end")).toBe("base end");
  });
});

describe("formatDate", () => {
  it("should format a valid date string", () => {
    const result = formatDate("2024-06-15T10:30:00Z");
    // Should contain month, day, year
    expect(result).toContain("2024");
    expect(result).toContain("15");
  });

  it("should format a Date object", () => {
    const result = formatDate(new Date("2024-01-01T00:00:00Z"));
    expect(result).toContain("2024");
  });

  it("should return '-' for null", () => {
    expect(formatDate(null)).toBe("-");
  });

  it("should return '-' for undefined", () => {
    expect(formatDate(undefined)).toBe("-");
  });

  it("should return '-' for empty string", () => {
    expect(formatDate("")).toBe("-");
  });
});

describe("formatRelativeTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-06-15T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should return 'just now' for time within 60 seconds", () => {
    expect(formatRelativeTime("2024-06-15T11:59:30Z")).toBe("just now");
  });

  it("should return minutes ago", () => {
    expect(formatRelativeTime("2024-06-15T11:55:00Z")).toBe("5m ago");
  });

  it("should return hours ago", () => {
    expect(formatRelativeTime("2024-06-15T09:00:00Z")).toBe("3h ago");
  });

  it("should return days ago", () => {
    expect(formatRelativeTime("2024-06-13T12:00:00Z")).toBe("2d ago");
  });

  it("should fall back to formatDate for 7+ days", () => {
    const result = formatRelativeTime("2024-06-01T12:00:00Z");
    // Should use formatDate format, containing year
    expect(result).toContain("2024");
  });

  it("should return '-' for null", () => {
    expect(formatRelativeTime(null)).toBe("-");
  });

  it("should return '-' for undefined", () => {
    expect(formatRelativeTime(undefined)).toBe("-");
  });
});

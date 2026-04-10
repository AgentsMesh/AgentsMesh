import { describe, it, expect } from "vitest";
import { ApiError } from "@/lib/api/base";
import { isNonRetryableError } from "../relayConnectionRetry";

describe("isNonRetryableError", () => {
  it("returns true for 400 ApiError", () => {
    expect(isNonRetryableError(new ApiError(400, "Bad Request"))).toBe(true);
  });

  it("returns true for 403 ApiError", () => {
    expect(isNonRetryableError(new ApiError(403, "Forbidden"))).toBe(true);
  });

  it("returns true for 404 ApiError", () => {
    expect(isNonRetryableError(new ApiError(404, "Not Found"))).toBe(true);
  });

  it("returns false for 500 ApiError", () => {
    expect(isNonRetryableError(new ApiError(500, "Internal Server Error"))).toBe(false);
  });

  it("returns false for 503 ApiError", () => {
    expect(isNonRetryableError(new ApiError(503, "Service Unavailable"))).toBe(false);
  });

  it("returns false for generic Error", () => {
    expect(isNonRetryableError(new Error("network error"))).toBe(false);
  });

  it("returns false for non-error values", () => {
    expect(isNonRetryableError("string")).toBe(false);
    expect(isNonRetryableError(null)).toBe(false);
    expect(isNonRetryableError(undefined)).toBe(false);
  });
});

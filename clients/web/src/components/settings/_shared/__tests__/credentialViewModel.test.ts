import { describe, it, expect } from "vitest";
import { getConfiguredKeys } from "../credentialViewModel";

describe("getConfiguredKeys", () => {
  it("unions secret field names and non-secret value keys", () => {
    const keys = getConfiguredKeys({
      configured_fields: ["ANTHROPIC_AUTH_TOKEN"],
      configured_values: { ANTHROPIC_BASE_URL: "https://x" },
    });
    expect(keys).toEqual(
      expect.arrayContaining(["ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL"])
    );
    expect(keys).toHaveLength(2);
  });

  it("dedupes a key that erroneously lands in both slots", () => {
    expect(
      getConfiguredKeys({
        configured_fields: ["DUP"],
        configured_values: { DUP: "x" },
      })
    ).toEqual(["DUP"]);
  });

  it("returns empty for a profile with neither slot", () => {
    expect(getConfiguredKeys({})).toEqual([]);
  });

  it("returns keys sorted, so Go-map iteration order can't make the UI flaky", () => {
    expect(
      getConfiguredKeys({
        configured_fields: ["ZED_KEY", "ALPHA_KEY"],
        configured_values: { MID_URL: "x" },
      })
    ).toEqual(["ALPHA_KEY", "MID_URL", "ZED_KEY"]);
  });
});

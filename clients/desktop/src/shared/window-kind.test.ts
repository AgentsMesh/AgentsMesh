import { describe, it, expect } from "vitest";
import { windowTraits, type WindowKind } from "./window-kind";

describe("windowTraits", () => {
  it("primary: full shell, persistent layout, seeds notifications", () => {
    expect(windowTraits("primary")).toEqual({
      primaryEligible: true,
      reloadOnConfigEdit: true,
      ephemeralLayout: false,
      seedsNotificationOwner: true,
    });
  });

  it("secondary: eligible but ephemeral layout, not the notification seed", () => {
    expect(windowTraits("secondary")).toMatchObject({
      primaryEligible: true,
      ephemeralLayout: true,
      seedsNotificationOwner: false,
    });
  });

  it("terminal popout: not eligible, not reloaded on a cosmetic edit", () => {
    expect(windowTraits("terminal")).toMatchObject({
      primaryEligible: false,
      reloadOnConfigEdit: false,
      ephemeralLayout: true,
    });
  });

  it("channel popout: not eligible (chromeless) but reloaded on a cosmetic edit", () => {
    expect(windowTraits("channel")).toMatchObject({
      primaryEligible: false,
      reloadOnConfigEdit: true,
      ephemeralLayout: true,
    });
  });

  it("no chromeless popout is primaryEligible (oauth/updater/notifications never land on one)", () => {
    expect(windowTraits("terminal").primaryEligible).toBe(false);
    expect(windowTraits("channel").primaryEligible).toBe(false);
  });

  it("falls back to primary for an unknown kind (guards preload's `as WindowKind` cast)", () => {
    expect(windowTraits("nonsense" as WindowKind)).toEqual(windowTraits("primary"));
  });
});

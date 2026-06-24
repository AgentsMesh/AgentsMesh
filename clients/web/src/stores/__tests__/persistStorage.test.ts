import { describe, it, expect, afterEach } from "vitest";
import { resolvePersistStorage } from "../persistStorage";

type WinWithApi = { electronAPI?: { ephemeralLayout?: boolean } };
const setApi = (api: WinWithApi["electronAPI"] | undefined) => {
  (window as unknown as WinWithApi).electronAPI = api;
};

afterEach(() => setApi(undefined));

describe("resolvePersistStorage", () => {
  it("web (no electronAPI) → localStorage", () => {
    setApi(undefined);
    expect(resolvePersistStorage()).toBe(window.localStorage);
  });

  it("primary window (ephemeralLayout false) → localStorage", () => {
    setApi({ ephemeralLayout: false });
    expect(resolvePersistStorage()).toBe(window.localStorage);
  });

  it("non-primary window (ephemeralLayout true) → sessionStorage", () => {
    setApi({ ephemeralLayout: true });
    expect(resolvePersistStorage()).toBe(window.sessionStorage);
  });
});

import { test, expect } from "../../fixtures";

// Smoke only: NODE_ENV=test puts auto_updater in its disabled branch (no
// app-update.yml), so we verify the IPC contract + dev-guard behaviour, not a
// real GitHub feed. The full download→staple→restart flow is exercised by the
// nightly release channel, not here.
type Bridge = {
  invoke: (channel: string, ...args: unknown[]) => Promise<unknown>;
  onUpdaterSnapshot?: unknown;
};

test.describe("Electron · updater IPC", () => {
  test("updater:getVersion returns a non-empty version string", async ({ page }) => {
    const version = await page.evaluate(() => (window as never as { electronAPI: Bridge }).electronAPI.invoke("updater:getVersion"));
    expect(typeof version).toBe("string");
    expect((version as string).length).toBeGreaterThan(0);
  });

  test("updater:getState returns a snapshot; dev-guard keeps it idle", async ({ page }) => {
    const snap = (await page.evaluate(() =>
      (window as never as { electronAPI: Bridge }).electronAPI.invoke("updater:getState"),
    )) as { state?: string; percent?: number };
    expect(snap).toMatchObject({ state: expect.any(String), percent: expect.any(Number) });
    // disabled branch: boot check sends not-available → reduces to idle.
    expect(snap.state).toBe("idle");
  });

  test("updater:check resolves under the dev guard (no real feed, no throw)", async ({ page }) => {
    const result = await page
      .evaluate(() => (window as never as { electronAPI: Bridge }).electronAPI.invoke("updater:check"))
      .then(() => ({ ok: true }))
      .catch((e: Error) => ({ ok: false, error: e.message }));
    expect(result).toEqual({ ok: true });
  });

  test("onUpdaterSnapshot subscriber is exposed on the bridge", async ({ page }) => {
    const hasSubscriber = await page.evaluate(
      () =>
        typeof (window as never as { electronAPI: Bridge }).electronAPI.onUpdaterSnapshot ===
        "function",
    );
    expect(hasSubscriber).toBe(true);
  });
});

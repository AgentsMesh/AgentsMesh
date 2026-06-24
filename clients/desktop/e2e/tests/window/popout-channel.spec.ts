import { test, expect } from "../../fixtures";
import { invokeIpc } from "../../helpers/ipc";

// Phase 2 multi-window regression guard for the generic `window:open` path
// (channel kind). Opens a chromeless single-channel window via IPC and asserts
// it lands on the popout channel route + the primary survives its close (same
// ref-count invariant as the popout-terminal spec, exercised through the
// kind→route mapping in window_ipc.ts). channelId need not be a live channel —
// we're verifying window creation + routing, not message loading.
test.describe("Desktop multi-window · popout channel", () => {
  test("window:open(channel) spawns a chromeless channel window; primary survives close", async ({ page, electronApp }) => {
    const live = ["connecting", "connected", "reconnecting"];

    await invokeIpc(page, "realtime:connect");
    await page.waitForTimeout(1000);

    const wcId = await invokeIpc<number>(page, "window:open", "channel", { id: "1" });
    expect(typeof wcId, "window:open returns the new webContents id").toBe("number");

    const popout = await electronApp.waitForEvent("window", { timeout: 15_000 });
    await popout.waitForLoadState("domcontentloaded");
    expect(popout.url(), "popout loads the chromeless channel route").toContain("/popout/channel/1");

    await popout.close();
    await page.waitForTimeout(1000);

    const after = await invokeIpc<string>(page, "realtime:getState");
    expect(live, `primary realtime must survive popout close, got ${after}`).toContain(after);
  });
});

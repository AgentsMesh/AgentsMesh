import { test, expect } from "../../fixtures";
import { invokeIpc } from "../../helpers/ipc";

// Phase 1 multi-window regression guard.
//
// Desktop popout terminals open as their own BrowserWindow via the
// `window:openTerminal` IPC (clients/desktop/src/main/window_ipc.ts), reusing
// the chromeless `/popout/terminal/:podKey` route. The load-bearing invariant
// this spec protects: closing a popout must NOT tear down realtime for the
// primary window. Main ref-counts realtime:connect/disconnect by
// webContentsId (clients/desktop/src/main/realtime.ts), so the shared EventBus
// stream survives until the LAST window leaves — the pre-fix single-cast
// bridge would `eventsDisconnect()` on the first window to close.
//
// Exercises window lifecycle + ref-counting only; the PTY data plane (relay
// directed multicast to the popout) is covered by the terminal round-trip spec
// plus manual verification — podKey here need not be a live pod.
test.describe("Desktop multi-window · popout terminal", () => {
  test("opens a popout window and the primary's realtime survives its close", async ({ page, electronApp }) => {
    const live = ["connecting", "connected", "reconnecting"];

    await invokeIpc(page, "realtime:connect");
    await page.waitForTimeout(1500);
    const before = await invokeIpc<string>(page, "realtime:getState");
    expect(live, `primary realtime before popout: ${before}`).toContain(before);

    const wcId = await invokeIpc<number>(page, "window:openTerminal", "e2e-popout-probe");
    expect(typeof wcId, "window:openTerminal returns the new webContents id").toBe("number");

    const popout = await electronApp.waitForEvent("window", { timeout: 15_000 });
    await popout.waitForLoadState("domcontentloaded");
    expect(popout.url(), "popout loads the chromeless terminal route").toContain("/popout/terminal/");

    // The popout's RealtimeProvider connects too; closing it must only
    // decrement the ref-count, not disconnect the shared stream.
    await popout.close();
    await page.waitForTimeout(1500);

    const after = await invokeIpc<string>(page, "realtime:getState");
    expect(live, `primary realtime must survive popout close, got ${after}`).toContain(after);
  });
});

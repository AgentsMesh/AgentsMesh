import { test, expect } from "../../fixtures";
import { invokeIpc } from "../../helpers/ipc";

// Primary-designation invariants (WindowRegistry.recomputePrimary). A primary is
// always a full-shell (primaryEligible) window: chromeless popouts never win, and
// when the incumbent primary closes the designation transfers to a surviving full
// window (or becomes none if only popouts remain). window:isPrimary is a main-side
// IPC, so it answers per webContents regardless of renderer auth state.
test.describe("Desktop multi-window · primary election", () => {
  test("a channel popout is never primary, even after the primary closes (#2)", async ({ page, electronApp }) => {
    expect(await invokeIpc<boolean>(page, "window:isPrimary"), "the main window is primary").toBe(true);

    await invokeIpc<number>(page, "window:open", "channel", { id: "1" });
    const popout = await electronApp.waitForEvent("window", { timeout: 15_000 });
    await popout.waitForLoadState("domcontentloaded");
    expect(popout.url()).toContain("/popout/channel/");

    expect(await invokeIpc<boolean>(popout, "window:isPrimary"),
      "a kind:channel popout is chromeless / not primaryEligible").toBe(false);

    // Close the primary; only the chromeless popout remains → there is no primary,
    // so the popout still does not take over (electPrimary skips non-eligible kinds).
    await page.close();
    await popout.waitForTimeout(500);
    expect(await invokeIpc<boolean>(popout, "window:isPrimary"),
      "no primary when only a chromeless popout survives").toBe(false);
  });

  test("primary transfers to a surviving full window when the incumbent closes (#5)", async ({ page, electronApp }) => {
    expect(await invokeIpc<boolean>(page, "window:isPrimary")).toBe(true);

    // Open a second FULL window via File → New Window (kind:secondary, primaryEligible).
    await electronApp.evaluate(({ Menu }) => {
      const file = Menu.getApplicationMenu()?.items.find((i) => i.label === "File");
      const newWindow = file?.submenu?.items.find((i) => i.label === "New Window");
      if (!newWindow) throw new Error("File → New Window menu item not found");
      (newWindow as unknown as { click(): void }).click();
    });
    const secondary = await electronApp.waitForEvent("window", { timeout: 15_000 });
    await secondary.waitForLoadState("domcontentloaded");

    expect(await invokeIpc<boolean>(secondary, "window:isPrimary"),
      "a new full window does not preempt a live primary").toBe(false);

    // Close the incumbent primary → recomputePrimary elects the surviving full window.
    await page.close();
    await expect
      .poll(() => invokeIpc<boolean>(secondary, "window:isPrimary"), { timeout: 5_000 })
      .toBe(true);
  });
});

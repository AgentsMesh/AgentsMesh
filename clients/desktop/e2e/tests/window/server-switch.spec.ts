import { test, expect } from "../../fixtures";
import { invokeIpc } from "../../helpers/ipc";

// On a server switch the AppState rebind tears down all relay links, so chromeless
// popouts CLOSE (their pod/channel is on the old backend and can't migrate) rather
// than reload into a wedged "Connecting forever" — WindowRegistry.applyConfigChange.
test.describe("Desktop multi-window · server switch", () => {
  test("switching servers closes chromeless terminal popouts", async ({ page, electronApp }) => {
    await invokeIpc<number>(page, "window:openTerminal", "e2e-switch-probe");
    const popout = await electronApp.waitForEvent("window", { timeout: 15_000 });
    await popout.waitForLoadState("domcontentloaded");
    expect(popout.url()).toContain("/popout/terminal/");

    const terminalPopouts = () =>
      electronApp.evaluate(({ BrowserWindow }) =>
        BrowserWindow.getAllWindows().filter((w) =>
          w.webContents.getURL().includes("/popout/terminal/"),
        ).length,
      );
    expect(await terminalPopouts(), "the terminal popout is open").toBe(1);

    // Switch to the CN preset (a different URL → urlChanged). Fire-and-forget so the
    // primary's own reload doesn't destroy this invoke's execution context.
    await page.evaluate(() => {
      const api = (window as unknown as {
        electronAPI: { invoke: (c: string, ...a: unknown[]) => Promise<unknown> };
      }).electronAPI;
      void api.invoke("serverConfig:set", { kind: "cn", customLabel: "", customUrl: "" });
    });

    // applyConfigChange closes the terminal popout (its pod is on the old backend).
    await expect.poll(terminalPopouts, { timeout: 8_000 }).toBe(0);
  });
});

import { test, expect } from "../../fixtures";
import { BlocksPage } from "../../pages/blocks.page";

// Covers a class of SSOT-sync bugs where desktop Electron adapters lag
// behind Rust SSOT — e.g. ElectronBlockstoreService previously lacked
// `blocks_json()` causing a BlocksSidebar render crash, then lacked
// `workspaces_json()` hydration causing the sidebar to sit forever on
// "Loading workspace…". Neither surfaced via pageerror alone.
test("Blocks · sidebar mounts without uncaught error", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  const blocks = new BlocksPage(page);
  await blocks.goto();
  await blocks.expectOnPage();

  // Give BlocksSidebar time to mount + hydrate from Rust SSOT.
  await page.waitForTimeout(2000);

  expect(errors, `blocks pageerror: ${errors.join(" | ")}`).toEqual([]);

  // Sidebar must leave the "Loading workspace…" state — a stuck sidebar is
  // the symptom of the Electron adapter's sync getter returning an empty
  // map because async IPC populators never hydrated it.
  await expect(page.getByText(/loading workspace/i)).toHaveCount(0, { timeout: 10_000 });
});

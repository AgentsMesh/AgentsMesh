import { test, expect } from "../../fixtures";
import { BlocksPage } from "../../pages/blocks.page";

// Regression: clicking "+" next to PAGES in the sidebar used to create the
// page AND echo a ghost nest ref locally (op_id used as ref_id before the
// server-assigned ref_id arrived via WS). The sidebar then rendered two
// PageNodes with the same key, triggering a React warning and sometimes
// visible duplicates. Fix: Rust skips ref-level local apply; page-tree also
// dedupes defensively.
test("Blocks · create page adds exactly one sidebar entry", async ({ page }) => {
  const warnings: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error" || msg.type() === "warning") {
      warnings.push(msg.text());
    }
  });

  const blocks = new BlocksPage(page);
  await blocks.goto();
  await blocks.expectOnPage();

  // Wait for sidebar workspace hydration.
  await expect(page.getByText(/loading workspace/i)).toHaveCount(0, { timeout: 10_000 });

  // Click the "+" button in the PAGES header.
  const addButton = page.getByRole("button", { name: /add page|新建页面|create page/i }).first();
  await addButton.click();

  // New page lands in sidebar — grab whichever testid appeared last.
  await page.waitForTimeout(1500);
  const pageItems = page.locator('[data-testid^="blocks-sidebar-page-"]');
  const count = await pageItems.count();
  expect(count).toBeGreaterThan(0);

  // Collect the ids; ensure none repeats.
  const ids = await pageItems.evaluateAll((els) =>
    els.map((el) => el.getAttribute("data-testid")),
  );
  const duplicates = ids.filter((id, i) => ids.indexOf(id) !== i);
  expect(duplicates, `duplicate sidebar page entries: ${duplicates.join(",")}`).toEqual([]);

  // React's "Encountered two children with the same key" must not fire.
  const keyWarns = warnings.filter((w) =>
    /Encountered two children with the same key/i.test(w),
  );
  expect(keyWarns, `React key collision: ${keyWarns.join(" | ")}`).toEqual([]);
});

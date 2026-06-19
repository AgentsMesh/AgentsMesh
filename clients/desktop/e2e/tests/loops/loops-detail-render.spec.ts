import { test, expect } from "../../fixtures";
import { LoopsPage } from "../../pages/loops.page";

// Regression coverage for the desktop loop-detail crash. ElectronLoopService's
// loopToCache (packages/electron-adapter/src/loop_cache_map.ts) was built on a
// dead proto schema and dropped max_concurrent_runs + ~20 other fields, so
// ConfigPanel crashed on undefined.toString(). ConfigPanel only mounts on the
// "Prompt & config" tab — LoopDetailPane defaults to "runs" — which is what the
// user meant by "clicking prompt". We switch tabs via data-testid (the tab
// label is i18n and currently absent on desktop). Render throws here are
// swallowed by the route errorElement so pageerror never fires; the guard is a
// POSITIVE assert on the crash field's seeded value (max_concurrent_runs=1) —
// the ?? 0 guard alone would otherwise mask a dropped field as a silent "0".
// Seed loop: nightly-dependency-audit (deploy/dev/seed/seed.sql).
test.describe("Desktop loops · detail config render", () => {
  test("prompt tab shows the real max-concurrent value (not the crash)", async ({ page }) => {
    const loops = new LoopsPage(page);
    await loops.openLoop("nightly-dependency-audit");
    await loops.expectOnPage();

    await page.getByTestId("loop-tab-prompt").click();

    const panel = page.getByTestId("loop-config-panel");
    await expect(
      panel,
      "config panel never mounted — ConfigPanel threw into the route error boundary?",
    ).toBeVisible({ timeout: 10_000 });

    // Assert the crash field's seeded value via data-testid (locale-independent
    // — the label text is i18n). A drifted loopToCache drops the field → the
    // ?? 0 guard renders "0"; the fix maps the real proto value → "1" (seed).
    await expect(
      page.getByTestId("loop-max-concurrent"),
      "max_concurrent_runs missing — loopToCache dropped it and the ?? 0 guard masked it as 0",
    ).toHaveText("1");
  });
});

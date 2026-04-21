import { test, expect } from "../../fixtures";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";

// Page-level smoke: each dashboard route must mount without throwing a JS
// exception. Catches SSOT-sync regressions where desktop Electron adapters
// lag behind Rust SSOT (e.g. a shim missing `blocks_json()` crashes the
// Blocks sidebar, but the old route-opens smoke didn't notice because React
// ErrorBoundary swallows the error).
//
// Uncaught exceptions bubble through `page.on("pageerror")`; we snapshot
// them per route and assert zero.

const ROUTES = [
  "workspace",
  "blocks",
  "channels",
  "mesh",
  "loops",
  "tickets",
  "infra",
] as const;

test.describe("all pages · no uncaught error", () => {
  for (const route of ROUTES) {
    test(`${route} page mounts without pageerror`, async ({ page }) => {
      const errors: string[] = [];
      page.on("pageerror", (err) => errors.push(err.message));

      await gotoHash(page, `/${TEST_ORG_SLUG}/${route}`);

      // Give React a moment to mount + effects to settle.
      await page.waitForTimeout(1500);

      expect(errors, `pageerror on /${route}: ${errors.join(" | ")}`).toEqual([]);
    });
  }
});

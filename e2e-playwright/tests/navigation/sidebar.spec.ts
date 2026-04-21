import { test, expect } from "@playwright/test";
import { SidebarPage, type NavSection } from "../../pages/sidebar.page";
import { TEST_ORG_SLUG } from "../../helpers/env";

test.describe("Sidebar Navigation", () => {
  let sidebar: SidebarPage;

  test.beforeEach(async ({ page }) => {
    sidebar = new SidebarPage(page, TEST_ORG_SLUG);
    // Start from workspace (default authenticated landing)
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");
  });

  test("navigate between main sections via activity bar", async ({ page }) => {
    const sections: NavSection[] = ["infra", "settings", "workspace"];

    for (const section of sections) {
      await sidebar.navigateTo(section);
      const isOn = await sidebar.isOnSection(section);
      expect(isOn).toBe(true);
    }
  });

  test("activity bar links are visible", async () => {
    const mainSections: NavSection[] = [
      "workspace",
      "tickets",
      "channels",
      "infra",
      "settings",
    ];

    for (const section of mainSections) {
      const link = sidebar.getNavLink(section);
      await expect(link).toBeVisible();
    }
  });

  test("navigate to infra and back to workspace", async ({ page }) => {
    await sidebar.navigateTo("infra");
    expect(page.url()).toContain(`/${TEST_ORG_SLUG}/infra`);

    await sidebar.navigateTo("workspace");
    expect(page.url()).toContain(`/${TEST_ORG_SLUG}/workspace`);
  });
});

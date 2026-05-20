import { test, expect } from "../../fixtures/index";
import { CLEANUP, uniqueEmail } from "../../helpers/test-data";
import { clearAuthRateLimit } from "../../helpers/redis";
import { getApiBaseUrl, getWebBaseUrl } from "../../helpers/env";

// Onboarding spec — guards the kudin.private bug regression and the
// downstream "Create Personal Workspace" flow. The original bug was that
// frontend built the slug as `${user.username}-workspace`, producing
// invalid identifiers for OAuth-derived usernames containing dots. Fix
// routes the call through POST /api/v1/orgs/personal so the server
// derives a slugkit-compliant slug from users.username.

test.describe("Auth · onboarding personal workspace", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => {
    clearAuthRateLimit();
  });

  test("API: POST /orgs/personal derives sanitized slug, no client-side slug needed", async ({ api, db }) => {
    const email = uniqueEmail("onboard");
    const username = `onboarduser${Date.now()}`;
    try { db.cleanup(CLEANUP.userAndOrgsByEmail(email)); } catch { /* noop */ }

    const regRes = await api.postPublic("/api/v1/auth/register", {
      email, username, password: "TestPass123!", name: "Onboard E2E",
    });
    expect(regRes.status).toBe(201);
    const { token } = await regRes.json();
    expect(token).toBeTruthy();

    // Critical: caller sends NO slug, server derives. This is the
    // post-fix contract — kudin.private regression cannot reoccur because
    // userService.EnsureUniqueUsername + orgService.CreatePersonal both
    // funnel through slugkit.Sanitize.
    const createRes = await api.postWithToken("/api/v1/orgs/personal", {}, token);
    expect(createRes.status).toBe(201);
    const { organization } = await createRes.json();
    expect(organization.slug).toMatch(/^[a-z0-9]+(-[a-z0-9]+)*$/);
    expect(organization.slug.endsWith("-workspace")).toBe(true);

    db.cleanup(CLEANUP.userAndOrgsByEmail(email));
  });

  // Legacy dot-username scenario was removed: Phase 4 VALIDATE CONSTRAINT
  // makes that DB state impossible — even direct INSERT with bypass SQL
  // fails the `users_username_format` CHECK. The system invariant after
  // backfill is "no dot in users.username", so there's nothing to test.

  test("UI: Quick Start button calls /orgs/personal and navigates onward", async ({ page, api, db }) => {
    const email = uniqueEmail("uionboard");
    const username = `uionboarduser${Date.now()}`;
    try { db.cleanup(CLEANUP.userAndOrgsByEmail(email)); } catch { /* noop */ }

    const regRes = await api.postPublic("/api/v1/auth/register", {
      email, username, password: "TestPass123!", name: "UI Onboard",
    });
    expect(regRes.status).toBe(201);
    const { token, refresh_token, expires_in } = await regRes.json();

    // Mirror global.setup.ts: inject PersistedSession blob so the wasm
    // bootstrap is happy when /onboarding loads.
    const baseUrl = getWebBaseUrl();
    const expiresAt = Math.floor(Date.now() / 1000) + (expires_in ?? 3600);
    await page.context().addInitScript(
      ({ token, refresh_token, expiresAt, baseUrl }) => {
        const u = new URL(baseUrl);
        const port = u.port ? `_${u.port}` : "";
        const raw = `${u.protocol.replace(":", "")}_${u.hostname.toLowerCase()}${port}`;
        const slug = raw.replace(/[^a-zA-Z0-9]/g, "_").slice(0, 64);
        const blob = {
          access_token: token, refresh_token, expires_at: expiresAt,
          base_url: baseUrl, current_org_slug: null, schema_version: 1,
        };
        localStorage.setItem(`agentsmesh-auth/${slug}/session`, JSON.stringify(blob));
      },
      { token, refresh_token, expiresAt, baseUrl },
    );

    // Watch for the /orgs/personal POST to confirm correct endpoint usage.
    const personalReq = page.waitForRequest(
      (req) => req.url().endsWith("/api/v1/orgs/personal") && req.method() === "POST",
    );

    await page.goto("/onboarding");
    await page.getByRole("button", { name: /Create Personal Workspace|创建个人工作区/i }).click();
    const req = await personalReq;
    expect(req.postData()).toBe("{}");

    // Successful create navigates to setup-runner (per onboarding/page.tsx).
    await page.waitForURL((u) => !u.pathname.endsWith("/onboarding"), { timeout: 15_000 });

    db.cleanup(CLEANUP.userAndOrgsByEmail(email));
  });
});

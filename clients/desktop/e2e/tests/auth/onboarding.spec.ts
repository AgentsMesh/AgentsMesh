import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { getApiBaseUrl } from "../../helpers/env";
import { invokeIpc } from "../../helpers/ipc";
import { resolve } from "node:path";
import { rmSync } from "node:fs";
import { execSync } from "node:child_process";

// Desktop onboarding spec — regression guard for the kudin.private bug
// via the wasm path (organizationApi.createPersonal -> WasmOrgApiService
// -> NAPI org_create_personal -> backend). Distinct from web's lightFetch
// direct path; both must produce slugkit-compliant personal workspace slugs.

const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-onboarding");
const PG_CONTAINER = process.env.AGENTSMESH_POSTGRES_CONTAINER || "agentsmesh-postgres";

test.use({ skipAuthRestore: true, userDataDir: FRESH_USER_DATA });

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test.describe("Desktop · onboarding personal workspace", () => {
  let email: string;
  const password = "TestPass123!";

  test.beforeEach(async () => {
    email = `desktop-onboard-${Date.now()}@test.local`;
    cleanupUser(email);
    const username = `desktoponboard${Date.now()}`;
    const res = await fetch(`${getApiBaseUrl()}/api/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, username, password, name: "Desktop Onboard" }),
    });
    if (!res.ok) throw new Error(`register failed: ${res.status}`);
  });

  test.afterEach(() => {
    cleanupUser(email);
  });

  test("NAPI orgCreatePersonal creates slugkit-compliant workspace", async ({ page }) => {
    // Drive the IPC bridge directly — bypasses React state-loading races
    // and exercises exactly what the Quick Start button delegates to:
    // organizationApi.createPersonal() -> ElectronOrgService.create_personal()
    // -> invoke("orgCreatePersonal") -> Rust org_create_personal ->
    // ApiClient.create_personal_organization() -> backend POST /orgs/personal.

    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await login.login(email, password);

    // Wait for onboarding (no org → router lands here).
    await page.waitForFunction(
      () => window.location.hash.includes("/onboarding"),
      undefined,
      { timeout: 30_000 },
    );

    // Call NAPI directly through the same electronAPI bridge the renderer uses.
    const json = await invokeIpc<string>(page, "orgCreatePersonal");
    const resp = JSON.parse(json);
    expect(resp.slug).toBeTruthy();
    // The whole point: slug derived server-side via slugkit, not client-built.
    expect(resp.slug).toMatch(/^[a-z0-9]+(-[a-z0-9]+)*$/);
    expect(resp.slug.endsWith("-workspace")).toBe(true);

    // Sanity: org now visible via /api/v1/orgs.
    const orgs = await fetchUserOrgs(email);
    expect(orgs.some((o) => o.slug === resp.slug)).toBe(true);
  });
});

function cleanupUser(email: string): void {
  const sql = `DELETE FROM organizations WHERE id IN (SELECT om.organization_id FROM organization_members om JOIN users u ON om.user_id = u.id WHERE u.email = '${email}'); DELETE FROM users WHERE email = '${email}';`;
  try {
    execSync(
      `docker exec ${PG_CONTAINER} psql -U agentsmesh -d agentsmesh -c "${sql}"`,
      { timeout: 10_000, stdio: "pipe" },
    );
  } catch { /* ignore — user may not exist yet */ }
}

async function fetchUserOrgs(email: string): Promise<Array<{ slug: string; id: number; name: string }>> {
  const loginRes = await fetch(`${getApiBaseUrl()}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password: "TestPass123!" }),
  });
  if (!loginRes.ok) throw new Error(`login failed: ${loginRes.status}`);
  const { token } = await loginRes.json();
  const orgsRes = await fetch(`${getApiBaseUrl()}/api/v1/orgs`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const data = await orgsRes.json();
  return data.organizations || [];
}

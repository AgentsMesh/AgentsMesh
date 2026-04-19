/* eslint-disable react-hooks/rules-of-hooks -- Playwright fixtures use function
   arguments named `use` which happen to trigger the React Hooks rule. */
import { test as base, expect, Page } from "@playwright/test";

// E2E fixtures for Block Store specs. Provides:
//   - `token` / `workspaceID`: authenticated session + default workspace id
//   - `authenticatedPage`: a page that's already logged in (JWT in localStorage)
//   - `api`: tiny fetch wrapper with Authorization header pre-applied
//
// Tests should prefer driving state via `api` (deterministic) and asserting
// via the UI (via `authenticatedPage`). Direct-from-UI setup is slower and
// flakier when the backend is also under test.

const API_BASE = `http://localhost:${process.env.HTTP_PORT ?? "15300"}`;
const ORG_SLUG = "dev-org";
const DEV_EMAIL = "dev@agentsmesh.local";
const DEV_PASSWORD = "devpass123";

interface BlockstoreFixtures {
  token: string;
  workspaceID: string;
  /**
   * A freshly-provisioned workspace for this single test. Root block created
   * server-side. Isolates the test from accumulated writes in the shared
   * default workspace, so assertions that depend on "state count" (semantic
   * search ranking, slash-menu contents, etc.) stay stable run-over-run.
   * Cleaned up automatically after the test finishes (best-effort DELETE on
   * the workspace id) so the dev DB doesn't accumulate orphan rows across
   * repeated runs. Failures during teardown are swallowed — we never want a
   * cleanup error to mask the test's real assertion failure.
   */
  isolatedWorkspace: { id: string; rootID: string };
  api: ApiClient;
  authenticatedPage: Page;
}

interface ApiClient {
  post<T = unknown>(path: string, body: unknown): Promise<T>;
  get<T = unknown>(path: string): Promise<T>;
}

async function login(): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: DEV_EMAIL, password: DEV_PASSWORD }),
  });
  if (!res.ok) throw new Error(`login failed: ${res.status}`);
  const json = (await res.json()) as { token: string };
  return json.token;
}

// Shared token cache — fixtures are instantiated once per test, and dev's
// 20/min /auth/login rate limit would 429 a 20+ spec run otherwise. One
// token lives for the whole Playwright process; it's refreshed on token
// expiry (reject with 401 retry path), which the REST wrapper handles.
let cachedToken: Promise<string> | null = null;
function sharedLogin(): Promise<string> {
  if (!cachedToken) cachedToken = login();
  return cachedToken;
}

// Shared workspace id cache — ensureWorkspace is idempotent server-side but
// cheap to skip entirely.
let cachedWorkspaceID: Promise<string> | null = null;
function sharedEnsureWorkspace(token: string): Promise<string> {
  if (!cachedWorkspaceID) cachedWorkspaceID = ensureWorkspace(token);
  return cachedWorkspaceID;
}

async function ensureWorkspace(token: string): Promise<string> {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/${ORG_SLUG}/blocks/workspaces/default`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "X-Organization-Slug": ORG_SLUG,
      },
    },
  );
  if (!res.ok) throw new Error(`ensureWorkspace failed: ${res.status}`);
  const json = (await res.json()) as { id: string };
  return json.id;
}

// provisionIsolatedWorkspace creates a brand-new workspace each call, which
// a test uses instead of the shared default when it needs a clean slate.
// Slug randomisation is aggressive because Playwright's workers + reruns can
// share the same millisecond; collisions surface as 409 which is explicit.
async function provisionIsolatedWorkspace(
  token: string,
): Promise<{ id: string; rootID: string }> {
  const slug = `e2e-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const res = await fetch(`${API_BASE}/api/v1/orgs/${ORG_SLUG}/blocks/workspaces`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "X-Organization-Slug": ORG_SLUG,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ slug, name: slug }),
  });
  if (!res.ok) throw new Error(`provisionIsolatedWorkspace failed: ${res.status}: ${await res.text()}`);
  const json = (await res.json()) as { id: string; root_block_id: string };
  return { id: json.id, rootID: json.root_block_id };
}

// tearDownIsolatedWorkspace deletes the workspace + every row it owns via
// the backend's DELETE /blocks/workspaces/:ws_id. Intentionally best-effort:
// a teardown error must never convert a green test into a red one, so we
// log (via console.warn) and swallow. The worst case is one leaked row;
// periodic resets still catch it.
async function tearDownIsolatedWorkspace(token: string, wsID: string): Promise<void> {
  try {
    await fetch(`${API_BASE}/api/v1/orgs/${ORG_SLUG}/blocks/workspaces/${wsID}`, {
      method: "DELETE",
      headers: {
        Authorization: `Bearer ${token}`,
        "X-Organization-Slug": ORG_SLUG,
      },
    });
  } catch (err) {
    // eslint-disable-next-line no-console
    console.warn(`[e2e] workspace teardown failed for ${wsID}:`, err);
  }
}

function makeApi(token: string): ApiClient {
  const headers = {
    Authorization: `Bearer ${token}`,
    "X-Organization-Slug": ORG_SLUG,
    "Content-Type": "application/json",
  };
  return {
    async post<T>(path: string, body: unknown): Promise<T> {
      const res = await fetch(`${API_BASE}${path}`, {
        method: "POST",
        headers,
        body: JSON.stringify(body ?? {}),
      });
      if (!res.ok) {
        throw new Error(`${path} → ${res.status}: ${await res.text()}`);
      }
      return res.json() as Promise<T>;
    },
    async get<T>(path: string): Promise<T> {
      const res = await fetch(`${API_BASE}${path}`, { headers });
      if (!res.ok) {
        throw new Error(`${path} → ${res.status}: ${await res.text()}`);
      }
      return res.json() as Promise<T>;
    },
  };
}

// Inject the JWT into localStorage before Next.js boots so `useAuthStore`
// hydrates as logged-in on first render. The dev org has a single user, so
// this sidesteps the login form entirely. The key + shape match zustand's
// persist middleware (see stores/auth.ts → name: "agentsmesh-auth").
async function seedAuth(page: Page, token: string): Promise<void> {
  const me = await fetch(`${API_BASE}/api/v1/users/me`, {
    headers: { Authorization: `Bearer ${token}` },
  }).then((r) => r.json());
  const orgs = await fetch(`${API_BASE}/api/v1/orgs`, {
    headers: { Authorization: `Bearer ${token}` },
  }).then((r) => r.json());
  const current = (orgs.organizations ?? []).find((o: { slug: string }) => o.slug === ORG_SLUG);

  await page.addInitScript(
    ({ tok, user, org }) => {
      const state = {
        state: {
          token: tok,
          refreshToken: null,
          user,
          currentOrg: org,
          organizations: [org],
        },
        version: 0,
      };
      window.localStorage.setItem("agentsmesh-auth", JSON.stringify(state));
    },
    { tok: token, user: me.user, org: current },
  );
}

export const test = base.extend<BlockstoreFixtures>({
  token: async ({}, use) => {
    const tok = await sharedLogin();
    await use(tok);
  },
  workspaceID: async ({ token }, use) => {
    const ws = await sharedEnsureWorkspace(token);
    await use(ws);
  },
  isolatedWorkspace: async ({ token }, use) => {
    const ws = await provisionIsolatedWorkspace(token);
    try {
      await use(ws);
    } finally {
      await tearDownIsolatedWorkspace(token, ws.id);
    }
  },
  api: async ({ token }, use) => {
    await use(makeApi(token));
  },
  authenticatedPage: async ({ page, token, workspaceID }, use) => {
    void workspaceID; // ensures default workspace exists before the page loads
    await seedAuth(page, token);
    await use(page);
  },
});

export { expect };
export const orgSlug = ORG_SLUG;
export const apiBase = API_BASE;

import { test, expect } from "@playwright/test";
import { _electron as electron } from "@playwright/test";
import { createServer, type Server } from "node:http";
import { resolve } from "node:path";
import { rmSync } from "node:fs";
import { getElectronMainPath, isCi } from "../../helpers/env";

// Regression for the 2026-05-10 OrbStack incident.
//
// Root cause: desktop main process used to fall back to
// `http://localhost:25350` when AGENTSMESH_API_URL was unset. OrbStack
// happened to listen on that port and returned `502 Bad Gateway` for any
// request, surfacing as `Sign in failed: {"kind":"http","status":502,
// "code":null,"message":"Bad Gateway"}` — completely opaque about WHICH
// host produced the 502.
//
// The fix is two-layered:
//   1. Remove the localhost:25350 fallback (server_config.ts owns defaults).
//   2. Tag every ApiError::Http with the request URL so wire errors
//      become self-describing.
//
// This spec pins layer 2: when the upstream returns 502, the message the
// renderer sees MUST contain the URL it hit. Without that string, no user
// can diagnose this class of bug from a screenshot.

const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-orbstack-conflict");

let mockServer: Server;
let mockPort: number;

test.beforeAll(async () => {
  mockServer = createServer((_req, res) => {
    res.writeHead(502, { "Content-Type": "text/plain" });
    res.end("Bad Gateway"); // mimic OrbStack's 11-byte body verbatim
  });
  await new Promise<void>((r) => mockServer.listen(0, "127.0.0.1", () => r()));
  mockPort = (mockServer.address() as { port: number }).port;
});

test.afterAll(() => {
  mockServer.close();
});

test.beforeEach(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test("502 from upstream surfaces the URL in the error message", async () => {
  const ciArgs = isCi() && process.platform === "linux"
    ? ["--no-sandbox", "--disable-dev-shm-usage"]
    : [];
  const app = await electron.launch({
    args: [getElectronMainPath(), `--user-data-dir=${FRESH_USER_DATA}`, ...ciArgs],
    env: {
      ...process.env,
      AGENTSMESH_API_URL: `http://127.0.0.1:${mockPort}`,
      NODE_ENV: "test",
      ELECTRON_DISABLE_SECURITY_WARNINGS: "true",
    },
    timeout: isCi() ? 120_000 : 30_000,
  });
  try {
    const page = await app.firstWindow();
    await page.waitForLoadState("domcontentloaded");

    // Drive `userGetMe` directly via IPC (no need for actual login —
    // Rust's UserApiService.get_me() hits /api/v1/users/me regardless of
    // auth state, and our mock returns 502 for any path).
    const errorMessage = await page.evaluate(async () => {
      try {
        await window.electronAPI.invoke("userGetMe");
        return null;
      } catch (e) {
        return (e as Error).message;
      }
    });

    expect(errorMessage).not.toBeNull();
    // The wire error includes status 502.
    expect(errorMessage).toContain("502");
    // And — the load-bearing assertion — the URL the request hit. Without
    // this, future regressions producing a 502 from the wrong host would
    // be just as opaque as the original incident.
    expect(errorMessage).toMatch(/127\.0\.0\.1|localhost/);
  } finally {
    await app.close().catch(() => undefined);
  }
});

import { test, expect } from "@playwright/test";
import { _electron as electron } from "@playwright/test";
import { resolve } from "node:path";
import { rmSync } from "node:fs";
import { getElectronMainPath, isCi } from "../../helpers/env";

// Regression for the 2026-05-10 OrbStack incident, layer 1: the packaged
// build must not fall back to localhost when AGENTSMESH_API_URL is unset.
// In the original bug, an empty env triggered the magic
// `?? "http://localhost:25350"` default, which OrbStack happened to occupy.
//
// After server_config.ts SSOT refactor, an unset env + missing server.json
// resolves to DEFAULT (global → https://agentsmesh.ai). This test pins
// that contract — anyone re-introducing a localhost fallback in
// main/index.ts will trip this.

const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-cold-start-no-env");

test.beforeEach(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test("packaged default points at prod when AGENTSMESH_API_URL unset", async () => {
  const ciArgs = isCi() && process.platform === "linux"
    ? ["--no-sandbox", "--disable-dev-shm-usage"]
    : [];

  // Strip AGENTSMESH_API_URL from inherited env. The dev shell or e2e
  // runner may have set it; without this scrub the test would silently
  // use the override path instead of the cold-start path it's pinning.
  const env = Object.fromEntries(
    Object.entries(process.env).filter(([k]) => k !== "AGENTSMESH_API_URL"),
  );

  const app = await electron.launch({
    args: [getElectronMainPath(), `--user-data-dir=${FRESH_USER_DATA}`, ...ciArgs],
    env: {
      ...env,
      NODE_ENV: "test",
      ELECTRON_DISABLE_SECURITY_WARNINGS: "true",
    },
    timeout: isCi() ? 120_000 : 30_000,
  });
  try {
    const page = await app.firstWindow();
    await page.waitForLoadState("domcontentloaded");

    const apiUrl = await page.evaluate(() => window.electronAPI.apiUrl);
    expect(apiUrl).toBe("https://agentsmesh.ai");

    const cfg = await page.evaluate(() => window.electronAPI.serverConfig.snapshot);
    expect(cfg).toEqual({
      kind: "global",
      customLabel: "",
      customUrl: "",
    });
  } finally {
    await app.close().catch(() => undefined);
  }
});

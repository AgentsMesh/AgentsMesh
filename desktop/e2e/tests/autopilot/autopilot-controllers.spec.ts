import { test, expect } from "../../fixtures";

// Guards the Rust↔Backend DTO contract for `/autopilot-controllers`.
// Regression: backend returns `[...]` (array) but Rust ApiClient was typed
// as `AutopilotListResponse { controllers: Vec<...> }` causing a deserialize
// failure ("invalid type: map, expected a sequence"). The IPC call swallows
// the error into a rejected Promise, so pageerror doesn't catch it — we
// must invoke the IPC directly and assert success.
test("Autopilot · IPC autopilotFetchControllers resolves without DTO error", async ({ page }) => {
  const result = await page.evaluate(async () => {
    try {
      const raw = await (window as unknown as {
        electronAPI: { invoke: (ch: string, ...a: unknown[]) => Promise<unknown> };
      }).electronAPI.invoke("autopilotFetchControllers");
      return { ok: true, raw };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : String(err) };
    }
  });

  expect(result.ok, `autopilotFetchControllers rejected: ${("error" in result) ? result.error : ""}`)
    .toBe(true);
});

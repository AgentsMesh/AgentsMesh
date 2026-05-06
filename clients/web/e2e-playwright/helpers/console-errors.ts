import { expect, type Page } from "@playwright/test";

export function collectConsoleErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(msg.text());
  });
  return errors;
}

export function assertNoWasmErrors(errors: string[]) {
  const critical = errors.filter(
    (e) =>
      (e.includes("missing field") ||
      e.includes("is not valid JSON") ||
      e.includes("Failed to fetch board") ||
      e.includes("Failed to fetch topology") ||
      e.includes("Failed to load runner") ||
      e.includes("Failed to load repository") ||
      e.includes("Failed to load ticket")) &&
      !e.includes("Failed to load resource")
  );
  expect(critical).toHaveLength(0);
}

/**
 * wasm-bindgen's RefCell-style borrow check throws this exact string when
 * `&mut self` and `&self` calls overlap on the same WASM object. The ACP
 * session manager regression (render-time wasm read racing relay-driven
 * mutators) surfaced as exactly this message.
 */
export function assertNoWasmRecursiveBorrow(errors: string[]) {
  const offenders = errors.filter((e) =>
    e.includes("recursive use of an object detected"),
  );
  expect(offenders, `wasm borrow conflict regressed:\n${offenders.join("\n")}`).toHaveLength(0);
}

export function collectPageErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));
  return errors;
}

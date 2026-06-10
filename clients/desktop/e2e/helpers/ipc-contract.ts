import type { Page } from "@playwright/test";
import { expect } from "@playwright/test";
import { invokeIpc } from "./ipc";

// A typed bridge error (unauthorized / not_found / invalid_argument /
// connect_failed / …) means the route IS wired — the handler reached the
// service layer and rejected with a recognized error code. We want to keep
// those passing.
//
// A *wire fault* is a JavaScript runtime error that escaped the bridge: the
// command was not registered, the napi binding moved, the preload exposed
// no `invoke`, or the handler threw before returning a typed Result. Those
// produce error messages with shapes like:
//   - "command_x is not a function"
//   - "Cannot read properties of undefined (reading 'invoke')"
//   - "TypeError: ..."
//   - "ReferenceError: ..."
//
// Catching these is the whole point of the IPC contract layer — `toBeDefined`
// alone passes them silently because the catch handler wraps them in
// `{__ipcError}` which is itself defined.
const WIRE_FAULT_PATTERNS = [
  /TypeError\b/i,
  /ReferenceError\b/i,
  /\bis not a function\b/i,
  /Cannot read prop/i,
  /Cannot destructure/i,
  /undefined is not/i,
  /is not registered/i,
  /No such IPC handler/i,
  /No handler registered/i,
  /Handler did not respond/i,
  // ServeMux 404 for an unregistered route — distinct from a connect not_found.
  /\bpage not found\b/i,
];

function isWireFault(msg: string): boolean {
  return WIRE_FAULT_PATTERNS.some((p) => p.test(msg));
}

export interface IpcContractOpts {
  method: string;
  returnType: string;
}

export async function invokeIpcContract(
  page: Page,
  opts: IpcContractOpts,
  ...args: unknown[]
): Promise<void> {
  const { method, returnType } = opts;
  const result = await invokeIpc(page, method, ...args).catch((err: Error) => ({
    __ipcError: err.message,
  }));

  // A void/() return surfaces as undefined on success — that's the expected
  // signal that the handler completed without payload. We accept it before
  // the generic undefined-rejection so we don't conflate "preload not
  // wired" (no result at all) with "void handler returned cleanly."
  if (result === undefined && /^(void|\(\))$/i.test(returnType.trim())) return;

  expect(result, `${method}: bridge returned undefined — preload not wired?`).not.toBeUndefined();

  if (result && typeof result === "object" && "__ipcError" in result) {
    const msg = String((result as { __ipcError: unknown }).__ipcError);
    expect(
      isWireFault(msg),
      `${method}: bridge wire-fault leaked through IPC — "${msg}". Typed service errors are OK; JS runtime errors are not.`,
    ).toBe(false);
    return;
  }

  assertReturnShape(method, returnType, result);
}

function assertReturnShape(method: string, returnType: string, result: unknown): void {
  // `T | null` / `T | undefined` accepts null/undefined for free; the rest
  // of the check then runs against the unwrapped type.
  const nullable = /\|\s*(null|undefined)/.test(returnType);
  if (nullable && (result === null || result === undefined)) return;
  const rt = returnType
    .replace(/\s*\|\s*(null|undefined)/g, "")
    .trim();

  // `any` is the generator's fallback when no signature was discovered for
  // a handler. We only verified the bridge replied without a wire fault —
  // accept whatever payload shape it carries.
  if (rt === "any" || rt === "unknown") return;

  if (rt === "void" || rt === "()") {
    expect(
      result === null || result === undefined || result === "",
      `${method}: returnType=${returnType} expects null/undefined/empty, got ${JSON.stringify(result)}`,
    ).toBe(true);
    return;
  }

  if (rt === "boolean" || rt === "bool") {
    expect(
      typeof result,
      `${method}: returnType=${returnType} expects boolean, got ${typeof result}`,
    ).toBe("boolean");
    return;
  }

  if (rt === "string" || rt === "String") {
    expect(
      typeof result,
      `${method}: returnType=${returnType} expects string, got ${typeof result}`,
    ).toBe("string");
    expect(
      result,
      `${method}: bridge returned literal "undefined" string — handler likely threw silently`,
    ).not.toBe("undefined");
    return;
  }

  if (rt.startsWith("Option<")) {
    if (result === null || result === undefined) return;
    const inner = rt.replace(/^Option<(.+)>?$/, "$1");
    assertReturnShape(method, inner, result);
    return;
  }

  // Array<number> / Vec<u8> / Buffer / Uint8Array
  if (
    rt === "Buffer" ||
    rt === "Vec<u8>" ||
    rt === "Uint8Array" ||
    /^Array<\s*number\s*>$/.test(rt) ||
    /^Array<\s*u8\s*>$/.test(rt)
  ) {
    // A Uint8Array returned through page.evaluate() loses its type across the
    // CDP boundary and arrives as a numeric-keyed plain object ({"0":..,"1":..}).
    // Real ipcRenderer paths (structured clone) preserve it — this only bites
    // the probe. Accept that shape so non-empty binary returns validate.
    const isSerializedBytes =
      typeof result === "object" && result !== null && !Array.isArray(result) &&
      Object.keys(result as object).length > 0 &&
      Object.keys(result as object).every((k) => /^\d+$/.test(k));
    const ok =
      result instanceof Uint8Array ||
      (typeof result === "object" && result !== null && "byteLength" in (result as object)) ||
      Array.isArray(result) ||
      isSerializedBytes;
    expect(
      ok,
      `${method}: returnType=${returnType} expects byte container, got ${typeof result}`,
    ).toBe(true);
    return;
  }

  // Generic Array<X>
  if (/^Array</.test(rt)) {
    expect(
      Array.isArray(result),
      `${method}: returnType=${returnType} expects array, got ${typeof result}`,
    ).toBe(true);
    return;
  }

  if (rt === "number" || /^(u|i)\d+$/.test(rt) || rt === "usize" || rt === "isize") {
    expect(
      typeof result === "number" || typeof result === "string" || typeof result === "bigint",
      `${method}: returnType=${returnType} expects numeric, got ${typeof result}`,
    ).toBe(true);
    return;
  }

  // Fallback: any struct/object return must at minimum be non-null.
  expect(result, `${method}: returnType=${returnType} expects non-null value`).not.toBeNull();
}

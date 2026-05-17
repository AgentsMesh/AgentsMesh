import { wasmLogEvent } from "./wasm-core";

type Level = "trace" | "debug" | "info" | "warn" | "error";

declare global {
  interface Window {
    electronAPI?: {
      log?: (level: string, target: string, msg: string) => Promise<void>;
    };
  }
}

// Single fan-out point for renderer-side log emission. Routes (in priority
// order):
//   1. Electron IPC → main → Rust subscriber → rolling file (Desktop)
//   2. Direct wasm-bindgen call → Rust subscriber → console (Web)
//   3. Native console fallback when wasm hasn't initialised yet, so
//      early-boot logs don't disappear.
//
// All three branches end up funnelled through the same Rust subscriber
// (where one exists), so the log format, level filtering, and rotation
// behave identically across platforms.
function emit(level: Level, target: string, msg: string): void {
  const electronLog = typeof window !== "undefined" ? window.electronAPI?.log : undefined;
  if (electronLog) {
    void electronLog(level, target, msg);
    return;
  }
  try {
    wasmLogEvent(level, target, msg);
    return;
  } catch {
    // wasm not ready — fall through to console fallback below.
  }
  const formatted = `[${target}] ${msg}`;
  switch (level) {
    case "error":
      console.error(formatted);
      break;
    case "warn":
      console.warn(formatted);
      break;
    case "info":
      console.info(formatted);
      break;
    default:
      console.debug(formatted);
  }
}

export const logger = {
  trace: (target: string, msg: string) => emit("trace", target, msg),
  debug: (target: string, msg: string) => emit("debug", target, msg),
  info: (target: string, msg: string) => emit("info", target, msg),
  warn: (target: string, msg: string) => emit("warn", target, msg),
  error: (target: string, msg: string) => emit("error", target, msg),
};

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
//   2. Native console — on Web the wasm-side tracing subscriber renders
//      Rust events to console too, so the destination is the same.
// Why not also push to the wasm subscriber from here: doing so requires
// importing `wasm-core` (which depends on the `agentsmesh-wasm` package).
// Desktop bundles that package out via vite alias, so a renderer-shared
// dependency on it breaks the Desktop build. Keeping logger console-only
// avoids that coupling; Desktop gets file persistence via IPC, Web gets
// console output where the wasm subscriber also writes.
function emit(level: Level, target: string, msg: string): void {
  const electronLog = typeof window !== "undefined" ? window.electronAPI?.log : undefined;
  if (electronLog) {
    void electronLog(level, target, msg);
    return;
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

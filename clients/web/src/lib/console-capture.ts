import { logger } from "./logger";

// Mirrors console.warn / console.error (and optionally log/info under
// AGENTSMESH_CONSOLE_CAPTURE=verbose) to the cross-platform `logger`. The
// originals still fire so DevTools/console keep their normal output — we
// only fan out an additional copy to the Rust subscriber.
// PII surface: warn/error in this codebase are written defensively (status
// messages, error paths) — no opt-in needed. log/info commonly carry user
// data (channel messages, user objects, blockstore ops) so they're gated
// behind a verbose flag for ad-hoc debugging.
let installed = false;
let inLogger = false;

function safeStringify(arg: unknown): string {
  if (typeof arg === "string") return arg;
  if (arg instanceof Error) {
    return arg.stack ? `${arg.name}: ${arg.message}\n${arg.stack}` : `${arg.name}: ${arg.message}`;
  }
  try {
    return JSON.stringify(arg);
  } catch {
    return String(arg);
  }
}

function formatArgs(args: unknown[]): string {
  return args.map(safeStringify).join(" ");
}

function wrap(
  level: "info" | "warn" | "error",
  original: (...args: unknown[]) => void,
): (...args: unknown[]) => void {
  return function (this: unknown, ...args: unknown[]) {
    original.apply(console, args);
    // Guard the logger → console fallback path from re-entering capture
    // and forming a tight loop (logger.warn() → console.warn() → wrap()
    // → logger.warn() …). The flag is module-local; renderer JS is
    // single-threaded so a simple boolean is sufficient.
    if (inLogger) return;
    inLogger = true;
    try {
      logger[level]("console", formatArgs(args));
    } finally {
      inLogger = false;
    }
  };
}

export function installConsoleCapture(): void {
  if (installed) return;
  installed = true;
  const verbose = process.env.NEXT_PUBLIC_CONSOLE_CAPTURE === "verbose";
  const orig = {
    log: console.log.bind(console),
    info: console.info.bind(console),
    warn: console.warn.bind(console),
    error: console.error.bind(console),
  };
  console.warn = wrap("warn", orig.warn);
  console.error = wrap("error", orig.error);
  if (verbose) {
    console.log = wrap("info", orig.log);
    console.info = wrap("info", orig.info);
  }
}

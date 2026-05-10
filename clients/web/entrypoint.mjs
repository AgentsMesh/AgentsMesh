#!/usr/bin/env node
// Runtime env-var injection for the standalone Next.js bundle.
// Replaces the legacy `docker-entrypoint.sh` (sh + sed + find), which
// can't run on the distroless base. Walks `$APP_DIR/.next` and
// substitutes `__PLACEHOLDER__` tokens that `next.config.bazel.mjs` baked
// into the bundle, then execs server.js.
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { spawn } from "node:child_process";
import path from "node:path";

const appDir = process.env.APP_DIR || "/app/clients/web";
const subs = {
  "__PRIMARY_DOMAIN__": process.env.PRIMARY_DOMAIN || "",
  "__USE_HTTPS__": process.env.USE_HTTPS || "false",
  "__POSTHOG_KEY__": process.env.POSTHOG_KEY || "",
  "__POSTHOG_HOST__": process.env.POSTHOG_HOST || "",
};

function walk(dir) {
  let touched = 0;
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      touched += walk(full);
      continue;
    }
    if (!entry.isFile() || !full.endsWith(".js")) continue;
    let body = readFileSync(full, "utf8");
    let changed = false;
    for (const [k, v] of Object.entries(subs)) {
      if (body.includes(k)) {
        body = body.replaceAll(k, v);
        changed = true;
      }
    }
    if (changed) {
      writeFileSync(full, body);
      touched++;
    }
  }
  return touched;
}

try {
  const replaced = walk(path.join(appDir, ".next"));
  console.log(`[entrypoint] Replaced placeholders in ${replaced} file(s).`);
  for (const [k, v] of Object.entries(subs)) {
    const safe = k.endsWith("KEY__") || k.endsWith("SECRET__")
      ? (v ? "<set>" : "<empty>")
      : (v || "<empty>");
    console.log(`[entrypoint]   ${k} = ${safe}`);
  }
} catch (e) {
  console.error("[entrypoint] Substitution failed:", e.message);
}

const server = path.join(appDir, "server.js");
console.log(`[entrypoint] Starting ${server}`);
// Next.js standalone server.js binds to `process.env.HOSTNAME` if set,
// which Docker pre-populates with the container ID — yielding a
// listener on the wrong interface (the in-container probe at
// `127.0.0.1` cannot reach it). Force-bind to all interfaces unless
// the operator explicitly overrode HOSTNAME.
const childEnv = { ...process.env };
if (!childEnv.HOSTNAME_OVERRIDE) childEnv.HOSTNAME = "0.0.0.0";
const child = spawn(process.execPath, [server], {
  stdio: "inherit",
  env: childEnv,
});
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  else process.exit(code ?? 1);
});
process.on("SIGTERM", () => child.kill("SIGTERM"));
process.on("SIGINT", () => child.kill("SIGINT"));

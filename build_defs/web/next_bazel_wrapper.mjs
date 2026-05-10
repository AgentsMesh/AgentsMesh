// Vendored from aspect_rules_js v2.9.2 contrib/nextjs/next.bazel.mjs.
// We re-host it here because the upstream `nextjs_standalone_build` macro
// hard-codes `args = ["build"]` and gives no way to opt into `--webpack`.
// Next 16's default Turbopack pipeline crashes with "Symlink package.json
// is invalid, it points out of the filesystem root" inside the Bazel
// sandbox (cf. //build_defs/web:next.bzl `next_app` already routes
// around the same bug). So `next_standalone_app` (in next.bzl)
// reimplements the macro and references this wrapper directly.
//
// Behavior on `process.exit`:
//   Walk every `.next/standalone/<p>/node_modules` (where <p> climbs
//   from BAZEL_PACKAGE up to the workspace root). For every entry that
//   is an absolute symlink (NFT writes the pnpm virtual-store path as
//   `/private/var/.../sandbox/.../bazel-out/.../bin/node_modules/.aspect_rules_js/...`),
//   replace it with a recursive copy of the real target. Bazel's
//   declared-output validation refuses absolute symlinks, but real
//   files are fine.
//
//   This preserves NFT's traced dependency closure (next + its
//   transitives such as styled-jsx, @next/env, caniuse-lite, ...)
//   without forcing the BUILD file to enumerate every transitive in
//   `node_modules_runtime`.

import { existsSync, readdirSync, readlinkSync, realpathSync, rmSync, cpSync } from "node:fs";
import { join, dirname, isAbsolute } from "node:path";

const bazelPackage = process.env["BAZEL_PACKAGE"];
if (!process.cwd().endsWith(bazelPackage)) {
  throw new Error("This script must be run from Next.js app root");
}

let nextjsStandaloneConfig = process.env["NEXTJS_STANDALONE_CONFIG"];
if (nextjsStandaloneConfig.startsWith(process.env.BAZEL_BINDIR)) {
  nextjsStandaloneConfig = nextjsStandaloneConfig.slice(
    process.env.BAZEL_BINDIR.length + 1,
  );
}
nextjsStandaloneConfig = nextjsStandaloneConfig.replace(
  /^external\/[^/]+\//,
  "",
);

if (!nextjsStandaloneConfig.startsWith(bazelPackage)) {
  throw new Error(
    `Next.js config must be relative to the app root: ${nextjsStandaloneConfig}`,
  );
}

const NEXTJS_OUTDIR = ".next";

log(`Wrapping config: ${nextjsStandaloneConfig}`);
log(`Output dir: ${NEXTJS_OUTDIR}`);

let dereferencedCount = 0;
let removedCount = 0;

function dereferenceAbsoluteSymlinks(dir) {
  if (!existsSync(dir)) return;
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name);
    if (entry.isSymbolicLink()) {
      const target = readlinkSync(full);
      if (isAbsolute(target)) {
        try {
          const real = realpathSync(full);
          rmSync(full, { force: true });
          cpSync(real, full, { recursive: true, dereference: true });
          dereferencedCount++;
        } catch (e) {
          // Dangling symlink or permission issue — drop it so Bazel
          // can validate the output tree.
          try {
            rmSync(full, { force: true });
            removedCount++;
          } catch {
            /* ignore */
          }
        }
      }
      // Relative symlinks are kept intact: they resolve within the
      // standalone tree.
    } else if (entry.isDirectory()) {
      dereferenceAbsoluteSymlinks(full);
    }
  }
}

function nextjsFixSymlinks() {
  for (let p = process.env["BAZEL_PACKAGE"]; ; p = dirname(p)) {
    const d = join(NEXTJS_OUTDIR, "standalone", p, "node_modules");
    if (existsSync(d)) {
      dereferenceAbsoluteSymlinks(d);
    }
    if (p === "" || p === ".") break;
  }
  log(`Dereferenced ${dereferencedCount} absolute symlink(s); removed ${removedCount} dangling`);
}

function log(...args) {
  console.log(`[NextJs Bazel (${process.pid})]: `, ...args);
}

process.on("exit", nextjsFixSymlinks);

const nextjsStandaloneConfigRel = nextjsStandaloneConfig.slice(
  bazelPackage.length + 1,
);
const c = await import(`./${nextjsStandaloneConfigRel}`);
export default c.default || c;

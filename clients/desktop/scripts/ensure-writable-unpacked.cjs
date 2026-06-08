// Bazel marks build outputs read-only (0555); macOS removexattr — which
// Squirrel.Mac calls to strip com.apple.quarantine while installing an
// auto-update — needs write permission, so a read-only .node/.js aborts the
// install and the update silently rolls back. Restore owner-write on the
// unpacked native tree post-pack (electron-builder runs this before signing).
const { readdirSync, statSync, chmodSync, existsSync } = require("node:fs");
const { join } = require("node:path");

function ensureWritable(dir) {
  if (!existsSync(dir)) return;
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, entry.name);
    if (entry.isDirectory()) ensureWritable(p);
    chmodSync(p, statSync(p).mode | 0o200);
  }
}

function unpackedDir(context) {
  const { appOutDir, electronPlatformName, packager } = context;
  const resources =
    electronPlatformName === "darwin"
      ? join(appOutDir, `${packager.appInfo.productFilename}.app`, "Contents", "Resources")
      : join(appOutDir, "resources");
  return join(resources, "app.asar.unpacked");
}

exports.default = async function ensureWritableUnpacked(context) {
  ensureWritable(unpackedDir(context));
};

exports.ensureWritable = ensureWritable;
exports.unpackedDir = unpackedDir;

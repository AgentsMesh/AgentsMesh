// After release.yml staples the .app inside each zip and repacks it, the build-
// time sha512/size in latest-mac.yml no longer match — electron-updater verifies
// them and silently rejects the update, so rewrite them. Pure Node (js-yaml isn't
// require()-able from the repo root) over the line-stable format.
//
// Usage: node rewrite-latest-mac.mjs <latest-mac.yml> <stage-dir>
import { readFileSync, writeFileSync, statSync } from "node:fs";
import { createHash } from "node:crypto";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

export function rewriteLatestMac(ymlPath, stageDir) {
  // url is %-encoded; the file on disk is decoded. decodeURIComponent throws on a
  // literal '%', so fall back to raw (→ a clear ENOENT). Cache so the primary
  // artifact (also the top-level path:) isn't hashed twice.
  const hashCache = new Map();
  const decodeName = (name) => {
    try {
      return decodeURIComponent(name);
    } catch {
      return name;
    }
  };
  const filePath = (name) => join(stageDir, decodeName(name));
  const sha512 = (name) => {
    let h = hashCache.get(name);
    if (h === undefined) {
      h = createHash("sha512").update(readFileSync(filePath(name))).digest("base64");
      hashCache.set(name, h);
    }
    return h;
  };
  const sizeOf = (name) => statSync(filePath(name)).size;

  const lines = readFileSync(ymlPath, "utf8").split("\n");

  // Top-level `path:` names the artifact the top-level sha512 mirrors.
  let topPath = null;
  for (const l of lines) {
    const m = /^path:\s*(\S+)/.exec(l);
    if (m) topPath = m[1];
  }

  let curFile = null;
  const out = [];
  for (const line of lines) {
    const urlM = /^(\s*-\s*url:\s*)(\S+)/.exec(line);
    if (urlM) {
      curFile = urlM[2];
      out.push(line);
      continue;
    }
    const shaIndented = /^(\s+sha512:\s*).*/.exec(line);
    if (shaIndented && curFile) {
      out.push(`${shaIndented[1]}${sha512(curFile)}`);
      continue;
    }
    const sizeIndented = /^(\s+size:\s*).*/.exec(line);
    if (sizeIndented && curFile) {
      out.push(`${sizeIndented[1]}${sizeOf(curFile)}`);
      continue;
    }
    // The repack invalidates the blockmap; drop blockMapSize to force a full download.
    if (/^\s+blockMapSize:\s*/.test(line) && curFile) continue;
    const shaTop = /^(sha512:\s*).*/.exec(line);
    if (shaTop && topPath) {
      out.push(`${shaTop[1]}${sha512(topPath)}`);
      continue;
    }
    if (/^\S/.test(line)) curFile = null;
    out.push(line);
  }

  writeFileSync(ymlPath, out.join("\n"));
}

// CLI entry — release.yml runs `node rewrite-latest-mac.mjs <yml> <stage>`.
if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  const [, , ymlPath, stageDir] = process.argv;
  if (!ymlPath || !stageDir) {
    console.error("usage: rewrite-latest-mac.mjs <latest-mac.yml> <stage-dir>");
    process.exit(1);
  }
  rewriteLatestMac(ymlPath, stageDir);
  console.log("rewrote", ymlPath);
}

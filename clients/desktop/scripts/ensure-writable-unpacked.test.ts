import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, rmSync, mkdirSync, writeFileSync, chmodSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { ensureWritable, unpackedDir } from "./ensure-writable-unpacked.cjs";

const isWritable = (p: string) => (statSync(p).mode & 0o200) !== 0;

describe("ensureWritable", () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), "unpacked-"));
  });
  afterEach(() => {
    rmSync(dir, { recursive: true, force: true });
  });

  it("adds owner-write to a read-only file, preserving r/x bits", () => {
    const f = join(dir, "addon.node");
    writeFileSync(f, "x");
    chmodSync(f, 0o555);
    ensureWritable(dir);
    expect(isWritable(f)).toBe(true);
    expect(statSync(f).mode & 0o555).toBe(0o555);
  });

  it("recurses into nested read-only directories", () => {
    const sub = join(dir, "node_modules", "pkg");
    mkdirSync(sub, { recursive: true });
    const f = join(sub, "index.js");
    writeFileSync(f, "x");
    chmodSync(f, 0o555);
    ensureWritable(dir);
    expect(isWritable(f)).toBe(true);
  });

  it("is a no-op for a missing directory", () => {
    expect(() => ensureWritable(join(dir, "absent"))).not.toThrow();
  });
});

describe("unpackedDir", () => {
  const ctx = (electronPlatformName: string) => ({
    appOutDir: "/out",
    electronPlatformName,
    packager: { appInfo: { productFilename: "AgentsMesh" } },
  });

  it("resolves the macOS .app Resources path", () => {
    expect(unpackedDir(ctx("darwin"))).toBe("/out/AgentsMesh.app/Contents/Resources/app.asar.unpacked");
  });

  it("resolves the win/linux resources path", () => {
    expect(unpackedDir(ctx("win32"))).toBe("/out/resources/app.asar.unpacked");
  });
});

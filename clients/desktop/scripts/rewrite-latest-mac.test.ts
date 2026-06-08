import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, rmSync, writeFileSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createHash } from "node:crypto";
import { rewriteLatestMac } from "./rewrite-latest-mac.mjs";

const sha512 = (buf: Buffer) => createHash("sha512").update(buf).digest("base64");

describe("rewriteLatestMac", () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), "latestmac-"));
  });
  afterEach(() => {
    rmSync(dir, { recursive: true, force: true });
  });

  const write = (name: string, content: string) => writeFileSync(join(dir, name), content);

  it("rewrites sha512+size per zip, drops blockMapSize, mirrors the top-level sha512", () => {
    const arm = Buffer.from("ARM_ZIP_CONTENT");
    const x64 = Buffer.from("X64_ZIP_CONTENT_is_longer");
    write("AgentsMesh-1.0.0-arm64.zip", arm.toString());
    write("AgentsMesh-1.0.0.zip", x64.toString());
    write(
      "latest-mac.yml",
      [
        "version: 1.0.0",
        "files:",
        "  - url: AgentsMesh-1.0.0-arm64.zip",
        "    sha512: OLD1==",
        "    size: 1",
        "    blockMapSize: 999",
        "  - url: AgentsMesh-1.0.0.zip",
        "    sha512: OLD2==",
        "    size: 2",
        "path: AgentsMesh-1.0.0-arm64.zip",
        "sha512: OLD1==",
        "releaseDate: '2026-06-08T00:00:00.000Z'",
        "",
      ].join("\n"),
    );

    rewriteLatestMac(join(dir, "latest-mac.yml"), dir);
    const yml = readFileSync(join(dir, "latest-mac.yml"), "utf8");

    expect(yml).not.toContain("blockMapSize");
    expect(yml).toContain(`sha512: ${sha512(arm)}`);
    expect(yml).toContain(`sha512: ${sha512(x64)}`);
    expect(yml).toContain(`size: ${arm.length}`);
    expect(yml).toContain(`size: ${x64.length}`);
    // top-level sha512 (after path:) mirrors the arm64 artifact path points at
    const topSha = /^sha512:\s*(\S+)/m.exec(yml.split("path:")[1] ?? "")?.[1];
    expect(topSha).toBe(sha512(arm));
    // untouched fields preserved verbatim
    expect(yml).toContain("releaseDate: '2026-06-08T00:00:00.000Z'");
    expect(yml).toContain("version: 1.0.0");
  });

  it("rewrites files[] even when there is no top-level path:", () => {
    const z = Buffer.from("ZIP");
    write("a.zip", z.toString());
    write("latest-mac.yml", ["files:", "  - url: a.zip", "    sha512: OLD==", "    size: 1", ""].join("\n"));
    rewriteLatestMac(join(dir, "latest-mac.yml"), dir);
    const yml = readFileSync(join(dir, "latest-mac.yml"), "utf8");
    expect(yml).toContain(`sha512: ${sha512(z)}`);
    expect(yml).toContain(`size: ${z.length}`);
  });

  it("decodes %-encoded url to the on-disk filename", () => {
    const z = Buffer.from("SPACED");
    write("Agents Mesh-1.0.0.zip", z.toString());
    write(
      "latest-mac.yml",
      ["files:", "  - url: Agents%20Mesh-1.0.0.zip", "    sha512: OLD==", "    size: 1", ""].join("\n"),
    );
    rewriteLatestMac(join(dir, "latest-mac.yml"), dir);
    const yml = readFileSync(join(dir, "latest-mac.yml"), "utf8");
    expect(yml).toContain(`sha512: ${sha512(z)}`);
  });
});

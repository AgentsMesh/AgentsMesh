import { describe, it, expect } from "vitest";
import { readdirSync } from "node:fs";
import { resolve } from "node:path";
import { MESSAGE_NAMESPACES, locales } from "./config";

// Contract: `MESSAGE_NAMESPACES` drives both web (request.ts) and desktop
// (IntlProvider.tsx) loaders. Any namespace file on disk that isn't in the
// list stays dark — `t("<ns>.<key>")` falls back to raw key display.
// Regression: adding `blockstore.sidebar.deletePage` rendered as literal
// "blocks.sidebar.deletePage" in desktop because `blockstore` was missing
// from the desktop module list.

describe("i18n · namespace coverage", () => {
  const msgRoot = resolve(__dirname, "../../messages");

  it("MESSAGE_NAMESPACES lists every JSON file present in the default locale", () => {
    const onDisk = readdirSync(resolve(msgRoot, "en"))
      .filter((f) => f.endsWith(".json"))
      .map((f) => f.replace(/\.json$/, ""))
      .sort();

    const declared = [...MESSAGE_NAMESPACES].sort();
    const missing = onDisk.filter((ns) => !declared.includes(ns as typeof MESSAGE_NAMESPACES[number]));
    const stale = declared.filter((ns) => !onDisk.includes(ns));

    expect(missing, `namespace files on disk but not loaded: ${missing.join(", ")}`).toEqual([]);
    expect(stale, `namespaces declared but file missing: ${stale.join(", ")}`).toEqual([]);
  });

  it("every locale has the same namespace set as the default", () => {
    const reference = readdirSync(resolve(msgRoot, "en"))
      .filter((f) => f.endsWith(".json"))
      .sort();

    for (const locale of locales) {
      const actual = readdirSync(resolve(msgRoot, locale))
        .filter((f) => f.endsWith(".json"))
        .sort();
      expect(
        actual,
        `locale "${locale}" is missing namespaces: ${reference.filter((f) => !actual.includes(f)).join(", ")}`,
      ).toEqual(reference);
    }
  });
});

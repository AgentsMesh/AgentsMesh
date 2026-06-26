import { describe, it, expect } from "vitest";
import { readdirSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { locales } from "./config";

// namespaces.test.ts guards namespace *files* per locale; this guards the
// *keys* inside each file. After the 2026-06 i18n cleanup every locale's key
// set matches en exactly — this keeps it that way: a new key added to en but
// forgotten in a translation (or an orphan left in a locale) fails CI instead
// of silently rendering the raw key path / leaving dead translations.

function keyPaths(obj: unknown, prefix = ""): string[] {
  if (obj === null || typeof obj !== "object") return [prefix];
  return Object.entries(obj as Record<string, unknown>).flatMap(([k, v]) =>
    keyPaths(v, prefix ? `${prefix}.${k}` : k),
  );
}

const msgRoot = resolve(__dirname, "../../messages");

function loadKeys(locale: string, ns: string): Set<string> {
  return new Set(keyPaths(JSON.parse(readFileSync(resolve(msgRoot, locale, ns), "utf8"))));
}

describe("i18n · key parity across locales", () => {
  const namespaces = readdirSync(resolve(msgRoot, "en")).filter((f) => f.endsWith(".json"));

  for (const ns of namespaces) {
    const enKeys = loadKeys("en", ns);
    for (const locale of locales) {
      if (locale === "en") continue;
      it(`${locale}/${ns} has exactly the keys en has`, () => {
        const localeKeys = loadKeys(locale, ns);
        const missing = [...enKeys].filter((k) => !localeKeys.has(k));
        const extra = [...localeKeys].filter((k) => !enKeys.has(k));
        expect(missing, `keys in en but missing from ${locale}/${ns}`).toEqual([]);
        expect(extra, `orphan keys in ${locale}/${ns} not in en`).toEqual([]);
      });
    }
  }
});

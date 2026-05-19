export const locales = ["en", "zh", "ja", "ko", "es", "fr", "de", "pt"] as const;
export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = "en";

export const localeNames: Record<Locale, string> = {
  en: "English",
  zh: "简体中文",
  ja: "日本語",
  ko: "한국어",
  es: "Español",
  fr: "Français",
  de: "Deutsch",
  pt: "Português",
};

export const LOCALE_COOKIE = "NEXT_LOCALE";

// Single source of truth for i18n message namespaces.
// Both web (request.ts) and desktop (IntlProvider.tsx) loaders import this;
// forgetting to add a namespace here is the only way to cause "raw key shows
// instead of translation" bugs across platforms.
export const MESSAGE_NAMESPACES = [
  "common", "auth", "landing", "app", "settings", "ide",
  "repositories", "runners", "docs", "content", "extensions",
  "loops", "channels", "blockstore", "infra",
] as const;

export function isValidLocale(locale: string): locale is Locale {
  return locales.includes(locale as Locale);
}

export function getLocaleFromHeaders(acceptLanguage: string | null): Locale {
  if (!acceptLanguage) return defaultLocale;

  const languages = acceptLanguage
    .split(",")
    .map((lang) => {
      const [code, qValue] = lang.trim().split(";q=");
      return {
        code: code.toLowerCase().split("-")[0],
        quality: qValue ? parseFloat(qValue) : 1,
      };
    })
    .sort((a, b) => b.quality - a.quality);

  for (const { code } of languages) {
    if (isValidLocale(code)) {
      return code;
    }
  }

  return defaultLocale;
}

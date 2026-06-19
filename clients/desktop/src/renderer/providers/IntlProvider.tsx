import { useEffect, useState, type ReactNode } from "react";
import { IntlProvider as ReactIntlProvider } from "next-intl";
import { type Locale, defaultLocale, isValidLocale, MESSAGE_NAMESPACES } from "@/lib/i18n/config";
import { deepMergeMessages } from "@/lib/i18n/messageFallback";

export const LOCALE_STORAGE_KEY = "app_locale";
export const LOCALE_CHANGE_EVENT = "app-locale-change";

function detectSystemLocale(): Locale {
  const lang = navigator.language.split("-")[0];
  return isValidLocale(lang) ? lang : defaultLocale;
}

function getSavedLocale(): Locale {
  const saved = localStorage.getItem(LOCALE_STORAGE_KEY);
  if (saved && isValidLocale(saved)) return saved as Locale;
  return detectSystemLocale();
}

async function loadLocaleMessages(locale: Locale): Promise<Record<string, unknown>> {
  const files = await Promise.all(
    MESSAGE_NAMESPACES.map((m) => import(`@/messages/${locale}/${m}.json`).catch(() => ({ default: {} })))
  );
  return Object.assign({}, ...files.map((f) => f.default));
}

// Non-default locales fall back to the (complete) default-locale messages for
// any key they omit. Web's non-en/zh locales lag the en baseline; without this
// every missing key renders as a raw key-path on desktop (unlike web/request.ts
// which already deep-merges). Mirrors that fallback.
// en base is invariant across locale switches in this long-lived renderer —
// load it once and reuse instead of re-importing all namespaces each switch.
let enBasePromise: Promise<Record<string, unknown>> | null = null;

async function loadMessages(locale: Locale): Promise<Record<string, unknown>> {
  const localeMessages = await loadLocaleMessages(locale);
  if (locale === defaultLocale) return localeMessages;
  enBasePromise ??= loadLocaleMessages(defaultLocale);
  return deepMergeMessages(await enBasePromise, localeMessages);
}

export function DesktopIntlProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(getSavedLocale);
  const [messages, setMessages] = useState<Record<string, unknown> | null>(null);

  useEffect(() => {
    loadMessages(locale).then(setMessages);
  }, [locale]);

  useEffect(() => {
    const handler = (e: Event) => {
      const next = (e as CustomEvent<Locale>).detail;
      if (isValidLocale(next)) setLocaleState(next);
    };
    window.addEventListener(LOCALE_CHANGE_EVENT, handler);
    return () => window.removeEventListener(LOCALE_CHANGE_EVENT, handler);
  }, []);

  if (!messages) return null;

  return (
    <ReactIntlProvider locale={locale} messages={messages}>
      {children}
    </ReactIntlProvider>
  );
}

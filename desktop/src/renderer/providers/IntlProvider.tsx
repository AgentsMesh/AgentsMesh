import React, { createContext, useContext, useEffect, useState, useCallback, useMemo } from "react";
import { IntlProvider as ReactIntlProvider } from "next-intl";
import { type Locale, defaultLocale, locales, isValidLocale, MESSAGE_NAMESPACES } from "@/lib/i18n/config";

const LOCALE_STORAGE_KEY = "app_locale";

interface IntlContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
}

const IntlContext = createContext<IntlContextValue | undefined>(undefined);

function detectSystemLocale(): Locale {
  const lang = navigator.language.split("-")[0];
  return isValidLocale(lang) ? lang : defaultLocale;
}

function getSavedLocale(): Locale {
  const saved = localStorage.getItem(LOCALE_STORAGE_KEY);
  if (saved && isValidLocale(saved)) return saved as Locale;
  return detectSystemLocale();
}

async function loadMessages(locale: Locale): Promise<Record<string, unknown>> {
  // Namespace list lives in @/lib/i18n/config — web and desktop share the
  // same source to prevent "raw key shows instead of translation" drift.
  const files = await Promise.all(
    MESSAGE_NAMESPACES.map((m) => import(`@/messages/${locale}/${m}.json`).catch(() => ({ default: {} })))
  );
  return Object.assign({}, ...files.map((f) => f.default));
}

export function DesktopIntlProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(getSavedLocale);
  const [messages, setMessages] = useState<Record<string, unknown> | null>(null);

  const setLocale = useCallback((newLocale: Locale) => {
    localStorage.setItem(LOCALE_STORAGE_KEY, newLocale);
    setLocaleState(newLocale);
  }, []);

  useEffect(() => {
    loadMessages(locale).then(setMessages);
  }, [locale]);

  const contextValue = useMemo(() => ({ locale, setLocale }), [locale, setLocale]);

  if (!messages) return null;

  return (
    <IntlContext.Provider value={contextValue}>
      <ReactIntlProvider locale={locale} messages={messages}>
        {children}
      </ReactIntlProvider>
    </IntlContext.Provider>
  );
}

export function useLocale() {
  const context = useContext(IntlContext);
  if (!context) throw new Error("useLocale must be used within DesktopIntlProvider");
  return context;
}

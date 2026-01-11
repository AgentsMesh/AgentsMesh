"use client";

import { createContext, useContext, useState, useCallback, ReactNode } from "react";
import { Locale, defaultLocale, locales, LOCALE_COOKIE } from "./config";

// Type for nested translation object
type TranslationValue = string | string[] | { [key: string]: TranslationValue };
type Translations = { [key: string]: TranslationValue };

// I18n context type
interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string, params?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nContextType | null>(null);

// Get nested value from object using dot notation
function getNestedValue(obj: Translations, path: string): string | undefined {
  const keys = path.split(".");
  let current: TranslationValue | undefined = obj;

  for (const key of keys) {
    if (current && typeof current === "object" && !Array.isArray(current) && key in current) {
      current = current[key];
    } else {
      return undefined;
    }
  }

  return typeof current === "string" ? current : undefined;
}

// Replace parameters in translation string
function interpolate(
  str: string,
  params?: Record<string, string | number>
): string {
  if (!params) return str;

  return str.replace(/\{(\w+)\}/g, (_, key) => {
    return params[key]?.toString() ?? `{${key}}`;
  });
}

// Import translations dynamically
const translationsCache: Partial<Record<Locale, Translations>> = {};

async function loadTranslations(locale: Locale): Promise<Translations> {
  if (translationsCache[locale]) {
    return translationsCache[locale]!;
  }

  try {
    const translations = await import(`@/messages/${locale}.json`);
    translationsCache[locale] = translations.default || translations;
    return translationsCache[locale]!;
  } catch {
    console.error(`Failed to load translations for ${locale}`);
    return {};
  }
}

interface I18nProviderProps {
  children: ReactNode;
  initialLocale?: Locale;
  initialTranslations?: Translations;
}

export function I18nProvider({
  children,
  initialLocale = defaultLocale,
  initialTranslations = {},
}: I18nProviderProps) {
  const [locale, setLocaleState] = useState<Locale>(initialLocale);
  const [translations, setTranslations] =
    useState<Translations>(initialTranslations);

  // Set locale and load translations
  const setLocale = useCallback(async (newLocale: Locale) => {
    if (!locales.includes(newLocale)) {
      console.warn(`Invalid locale: ${newLocale}`);
      return;
    }

    // Load translations
    const newTranslations = await loadTranslations(newLocale);
    setTranslations(newTranslations);
    setLocaleState(newLocale);

    // Save to cookie
    document.cookie = `${LOCALE_COOKIE}=${newLocale}; path=/; max-age=${60 * 60 * 24 * 365}`;

    // Update HTML lang attribute
    document.documentElement.lang = newLocale;
  }, []);

  // Translation function
  const t = useCallback(
    (key: string, params?: Record<string, string | number>): string => {
      const value = getNestedValue(translations, key);
      if (value === undefined) {
        console.warn(`Translation missing for key: ${key}`);
        return key;
      }
      return interpolate(value, params);
    },
    [translations]
  );

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

// Hook to use i18n
export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within an I18nProvider");
  }
  return context;
}

// Hook for just translations
export function useTranslations() {
  const { t } = useI18n();
  return t;
}

// Hook for locale
export function useLocale() {
  const { locale, setLocale } = useI18n();
  return { locale, setLocale };
}

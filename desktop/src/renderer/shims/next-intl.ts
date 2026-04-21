import React, { createContext, useContext, useCallback, useMemo } from "react";
import { defaultLocale, type Locale } from "@/lib/i18n/config";

type Messages = Record<string, unknown>;

interface IntlContextValue {
  locale: Locale;
  messages: Messages;
}

const IntlContext = createContext<IntlContextValue>({
  locale: defaultLocale,
  messages: {},
});

function getNestedRaw(obj: unknown, path: string): unknown {
  const keys = path.split(".");
  let current: unknown = obj;
  for (const key of keys) {
    if (current == null || typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[key];
  }
  return current;
}

function getNestedString(obj: unknown, path: string): string | undefined {
  const v = getNestedRaw(obj, path);
  return typeof v === "string" ? v : undefined;
}

function interpolate(template: string, values?: Record<string, unknown>): string {
  if (!values) return template;
  let result = template.replace(
    /\{(\w+),\s*plural,\s*(?:one\s*\{([^}]*)\}\s*)?other\s*\{([^}]*)\}\s*\}/g,
    (_, key, oneForm, otherForm) => {
      const count = Number(values[key] ?? 0);
      const form = (count === 1 && oneForm) ? oneForm : otherForm;
      return form.replace(/#/g, String(count));
    },
  );
  result = result.replace(/\{(\w+)\}/g, (_, key) => String(values[key] ?? `{${key}}`));
  return result;
}

type TranslateFunction = {
  (key: string, values?: Record<string, unknown>): string;
  rich: (key: string, values?: Record<string, unknown>) => string;
  raw: (key: string) => string;
  has: (key: string) => boolean;
};

export function useTranslations(namespace?: string): TranslateFunction {
  const { messages } = useContext(IntlContext);

  const scopedMessages = useMemo(() => {
    if (!namespace) return messages;
    const scoped = getNestedRaw(messages, namespace);
    return (scoped && typeof scoped === "object" ? scoped : {}) as Messages;
  }, [messages, namespace]);

  const t = useCallback(
    (key: string, values?: Record<string, unknown>): string => {
      const value = getNestedString(scopedMessages, key);
      if (value === undefined) {
        return namespace ? `${namespace}.${key}` : key;
      }
      return interpolate(value, values);
    },
    [scopedMessages, namespace],
  ) as TranslateFunction;

  t.rich = t;
  t.raw = (key: string) => getNestedString(scopedMessages, key) ?? key;
  t.has = (key: string) => getNestedString(scopedMessages, key) !== undefined;

  return t;
}

export function useLocale(): Locale {
  const { locale } = useContext(IntlContext);
  return locale;
}

export function useMessages(): Messages {
  const { messages } = useContext(IntlContext);
  return messages;
}

interface IntlProviderProps {
  locale: Locale;
  messages: Messages;
  children: React.ReactNode;
}

export function NextIntlClientProvider({ locale, messages, children }: IntlProviderProps) {
  const value = useMemo(() => ({ locale, messages }), [locale, messages]);
  return React.createElement(IntlContext.Provider, { value }, children);
}

export const IntlProvider = NextIntlClientProvider;

export { IntlContext };

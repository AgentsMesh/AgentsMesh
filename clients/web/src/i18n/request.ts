import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";
import {
  LOCALE_COOKIE,
  MESSAGE_NAMESPACES,
  defaultLocale,
  isValidLocale,
  getLocaleFromHeaders,
} from "@/lib/i18n/config";
import { deepMergeMessages } from "@/lib/i18n/messageFallback";

async function loadLocale(locale: string) {
  const files = await Promise.all(
    MESSAGE_NAMESPACES.map((ns) => import(`@/messages/${locale}/${ns}.json`)),
  );
  return Object.assign({}, ...files.map((f) => f.default));
}

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const localeCookie = cookieStore.get(LOCALE_COOKIE);
  let locale = defaultLocale;

  if (localeCookie && isValidLocale(localeCookie.value)) {
    locale = localeCookie.value;
  } else {
    const headersList = await headers();
    locale = getLocaleFromHeaders(headersList.get("accept-language"));
  }

  // en fills any key the active locale omits (no per-locale key-path holes).
  const localeMessages = await loadLocale(locale);
  const messages =
    locale === defaultLocale
      ? localeMessages
      : deepMergeMessages(await loadLocale(defaultLocale), localeMessages);
  return { locale, messages };
});

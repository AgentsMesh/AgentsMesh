import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";
import {
  LOCALE_COOKIE,
  MESSAGE_NAMESPACES,
  defaultLocale,
  isValidLocale,
  getLocaleFromHeaders,
} from "@/lib/i18n/config";

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

  const files = await Promise.all(
    MESSAGE_NAMESPACES.map((ns) => import(`@/messages/${locale}/${ns}.json`)),
  );

  const messages = Object.assign({}, ...files.map((f) => f.default));
  return { locale, messages };
});

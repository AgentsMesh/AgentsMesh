import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";
import {
  LOCALE_COOKIE,
  defaultLocale,
  isValidLocale,
  getLocaleFromHeaders,
} from "@/lib/i18n/config";

export default getRequestConfig(async () => {
  // Locale detection: cookie → Accept-Language → default
  const cookieStore = await cookies();
  const localeCookie = cookieStore.get(LOCALE_COOKIE);
  let locale = defaultLocale;

  if (localeCookie && isValidLocale(localeCookie.value)) {
    locale = localeCookie.value;
  } else {
    const headersList = await headers();
    locale = getLocaleFromHeaders(headersList.get("accept-language"));
  }

  const files = await Promise.all([
    import(`@/messages/${locale}/common.json`),
    import(`@/messages/${locale}/auth.json`),
    import(`@/messages/${locale}/landing.json`),
    import(`@/messages/${locale}/app.json`),
    import(`@/messages/${locale}/settings.json`),
    import(`@/messages/${locale}/ide.json`),
    import(`@/messages/${locale}/repositories.json`),
    import(`@/messages/${locale}/runners.json`),
    import(`@/messages/${locale}/docs.json`),
    import(`@/messages/${locale}/content.json`),
    import(`@/messages/${locale}/extensions.json`),
  ]);

  const messages = Object.assign({}, ...files.map((f) => f.default));
  return { locale, messages };
});

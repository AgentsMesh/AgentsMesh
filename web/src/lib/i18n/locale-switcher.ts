"use client";

import { useRouter } from "next/navigation";
import { useCallback } from "react";
import { Locale, LOCALE_COOKIE, locales } from "./config";

export function useSetLocale() {
  const router = useRouter();
  return useCallback(
    (newLocale: Locale) => {
      if (!locales.includes(newLocale)) return;
      document.cookie = `${LOCALE_COOKIE}=${newLocale}; path=/; max-age=${60 * 60 * 24 * 365}`;
      document.documentElement.lang = newLocale;
      router.refresh();
    },
    [router]
  );
}

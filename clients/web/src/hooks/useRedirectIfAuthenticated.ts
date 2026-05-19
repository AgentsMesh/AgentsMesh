"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useLightSession } from "@/hooks/useLightSession";
import { getDefaultRoute } from "@/lib/default-route";
import { fetchFirstOrgSlug } from "@/lib/light-auth";

export function useRedirectIfAuthenticated(): { hydrated: boolean; redirecting: boolean } {
  const router = useRouter();
  const { session, hydrated } = useLightSession();
  const shouldRedirect = hydrated && !!session?.isAuthenticated;

  useEffect(() => {
    if (!shouldRedirect) return;
    let cancelled = false;
    (async () => {
      const slug = session?.currentOrgSlug || (await fetchFirstOrgSlug());
      if (cancelled) return;
      router.replace(slug ? getDefaultRoute(slug) : "/onboarding");
    })();
    return () => { cancelled = true; };
  }, [shouldRedirect, session?.currentOrgSlug, router]);

  return { hydrated, redirecting: shouldRedirect };
}

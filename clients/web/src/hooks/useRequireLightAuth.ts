"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useLightSession } from "@/hooks/useLightSession";
import { loginUrlWithRedirect } from "@/lib/auth/redirect";

export function useRequireLightAuth(): { hydrated: boolean; authenticated: boolean } {
  const router = useRouter();
  const { session, hydrated } = useLightSession();
  const authenticated = !!session?.isAuthenticated;

  useEffect(() => {
    if (!hydrated) return;
    if (authenticated) return;
    const here = typeof window !== "undefined"
      ? window.location.pathname + window.location.search
      : null;
    router.replace(loginUrlWithRedirect(here ?? "/"));
  }, [hydrated, authenticated, router]);

  return { hydrated, authenticated };
}

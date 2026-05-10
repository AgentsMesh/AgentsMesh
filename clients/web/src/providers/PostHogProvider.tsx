"use client";

import posthog from "posthog-js";
import { PostHogProvider as PHProvider, usePostHog } from "posthog-js/react";
import { Suspense, useEffect } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import { useCurrentUser, useCurrentOrg } from "@/stores/auth";

// Filter out unresolved entrypoint.mjs placeholders (e.g. "__POSTHOG_KEY__")
function resolveEnv(val: string | undefined): string {
  if (!val || val.startsWith("__")) return "";
  return val;
}

const POSTHOG_KEY = resolveEnv(process.env.NEXT_PUBLIC_POSTHOG_KEY);
const POSTHOG_HOST = resolveEnv(process.env.NEXT_PUBLIC_POSTHOG_HOST);

if (typeof window !== "undefined" && POSTHOG_KEY) {
  posthog.init(POSTHOG_KEY, {
    api_host: POSTHOG_HOST,
    capture_pageview: false, // We capture manually below
    capture_pageleave: true,
    persistence: "localStorage+cookie",
    // Disable remote config / feature flags / surveys
    // (self-hosted instance does not serve /flags or /config endpoints)
    advanced_disable_decide: true,
  });
}

function PostHogPageView() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const ph = usePostHog();

  useEffect(() => {
    if (pathname && ph) {
      let url = window.origin + pathname;
      if (searchParams?.toString()) {
        url += "?" + searchParams.toString();
      }
      ph.capture("$pageview", { $current_url: url });
    }
  }, [pathname, searchParams, ph]);

  return null;
}

// User/org identification — uses wasm hooks, so MUST be mounted only inside
// (dashboard) / (auth) where wasm is loaded. Marketing pages must not import.
export function PostHogIdentify() {
  const ph = usePostHog();
  const user = useCurrentUser();
  const currentOrg = useCurrentOrg();

  useEffect(() => {
    if (!ph) return;

    if (user) {
      ph.identify(String(user.id), {
        email: user.email,
        username: user.username,
        name: user.name,
      });
    } else {
      ph.reset();
    }
  }, [ph, user]);

  useEffect(() => {
    if (!ph || !currentOrg) return;

    ph.group("organization", String(currentOrg.id), {
      name: currentOrg.name,
      slug: currentOrg.slug,
      subscription_plan: currentOrg.subscription_plan,
    });
  }, [ph, currentOrg]);

  return null;
}

// Root provider — pageview only. Identify is mounted separately inside
// authenticated route groups so marketing pages don't pull in wasm.
export function PostHogProvider({ children }: { children: React.ReactNode }) {
  return (
    <PHProvider client={posthog}>
      {POSTHOG_KEY && (
        <Suspense fallback={null}>
          <PostHogPageView />
        </Suspense>
      )}
      {children}
    </PHProvider>
  );
}

"use client";

import posthog from "posthog-js";
import { PostHogProvider as PHProvider, usePostHog } from "posthog-js/react";
import { Suspense, useEffect } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import { useAuthStore } from "@/stores/auth";

const POSTHOG_KEY = "phc_MA31kFOuqaLC9BaTOH6sL7BAXw7uPrfzDanUtXKOqIr";
const POSTHOG_HOST = "https://posthog.monitor.agentsmesh.ai";

if (typeof window !== "undefined") {
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

/**
 * Captures page views on route changes (Next.js App Router)
 */
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

/**
 * Identifies the current user and organization in PostHog
 */
function PostHogIdentify() {
  const ph = usePostHog();
  const { user, currentOrg } = useAuthStore();

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

export function PostHogProvider({ children }: { children: React.ReactNode }) {
  return (
    <PHProvider client={posthog}>
      <Suspense fallback={null}>
        <PostHogPageView />
      </Suspense>
      <PostHogIdentify />
      {children}
    </PHProvider>
  );
}

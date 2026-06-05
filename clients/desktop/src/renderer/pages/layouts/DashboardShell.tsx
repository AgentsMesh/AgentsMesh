import React, { useEffect, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  useAuthStore,
  useAuthOrganizations,
  useCurrentOrg,
} from "@/stores/auth";
import { ResponsiveShell } from "@/components/layout";
import { Spinner } from "@/components/ui/spinner";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { useBrowserNotification, useSessionKeepAlive } from "@/hooks";
import { handleNotificationEvent } from "@/stores/notificationHandler";
import type { RealtimeEvent } from "@/lib/realtime";

export function DashboardShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const params = useParams<{ org?: string }>();
  const _hasHydrated = useAuthStore((s) => s._hasHydrated);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setCurrentOrg = useAuthStore((s) => s.setCurrentOrg);
  const organizations = useAuthOrganizations();
  const currentOrg = useCurrentOrg();
  const { permission, showNotification, requestPermission } = useBrowserNotification();
  useSessionKeepAlive();

  // Evaluate per-render so token expiry (a time-driven flip with no store
  // event) becomes an effect-dependency change. Calling isAuthenticated
  // directly in the effect dep list keeps a stable fn ref, so the redirect
  // would never re-run and the shell would sit blank on an expired session.
  const authed = isAuthenticated();

  useEffect(() => {
    if (_hasHydrated && !authed) {
      router.push("/login");
    }
  }, [_hasHydrated, authed, router]);

  useEffect(() => {
    const orgSlug = params.org;
    if (!_hasHydrated || !isAuthenticated() || !orgSlug) return;
    if (currentOrg?.slug === orgSlug) return;

    const match = organizations.find((o) => o.slug === orgSlug);
    if (match) setCurrentOrg(match);
  }, [params.org, _hasHydrated, currentOrg?.slug, organizations, setCurrentOrg, isAuthenticated]);

  useEffect(() => {
    if (_hasHydrated && isAuthenticated() && permission === "default") {
      const timer = setTimeout(() => { requestPermission(); }, 3000);
      return () => clearTimeout(timer);
    }
  }, [_hasHydrated, isAuthenticated, permission, requestPermission]);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      handleNotificationEvent(event, {
        router,
        showBrowserNotification: (data) => {
          showNotification({
            title: data.title,
            body: data.body,
            tag: `notif-${data.link || data.title}`,
            onClick: () => {
              if (data.link && currentOrg?.slug) {
                router.push(`/${currentOrg.slug}${data.link}`);
              }
            },
          });
        },
      });
    },
    [showNotification, router, currentOrg]
  );

  // Not-hydrated and expired-session both park on a spinner; the redirect
  // effect above carries an expired session on to /login.
  if (!_hasHydrated || !authed) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <Spinner />
      </div>
    );
  }

  return (
    <RealtimeProvider onEvent={handleEvent}>
      <ResponsiveShell>{children}</ResponsiveShell>
    </RealtimeProvider>
  );
}

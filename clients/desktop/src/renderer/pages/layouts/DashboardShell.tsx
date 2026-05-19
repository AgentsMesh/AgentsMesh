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
import { useBrowserNotification } from "@/hooks";
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

  useEffect(() => {
    if (_hasHydrated && !isAuthenticated()) {
      router.push("/login");
    }
  }, [_hasHydrated, isAuthenticated, router]);

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

  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <Spinner />
      </div>
    );
  }

  if (!isAuthenticated()) {
    return null;
  }

  return (
    <RealtimeProvider onEvent={handleEvent}>
      <ResponsiveShell>{children}</ResponsiveShell>
    </RealtimeProvider>
  );
}

"use client";

import React, { useEffect, useCallback, useState } from "react";
import { useRouter } from "next/navigation";
import { useCurrentUser, useCurrentOrg, useAuthStore } from "@/stores/auth";
import { ResponsiveShell } from "@/components/layout";
import { Spinner } from "@/components/ui/spinner";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { initWasmCore, getAuthManager, getApiClient } from "@/lib/wasm-core";
import { useBrowserNotification } from "@/hooks";
import { handleNotificationEvent } from "@/stores/notificationHandler";
import type { RealtimeEvent } from "@/lib/realtime";

export default function DashboardShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const [wasmReady, setWasmReady] = useState(false);
  const user = useCurrentUser();
  const currentOrg = useCurrentOrg();
  const { permission, showNotification, requestPermission } = useBrowserNotification();

  useEffect(() => {
    initWasmCore().then(async () => {
      const mgr = getAuthManager();
      const restored = await mgr.restore_session();
      if (restored) {
        const token = mgr.get_token();
        const refreshToken = mgr.get_refresh_token();
        if (token) getApiClient().set_token(token, refreshToken || "");
        const userJson = mgr.get_current_user_json();
        if (userJson) {
          const user = typeof userJson === "string" ? JSON.parse(userJson) : userJson;
          useAuthStore.getState().setAuth(token || "", user, refreshToken || "");
          let orgs = JSON.parse(mgr.get_organizations_json() || "[]");
          if (orgs.length === 0) {
            try {
              const fetchedJson = await mgr.fetch_organizations();
              orgs = JSON.parse(fetchedJson || "[]");
            } catch { /* token may be expired */ }
          }
          if (orgs.length > 0) {
            const urlSlug = window.location.pathname.split("/")[1];
            const matchedOrg = orgs.find((o: { slug: string }) => o.slug === urlSlug);
            const org = matchedOrg || orgs[0];
            getApiClient().set_org_slug(org.slug);
            mgr.switch_org(org.slug);
            useAuthStore.getState().setOrganizations(orgs);
            useAuthStore.getState().setCurrentOrg(org);
          }
        }
      }
      useAuthStore.getState().setHasHydrated(true);
      setWasmReady(true);
    });
  }, []);

  useEffect(() => {
    if (wasmReady && !user) {
      router.push("/login");
    }
  }, [user, router, wasmReady]);

  useEffect(() => {
    if (wasmReady && user && permission === "default") {
      const timer = setTimeout(() => { requestPermission(); }, 3000);
      return () => clearTimeout(timer);
    }
  }, [wasmReady, user, permission, requestPermission]);

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

  if (!wasmReady) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <Spinner />
      </div>
    );
  }

  if (!user) {
    return null;
  }

  return (
    <RealtimeProvider onEvent={handleEvent}>
      <ResponsiveShell>{children}</ResponsiveShell>
    </RealtimeProvider>
  );
}

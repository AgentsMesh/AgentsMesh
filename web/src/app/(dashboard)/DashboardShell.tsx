"use client";

import React, { useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { ResponsiveShell } from "@/components/layout";
import { Spinner } from "@/components/ui/spinner";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { useBrowserNotification } from "@/hooks";
import type { TerminalNotificationData, TaskCompletedData } from "@/lib/realtime";

export default function DashboardShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const { token, currentOrg, _hasHydrated } = useAuthStore();
  const { permission, showNotification, requestPermission } = useBrowserNotification();

  useEffect(() => {
    // Only redirect after hydration is complete
    if (_hasHydrated && !token) {
      router.push("/login");
    }
  }, [token, router, _hasHydrated]);

  // Request notification permission on first load (if not yet requested)
  useEffect(() => {
    if (_hasHydrated && token && permission === "default") {
      // Auto-request permission after a short delay to not be intrusive
      const timer = setTimeout(() => {
        requestPermission();
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [_hasHydrated, token, permission, requestPermission]);

  // Navigate to workspace with specific pod
  const navigateToWorkspace = useCallback(
    (podKey: string) => {
      if (currentOrg?.slug) {
        router.push(`/${currentOrg.slug}/workspace?pod=${podKey}`);
      }
    },
    [router, currentOrg]
  );

  // Handle terminal notifications (OSC 777)
  const handleTerminalNotification = useCallback(
    (data: TerminalNotificationData) => {
      // Show browser notification for terminal events
      showNotification({
        title: data.title,
        body: data.body,
        tag: `terminal-${data.pod_key}`,
        data: { podKey: data.pod_key },
        onClick: () => navigateToWorkspace(data.pod_key),
      });
      console.log("[Notification] Terminal:", data.title, data.body);
    },
    [showNotification, navigateToWorkspace]
  );

  // Handle task completed notifications
  const handleTaskCompleted = useCallback(
    (data: TaskCompletedData) => {
      const podKeyShort = data.pod_key.substring(0, 8);
      // Show browser notification for task completion
      showNotification({
        title: "Task Completed",
        body: `Pod ${podKeyShort}... finished (${data.agent_status})`,
        tag: `task-${data.pod_key}`,
        data: { podKey: data.pod_key },
        onClick: () => navigateToWorkspace(data.pod_key),
      });
      console.log("[Notification] Task completed:", data.pod_key, data.agent_status);
    },
    [showNotification, navigateToWorkspace]
  );

  // Handle unified browser notifications (from NotificationDispatcher)
  const handleBrowserNotification = useCallback(
    (data: { title: string; body: string; link?: string }) => {
      showNotification({
        title: data.title,
        body: data.body,
        tag: `notif-${Date.now()}`,
        onClick: () => {
          if (data.link && currentOrg?.slug) {
            router.push(`/${currentOrg.slug}${data.link}`);
          }
        },
      });
    },
    [showNotification, router, currentOrg]
  );

  // Show loading state while hydrating
  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <Spinner />
      </div>
    );
  }

  if (!token) {
    return null;
  }

  return (
    <RealtimeProvider
      onTerminalNotification={handleTerminalNotification}
      onTaskCompleted={handleTaskCompleted}
      onBrowserNotification={handleBrowserNotification}
    >
      <ResponsiveShell>{children}</ResponsiveShell>
    </RealtimeProvider>
  );
}

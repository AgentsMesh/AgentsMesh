"use client";

import React, { useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuthStore } from "@/stores/auth";
import { ResponsiveShell } from "@/components/layout";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import type { TerminalNotificationData, TaskCompletedData } from "@/lib/realtime";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const { token, _hasHydrated } = useAuthStore();

  useEffect(() => {
    // Only redirect after hydration is complete
    if (_hasHydrated && !token) {
      router.push("/login");
    }
  }, [token, router, _hasHydrated]);

  // Show loading state while hydrating
  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  // Handle terminal notifications (OSC 777)
  const handleTerminalNotification = useCallback(
    (data: TerminalNotificationData) => {
      // Show toast notification for terminal events
      toast.info(data.title, {
        description: data.body,
        duration: 5000,
      });
      console.log("[Notification] Terminal:", data.title, data.body);
    },
    []
  );

  // Handle task completed notifications
  const handleTaskCompleted = useCallback((data: TaskCompletedData) => {
    // Show toast notification for task completion
    toast.success("Task Completed", {
      description: `Pod ${data.pod_key.substring(0, 12)}... finished (${data.agent_status})`,
      duration: 5000,
    });
    console.log("[Notification] Task completed:", data.pod_key, data.agent_status);
  }, []);

  if (!token) {
    return null;
  }

  return (
    <RealtimeProvider
      onTerminalNotification={handleTerminalNotification}
      onTaskCompleted={handleTaskCompleted}
    >
      <ResponsiveShell>{children}</ResponsiveShell>
    </RealtimeProvider>
  );
}

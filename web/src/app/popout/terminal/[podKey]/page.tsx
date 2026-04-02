"use client";

import React, { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { TerminalPane } from "@/components/workspace/TerminalPane";
import { Spinner } from "@/components/ui/spinner";

/**
 * Popout terminal page — renders a single pod terminal in its own browser window.
 * URL: /popout/terminal/[podKey]
 */
export default function PopoutTerminalPage() {
  const params = useParams<{ podKey: string }>();
  const router = useRouter();
  const { token, _hasHydrated } = useAuthStore();
  const podKey = params.podKey;

  useEffect(() => {
    if (_hasHydrated && !token) {
      router.push("/login");
    }
  }, [_hasHydrated, token, router]);

  // Set the window title to include the pod key
  useEffect(() => {
    if (podKey) {
      document.title = `Terminal — ${podKey.substring(0, 8)}`;
    }
  }, [podKey]);

  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-terminal-bg">
        <Spinner />
      </div>
    );
  }

  if (!token || !podKey) {
    return null;
  }

  return (
    <RealtimeProvider>
      <div className="h-screen w-screen bg-terminal-bg">
        <TerminalPane
          paneId={`popout-${podKey}`}
          podKey={podKey}
          isActive={true}
          showHeader={true}
          className="h-full w-full"
        />
      </div>
    </RealtimeProvider>
  );
}

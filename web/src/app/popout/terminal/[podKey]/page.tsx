"use client";

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useCurrentUser, useAuthStore } from "@/stores/auth";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { TerminalPane } from "@/components/workspace/TerminalPane";
import { Spinner } from "@/components/ui/spinner";
import { getShortPodKey } from "@/lib/pod-display-name";

export default function PopoutTerminalPage() {
  const { podKey } = useParams<{ podKey: string }>();
  const router = useRouter();
  const user = useCurrentUser();
  const _hasHydrated = useAuthStore((s) => s._hasHydrated);

  // Redirect to login if not authenticated
  useEffect(() => {
    if (_hasHydrated && !user) {
      router.push("/login");
    }
  }, [_hasHydrated, user, router]);

  // Set window title to identify the pod
  useEffect(() => {
    if (podKey) {
      document.title = `Terminal - ${getShortPodKey(podKey)}`;
    }
  }, [podKey]);

  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-terminal-bg">
        <Spinner />
      </div>
    );
  }

  if (!user || !podKey) return null;

  return (
    <RealtimeProvider>
      <div className="h-screen w-screen bg-terminal-bg">
        <TerminalPane
          paneId={`popout-${podKey}`}
          podKey={podKey}
          isActive={true}
          showHeader={true}
          allowSplit={false}
        />
      </div>
    </RealtimeProvider>
  );
}

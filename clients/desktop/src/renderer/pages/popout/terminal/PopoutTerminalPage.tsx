import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useCurrentUser } from "@/stores/auth";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { TerminalPane } from "@/components/workspace/TerminalPane";
import { getShortPodKey } from "@/lib/pod-display-name";

// router.tsx <RequireAuth> gates mount. window.open()-spawned BrowserWindow
// shares renderer's bootstrapAuth (PlatformGate) so session is rehydrated.
export function PopoutTerminalPage() {
  const { podKey } = useParams<{ podKey: string }>();
  const user = useCurrentUser();

  useEffect(() => {
    if (podKey) {
      document.title = `Terminal - ${getShortPodKey(podKey)}`;
    }
  }, [podKey]);

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

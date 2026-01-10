"use client";

import { useEffect } from "react";
import { useRouter, useParams } from "next/navigation";

/**
 * Legacy AgentPod page - redirects to new Workspace
 * Kept for backward compatibility with bookmarks and external links
 */
export default function AgentPodRedirect() {
  const router = useRouter();
  const params = useParams();
  const org = params.org as string;

  useEffect(() => {
    // Redirect to workspace
    router.replace(`/${org}/workspace`);
  }, [router, org]);

  return (
    <div className="flex items-center justify-center h-full">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
        <p className="text-muted-foreground">Redirecting to Workspace...</p>
      </div>
    </div>
  );
}

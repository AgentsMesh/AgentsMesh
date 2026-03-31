"use client";

import { useState } from "react";
import { relayPool } from "@/stores/relayConnection";
import { useAcpSessionStore } from "@/stores/acpSession";
import type { AcpPermissionRequest } from "@/stores/acpSession";
import { ShieldAlert, AlertTriangle } from "lucide-react";

interface AcpPermissionDialogProps {
  podKey: string;
  permissions: AcpPermissionRequest[];
}

export function AcpPermissionDialog({ podKey, permissions }: AcpPermissionDialogProps) {
  const removePermission = useAcpSessionStore((s) => s.removePermissionRequest);
  const [sendError, setSendError] = useState<string | null>(null);

  const handleRespond = (requestId: string, approved: boolean) => {
    if (!relayPool.isConnected(podKey)) {
      setSendError("Relay not connected. Please wait and try again.");
      return;
    }
    relayPool.sendAcpCommand(podKey, {
      type: "permission_response",
      requestId: requestId,
      approved,
    });
    removePermission(podKey, requestId);
    setSendError(null);
  };

  return (
    <div className="border-t bg-amber-50 dark:bg-amber-950/30 p-3 space-y-2">
      {sendError && (
        <div className="flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400">
          <AlertTriangle className="h-3 w-3" />
          {sendError}
        </div>
      )}
      {permissions.map((perm) => (
        <div
          key={perm.requestId}
          className="rounded-lg border border-amber-200 dark:border-amber-800 p-3"
        >
          <div className="flex items-center gap-2 mb-2">
            <ShieldAlert className="h-4 w-4 text-amber-600" />
            <span className="text-sm font-medium">Permission Required</span>
          </div>
          <p className="text-sm mb-1">{perm.description}</p>
          <p className="text-xs text-muted-foreground font-mono mb-2">
            {perm.toolName}
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => handleRespond(perm.requestId, true)}
              className="rounded bg-green-600 px-3 py-1 text-xs text-white hover:bg-green-700"
            >
              Approve
            </button>
            <button
              onClick={() => handleRespond(perm.requestId, false)}
              className="rounded bg-red-600 px-3 py-1 text-xs text-white hover:bg-red-700"
            >
              Deny
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

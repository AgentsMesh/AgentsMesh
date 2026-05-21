"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "next-intl";
import { relayPool } from "@/stores/relayConnection";
import { useAcpSessionStore } from "@/stores/acpSession";
import type { AcpPermissionRequest } from "@/stores/acpSession";
import { AlertTriangle } from "lucide-react";
import { AcpAskUserQuestion } from "./AcpAskUserQuestion";
import { AcpToolPermissionCard } from "./AcpToolPermissionCard";

interface AcpPermissionDialogProps {
  podKey: string;
  permissions: AcpPermissionRequest[];
}

export function AcpPermissionDialog({ podKey, permissions }: AcpPermissionDialogProps) {
  const t = useTranslations("acp.permissionDialog");
  const removePermission = useAcpSessionStore((s) => s.removePermissionRequest);
  const [sendError, setSendError] = useState<string | null>(null);

  const handleRespond = useCallback((requestId: string, approved: boolean, updatedInput?: Record<string, unknown>) => {
    if (!relayPool.isConnected(podKey)) {
      setSendError(t("relayNotConnected"));
      return;
    }
    const command: Record<string, unknown> = {
      type: "permission_response",
      requestId,
      approved,
    };
    if (updatedInput) {
      command.updatedInput = updatedInput;
    }
    relayPool.sendAcpCommand(podKey, command);
    removePermission(podKey, requestId);
    setSendError(null);
  }, [podKey, removePermission, t]);

  return (
    <div className="border-t bg-amber-50 dark:bg-amber-950/30 p-3 space-y-2">
      {sendError && (
        <div className="flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400">
          <AlertTriangle className="h-3 w-3" />
          {sendError}
        </div>
      )}
      {permissions.map((perm) =>
        perm.toolName === "AskUserQuestion" ? (
          <AcpAskUserQuestion
            key={perm.requestId}
            permission={perm}
            onRespond={handleRespond}
          />
        ) : (
          <AcpToolPermissionCard
            key={perm.requestId}
            permission={perm}
            onRespond={handleRespond}
          />
        )
      )}
    </div>
  );
}

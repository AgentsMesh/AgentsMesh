"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { ShieldAlert, ShieldCheck, Timer } from "lucide-react";
import type { AcpPermissionRequest } from "@/stores/acpSession";

const PERMISSION_TIMEOUT_SEC = 60;

interface AcpToolPermissionCardProps {
  permission: AcpPermissionRequest;
  onRespond: (requestId: string, approved: boolean, updatedInput?: Record<string, unknown>) => void;
}

export function AcpToolPermissionCard({ permission, onRespond }: AcpToolPermissionCardProps) {
  const t = useTranslations("acp.permissionCard");
  const [remaining, setRemaining] = useState(PERMISSION_TIMEOUT_SEC);
  const onRespondRef = useRef(onRespond);
  const deniedRef = useRef(false);

  useEffect(() => {
    onRespondRef.current = onRespond;
  });

  useEffect(() => {
    deniedRef.current = false;
    const timer = setInterval(() => {
      setRemaining((prev) => {
        if (prev <= 1 && !deniedRef.current) {
          deniedRef.current = true;
          clearInterval(timer);
          onRespondRef.current(permission.requestId, false);
          return 0;
        }
        return prev > 0 ? prev - 1 : 0;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [permission.requestId]); // stable dep — no onRespond

  return (
    <div className="rounded-lg border border-amber-200 dark:border-amber-800 p-3">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <ShieldAlert className="h-4 w-4 text-amber-600" />
          <span className="text-sm font-medium">{t("title")}</span>
        </div>
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Timer className="h-3 w-3" />
          <span>{remaining}s</span>
        </div>
      </div>
      <p className="text-sm mb-1">{permission.description}</p>
      <p className="text-xs text-muted-foreground font-mono mb-2">
        {permission.toolName}
      </p>
      <div className="flex gap-2">
        <button
          onClick={() => onRespond(permission.requestId, true)}
          className="rounded bg-green-600 px-3 py-1 text-xs text-white hover:bg-green-700"
        >
          {t("approve")}
        </button>
        <button
          onClick={() => onRespond(permission.requestId, true, {
            updatedPermissions: [{
              type: "addRules",
              destination: "session",
              rules: [{ tool: permission.toolName, permission: "allow" }],
            }],
          })}
          className="rounded border border-green-600 px-3 py-1 text-xs text-green-600 hover:bg-green-50 dark:hover:bg-green-950 flex items-center gap-1"
        >
          <ShieldCheck className="h-3 w-3" />
          {t("alwaysAllow")}
        </button>
        <button
          onClick={() => onRespond(permission.requestId, false)}
          className="rounded bg-red-600 px-3 py-1 text-xs text-white hover:bg-red-700"
        >
          {t("deny")}
        </button>
      </div>
    </div>
  );
}

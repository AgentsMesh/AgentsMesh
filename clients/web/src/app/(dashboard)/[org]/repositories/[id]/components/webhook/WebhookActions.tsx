"use client";

import { Button } from "@/components/ui/button";
import { Loader2, Trash2 } from "lucide-react";
import type { WebhookActionsProps } from "./types";

export function WebhookActions({
  status,
  actionLoading,
  onRegister,
  onDelete,
  onMarkConfigured,
  t,
}: WebhookActionsProps) {
  return (
    <div className="flex flex-wrap gap-3">
      {/* Register / Re-register button */}
      {(!status?.is_active || status?.needs_manual_setup) && (
        <Button
          variant="outline"
          onClick={onRegister}
          disabled={!!actionLoading}
        >
          {actionLoading === "register" ? (
            <Loader2 className="w-4 h-4 animate-spin mr-2" />
          ) : null}
          {status?.needs_manual_setup
            ? t("repositories.webhook.tryAgain")
            : t("repositories.webhook.register")}
        </Button>
      )}

      {/* Re-register button for active webhooks */}
      {status?.is_active && !status?.needs_manual_setup && (
        <Button
          variant="outline"
          onClick={onRegister}
          disabled={!!actionLoading}
        >
          {actionLoading === "register" ? (
            <Loader2 className="w-4 h-4 animate-spin mr-2" />
          ) : null}
          {t("repositories.webhook.reregister")}
        </Button>
      )}

      {/* Mark as configured button */}
      {status?.needs_manual_setup && (
        <Button
          variant="default"
          onClick={onMarkConfigured}
          disabled={!!actionLoading}
        >
          {actionLoading === "markConfigured" ? (
            <Loader2 className="w-4 h-4 animate-spin mr-2" />
          ) : null}
          {t("repositories.webhook.markConfigured")}
        </Button>
      )}

      {/* Delete button */}
      {(status?.registered || status?.needs_manual_setup) && (
        <Button
          variant="ghost"
          className="text-destructive hover:text-destructive"
          onClick={onDelete}
          disabled={!!actionLoading}
        >
          {actionLoading === "delete" ? (
            <Loader2 className="w-4 h-4 animate-spin mr-2" />
          ) : (
            <Trash2 className="w-4 h-4 mr-2" />
          )}
          {t("repositories.webhook.delete")}
        </Button>
      )}
    </div>
  );
}

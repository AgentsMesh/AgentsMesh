"use client";

import { CheckCircle, XCircle, AlertTriangle } from "lucide-react";
import type { WebhookStatusDisplayProps } from "./types";

export function WebhookStatusDisplay({ status, t }: WebhookStatusDisplayProps) {
  if (!status) return null;

  const renderIcon = () => {
    if (status.is_active && status.registered && !status.needs_manual_setup) {
      return <CheckCircle className="w-5 h-5 text-green-500" />;
    }
    if (status.needs_manual_setup) {
      return <AlertTriangle className="w-5 h-5 text-yellow-500" />;
    }
    return <XCircle className="w-5 h-5 text-red-500" />;
  };

  const renderText = () => {
    if (status.is_active && status.registered && !status.needs_manual_setup) {
      return (
        <span className="text-green-600 dark:text-green-400">
          {t("repositories.webhook.status.registered")}
        </span>
      );
    }
    if (status.needs_manual_setup) {
      return (
        <span className="text-yellow-600 dark:text-yellow-400">
          {t("repositories.webhook.status.needsManualSetup")}
        </span>
      );
    }
    return (
      <span className="text-muted-foreground">
        {t("repositories.webhook.status.notRegistered")}
      </span>
    );
  };

  return (
    <div className="flex items-center gap-3 mb-4">
      {renderIcon()}
      <div className="flex-1">
        <div className="font-medium">{renderText()}</div>
        {status.last_error && (
          <div className="text-xs text-muted-foreground mt-1">
            {status.last_error}
          </div>
        )}
      </div>
    </div>
  );
}

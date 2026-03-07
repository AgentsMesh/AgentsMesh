"use client";

import { useState } from "react";
import { AlertTriangle, Calendar, Zap, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
  ResponsiveDialogDescription,
  ResponsiveDialogBody,
  ResponsiveDialogFooter,
} from "@/components/ui/responsive-dialog";
import { billingApi } from "@/lib/api/billing";
import { getLocalizedErrorMessage } from "@/lib/api/errors";

interface CancelSubscriptionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  periodEnd: string;
  t: (key: string, params?: Record<string, string | number>) => string;
  onCancelled?: () => void;
}

export function CancelSubscriptionDialog({
  open,
  onOpenChange,
  periodEnd,
  t,
  onCancelled,
}: CancelSubscriptionDialogProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cancelType, setCancelType] = useState<"end_of_period" | "immediate">(
    "end_of_period"
  );

  const handleCancel = async () => {
    setLoading(true);
    setError(null);

    try {
      await billingApi.requestCancelSubscription(cancelType === "immediate");
      onCancelled?.();
      onOpenChange(false);
    } catch (err) {
      setError(getLocalizedErrorMessage(err, t, t("billing.cancel.failed") || "Failed to cancel subscription"));
    } finally {
      setLoading(false);
    }
  };

  const formattedDate = new Date(periodEnd).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  const dialogTitle = t("billing.cancel.title");

  return (
    <ResponsiveDialog open={open} onOpenChange={onOpenChange}>
      <ResponsiveDialogContent className="sm:max-w-md">
        <ResponsiveDialogHeader className="text-center pb-2" onClose={() => onOpenChange(false)}>
          <div className="mx-auto w-12 h-12 rounded-full bg-orange-100 dark:bg-orange-900/30 flex items-center justify-center mb-4">
            <AlertTriangle className="w-6 h-6 text-orange-600 dark:text-orange-400" />
          </div>
          <ResponsiveDialogTitle className="text-xl">
            {dialogTitle}
          </ResponsiveDialogTitle>
          <ResponsiveDialogDescription className="text-center">
            {t("billing.cancel.description")}
          </ResponsiveDialogDescription>
        </ResponsiveDialogHeader>

        <ResponsiveDialogBody className="space-y-3">
          {/* Cancel Options */}
          <div className="space-y-3">
            {/* End of Period Option (Recommended) */}
            <button
              type="button"
              className={`group relative w-full p-4 border-2 rounded-xl text-left transition-all duration-200 ${
                cancelType === "end_of_period"
                  ? "border-primary bg-primary/5 shadow-sm"
                  : "border-border hover:border-primary/50 hover:bg-muted/50"
              }`}
              onClick={() => setCancelType("end_of_period")}
            >
              <div className="flex items-start gap-3">
                <div className={`flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center transition-colors ${
                  cancelType === "end_of_period"
                    ? "bg-primary/10 text-primary"
                    : "bg-muted text-muted-foreground group-hover:bg-primary/10 group-hover:text-primary"
                }`}>
                  <Calendar className="w-5 h-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-foreground">
                      {t("billing.cancel.endOfPeriod")}
                    </span>
                    <span className="text-xs px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 font-medium">
                      {t("billing.cancel.recommended")}
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">
                    {t("billing.cancel.endOfPeriodDesc", { date: formattedDate })}
                  </p>
                </div>
                <div className={`flex-shrink-0 w-5 h-5 rounded-full border-2 flex items-center justify-center transition-colors ${
                  cancelType === "end_of_period"
                    ? "border-primary bg-primary"
                    : "border-muted-foreground/30"
                }`}>
                  {cancelType === "end_of_period" && (
                    <CheckCircle2 className="w-4 h-4 text-primary-foreground" />
                  )}
                </div>
              </div>
            </button>

            {/* Immediate Option */}
            <button
              type="button"
              className={`group relative w-full p-4 border-2 rounded-xl text-left transition-all duration-200 ${
                cancelType === "immediate"
                  ? "border-destructive bg-destructive/5 shadow-sm"
                  : "border-border hover:border-destructive/50 hover:bg-muted/50"
              }`}
              onClick={() => setCancelType("immediate")}
            >
              <div className="flex items-start gap-3">
                <div className={`flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center transition-colors ${
                  cancelType === "immediate"
                    ? "bg-destructive/10 text-destructive"
                    : "bg-muted text-muted-foreground group-hover:bg-destructive/10 group-hover:text-destructive"
                }`}>
                  <Zap className="w-5 h-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <span className="font-semibold text-foreground">
                    {t("billing.cancel.immediate")}
                  </span>
                  <p className="text-sm text-muted-foreground mt-1">
                    {t("billing.cancel.immediateDesc")}
                  </p>
                </div>
                <div className={`flex-shrink-0 w-5 h-5 rounded-full border-2 flex items-center justify-center transition-colors ${
                  cancelType === "immediate"
                    ? "border-destructive bg-destructive"
                    : "border-muted-foreground/30"
                }`}>
                  {cancelType === "immediate" && (
                    <CheckCircle2 className="w-4 h-4 text-destructive-foreground" />
                  )}
                </div>
              </div>
            </button>
          </div>

          {/* Warning for immediate cancellation */}
          {cancelType === "immediate" && (
            <div className="flex items-start gap-3 p-4 bg-destructive/10 border border-destructive/20 rounded-xl animate-in fade-in slide-in-from-top-2 duration-200">
              <AlertTriangle className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
              <p className="text-sm text-destructive">
                {t("billing.cancel.immediateWarning")}
              </p>
            </div>
          )}

          {/* Error message */}
          {error && (
            <div className="flex items-start gap-3 p-4 bg-destructive/10 border border-destructive/20 rounded-xl">
              <AlertTriangle className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}
        </ResponsiveDialogBody>

        <ResponsiveDialogFooter className="flex-col sm:flex-row gap-2">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
            className="w-full sm:w-auto"
          >
            {t("billing.cancel.keepSubscription")}
          </Button>
          <Button
            variant="destructive"
            onClick={handleCancel}
            loading={loading}
            className="w-full sm:w-auto"
          >
            {loading
              ? t("billing.cancel.cancelling")
              : t("billing.cancel.confirmCancel")}
          </Button>
        </ResponsiveDialogFooter>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}

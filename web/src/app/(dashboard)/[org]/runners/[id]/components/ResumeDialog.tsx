"use client";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
import { RefreshCw, RotateCcw } from "lucide-react";
import type { RunnerPodData } from "@/lib/api";
import { useTranslations } from "next-intl";

interface ResumeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pod: RunnerPodData | null;
  loading: boolean;
  onConfirm: () => void;
}

/**
 * Dialog for confirming pod resume action
 */
export function ResumeDialog({
  open,
  onOpenChange,
  pod,
  loading,
  onConfirm,
}: ResumeDialogProps) {
  const t = useTranslations();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("runners.detail.resumeDialogTitle")}</DialogTitle>
          <DialogDescription>
            {t("runners.detail.resumeDialogDescription", {
              podKey: pod?.pod_key || "",
            })}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {t("runners.detail.resumeDialogInfo")}
          </p>
        </DialogBody>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            {t("common.cancel")}
          </Button>
          <Button onClick={onConfirm} disabled={loading}>
            {loading ? (
              <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <RotateCcw className="w-4 h-4 mr-2" />
            )}
            {t("runners.detail.confirmResumeBtn")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

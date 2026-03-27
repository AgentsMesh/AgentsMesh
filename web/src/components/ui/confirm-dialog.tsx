"use client";

import * as React from "react";
import { useState, useCallback } from "react";
import { AlertTriangle, Info, AlertCircle, CheckCircle } from "lucide-react";
import { Dialog, DialogContent, DialogFooter, DialogBody } from "./dialog";
import { Button } from "./button";
import { cn } from "@/lib/utils";

// Re-export hook from separate file
export { useConfirmDialog } from "./use-confirm-dialog";
export type { UseConfirmDialogOptions } from "./use-confirm-dialog";

export type ConfirmDialogVariant = "default" | "destructive" | "warning" | "success";

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  onConfirm: () => void | Promise<void>;
  confirmText?: string;
  cancelText?: string;
  variant?: ConfirmDialogVariant;
  showIcon?: boolean;
  loading?: boolean;
  confirmDisabled?: boolean;
  children?: React.ReactNode;
}

const variantConfig: Record<
  ConfirmDialogVariant,
  {
    icon: React.ElementType;
    iconClass: string;
    confirmVariant: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
  }
> = {
  default: { icon: Info, iconClass: "text-primary bg-primary/10", confirmVariant: "default" },
  destructive: { icon: AlertTriangle, iconClass: "text-destructive bg-destructive/10", confirmVariant: "destructive" },
  warning: { icon: AlertCircle, iconClass: "text-yellow-600 dark:text-yellow-400 bg-yellow-100 dark:bg-yellow-900/30", confirmVariant: "default" },
  success: { icon: CheckCircle, iconClass: "text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/30", confirmVariant: "default" },
};

export function ConfirmDialog({
  open, onOpenChange, title, description, onConfirm,
  confirmText = "Confirm", cancelText = "Cancel", variant = "default",
  showIcon = true, loading: externalLoading, confirmDisabled, children,
}: ConfirmDialogProps) {
  const [internalLoading, setInternalLoading] = useState(false);
  const loading = externalLoading ?? internalLoading;
  const config = variantConfig[variant];
  const Icon = config.icon;

  const handleConfirm = useCallback(async () => {
    try {
      setInternalLoading(true);
      await onConfirm();
      onOpenChange(false);
    } catch (error) {
      console.error("ConfirmDialog onConfirm error:", error);
    } finally {
      setInternalLoading(false);
    }
  }, [onConfirm, onOpenChange]);

  const handleCancel = useCallback(() => {
    if (!loading) onOpenChange(false);
  }, [loading, onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleCancel}>
      <DialogContent className="max-w-sm">
        <DialogBody>
          <div className="flex flex-col items-center text-center">
            {showIcon && (
              <div className={cn("w-12 h-12 rounded-full flex items-center justify-center mb-4", config.iconClass)}>
                <Icon className="w-6 h-6" />
              </div>
            )}
            <h3 className="text-lg font-semibold">{title}</h3>
            {description && <p className="text-sm text-muted-foreground mt-2">{description}</p>}
            {children && <div className="mt-4 w-full text-left">{children}</div>}
          </div>
        </DialogBody>
        <DialogFooter className="justify-center sm:justify-center gap-3">
          <Button variant="outline" onClick={handleCancel} disabled={loading} className="min-w-[100px]">
            {cancelText}
          </Button>
          <Button variant={config.confirmVariant} onClick={handleConfirm}
            disabled={loading || confirmDisabled} className="min-w-[100px]">
            {loading ? (
              <span className="flex items-center gap-2">
                <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                Loading...
              </span>
            ) : confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

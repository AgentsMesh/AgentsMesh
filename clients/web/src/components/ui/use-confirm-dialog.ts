"use client";

import * as React from "react";
import { useState, useCallback } from "react";
import type { ConfirmDialogProps, ConfirmDialogVariant } from "./confirm-dialog";

export interface UseConfirmDialogOptions {
  title: string;
  description?: string;
  confirmText?: string;
  cancelText?: string;
  variant?: ConfirmDialogVariant;
}

export function useConfirmDialog(defaultOptions?: UseConfirmDialogOptions) {
  const [open, setOpen] = useState(false);
  const [currentOptions, setCurrentOptions] = useState<UseConfirmDialogOptions>(
    defaultOptions ?? { title: "" }
  );
  const resolveRef = React.useRef<((value: boolean) => void) | null>(null);

  const confirm = useCallback((options?: UseConfirmDialogOptions) => {
    if (options) {
      setCurrentOptions(options);
    } else if (defaultOptions) {
      setCurrentOptions(defaultOptions);
    }
    setOpen(true);
    return new Promise<boolean>((resolve) => {
      resolveRef.current = resolve;
    });
  }, [defaultOptions]);

  const handleConfirm = useCallback(() => {
    resolveRef.current?.(true);
    resolveRef.current = null;
  }, []);

  const handleOpenChange = useCallback((newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      resolveRef.current?.(false);
      resolveRef.current = null;
    }
  }, []);

  const dialogProps: ConfirmDialogProps = {
    open,
    onOpenChange: handleOpenChange,
    onConfirm: handleConfirm,
    ...currentOptions,
  };

  return {
    dialogProps,
    confirm,
    isOpen: open,
  };
}

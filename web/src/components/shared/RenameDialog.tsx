"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";

interface RenameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentName: string;
  onConfirm: (newName: string) => void;
}

const ALIAS_MAX_LENGTH = 100;

/**
 * Inner form content — mounted fresh each time the dialog opens,
 * so local state is automatically reset to `currentName`.
 */
function RenameForm({
  currentName,
  onConfirm,
  onClose,
}: {
  currentName: string;
  onConfirm: (newName: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("workspace");
  const [value, setValue] = useState(currentName);

  const handleSubmit = useCallback(() => {
    onConfirm(value.trim());
    onClose();
  }, [value, onConfirm, onClose]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        handleSubmit();
      }
    },
    [handleSubmit]
  );

  return (
    <DialogContent className="max-w-sm">
      <DialogHeader>
        <DialogTitle>{t("renameDialog.title")}</DialogTitle>
      </DialogHeader>
      <DialogBody>
        <Input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          maxLength={ALIAS_MAX_LENGTH}
          placeholder={t("renameDialog.placeholder")}
          className="w-full"
          autoFocus
        />
        <p className="text-xs text-muted-foreground mt-2">
          {t("renameDialog.hint")}
        </p>
      </DialogBody>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>
          {t("renameDialog.cancel")}
        </Button>
        <Button onClick={handleSubmit}>
          {t("renameDialog.confirm")}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}

export function RenameDialog({
  open,
  onOpenChange,
  currentName,
  onConfirm,
}: RenameDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open && (
        <RenameForm
          currentName={currentName}
          onConfirm={onConfirm}
          onClose={() => onOpenChange(false)}
        />
      )}
    </Dialog>
  );
}

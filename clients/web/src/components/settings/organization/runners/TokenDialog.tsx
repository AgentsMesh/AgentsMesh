"use client";

import { Button } from "@/components/ui/button";
import type { TranslationFn } from "../GeneralSettings";

interface TokenDialogProps {
  token: string;
  onClose: () => void;
  onCopy: () => void;
  t: TranslationFn;
}

export function TokenDialog({
  token,
  onClose,
  onCopy,
  t,
}: TokenDialogProps) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">
          {t("settings.runnersSection.tokenDialog.title")}
        </h3>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.runnersSection.tokenDialog.description")}
        </p>
        <div className="bg-muted p-3 rounded-lg mb-4 flex items-center justify-between">
          <code className="text-sm break-all">{token}</code>
          <Button variant="ghost" size="sm" onClick={onCopy}>
            {t("settings.runnersSection.tokenDialog.copy")}
          </Button>
        </div>
        <Button className="w-full" onClick={onClose}>
          {t("settings.runnersSection.tokenDialog.done")}
        </Button>
      </div>
    </div>
  );
}

export default TokenDialog;

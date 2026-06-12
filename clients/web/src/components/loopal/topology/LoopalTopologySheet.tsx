"use client";

import { useTranslations } from "next-intl";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { LoopalTopologyFlow } from "./LoopalTopologyFlow";

export function LoopalTopologySheet({
  podKey,
  open,
  onOpenChange,
}: {
  podKey: string;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const t = useTranslations("loopal");
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex w-[680px] max-w-[92vw] flex-col p-0">
        <SheetHeader className="border-b border-border px-4 py-3">
          <SheetTitle>{t("topology.sheetTitle")}</SheetTitle>
        </SheetHeader>
        <div className="min-h-0 flex-1">
          <LoopalTopologyFlow podKey={podKey} minimap />
        </div>
      </SheetContent>
    </Sheet>
  );
}

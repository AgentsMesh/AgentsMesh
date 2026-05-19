"use client";

import React, { useState } from "react";
import { ChevronRight } from "lucide-react";
import {
  Collapsible,
  CollapsibleTrigger,
  CollapsibleContent,
} from "@/components/ui/collapsible";

interface AdvancedOptionsProps {
  children: React.ReactNode;
  t: (key: string) => string;
}

export function AdvancedOptions({ children, t }: AdvancedOptionsProps) {
  const [open, setOpen] = useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors w-full py-2">
        <ChevronRight
          className={`h-4 w-4 transition-transform duration-200 ${
            open ? "rotate-90" : ""
          }`}
        />
        {t("ide.createPod.advancedOptions")}
      </CollapsibleTrigger>
      <CollapsibleContent className="space-y-4 pt-2">
        {children}
      </CollapsibleContent>
    </Collapsible>
  );
}

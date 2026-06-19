"use client";

import React from "react";
import { cn } from "@/lib/utils";

interface Tab {
  id: string;
  label: string;
}

interface LoopDetailTabsProps {
  active: string;
  onChange: (id: string) => void;
  tabs?: Tab[];
  rightSlot?: React.ReactNode;
}

export function LoopDetailTabs({ active, onChange, tabs, rightSlot }: LoopDetailTabsProps) {
  const resolvedTabs: Tab[] = tabs ?? [
    { id: "runs", label: "Runs" },
    { id: "prompt", label: "Prompt & config" },
    { id: "autopilot", label: "Autopilot rules" },
    { id: "history", label: "History" },
    { id: "permissions", label: "Permissions" },
  ];

  return (
    <div className="flex items-center gap-2 border-b border-border">
      <div className="flex flex-1 items-center gap-0">
        {resolvedTabs.map((tab) => {
          const isActive = active === tab.id;
          return (
            <button
              key={tab.id}
              type="button"
              data-testid={`loop-tab-${tab.id}`}
              onClick={() => onChange(tab.id)}
              className={cn(
                "relative flex flex-col items-center gap-1.5 px-3.5 py-2.5 text-[13px] transition-colors",
                isActive
                  ? "font-semibold text-foreground"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {tab.label}
              {isActive && (
                <span className="absolute bottom-0 left-1/2 h-0.5 w-9 -translate-x-1/2 rounded-full bg-primary" />
              )}
            </button>
          );
        })}
      </div>
      {rightSlot}
    </div>
  );
}

"use client";

import React, { useState, useRef, useEffect } from "react";
import { cn } from "@/lib/utils";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { PanelLeftClose, PanelLeft } from "lucide-react";

interface SideBarProps {
  className?: string;
  children?: React.ReactNode;
  headerAction?: React.ReactNode;
}

export function SideBar({ className, children, headerAction }: SideBarProps) {
  const t = useTranslations();
  const activeActivity = useIDEStore((s) => s.activeActivity);
  const sidebarOpen = useIDEStore((s) => s.sidebarOpen);
  const sidebarWidth = useIDEStore((s) => s.sidebarWidth);
  const setSidebarWidth = useIDEStore((s) => s.setSidebarWidth);
  const toggleSidebar = useIDEStore((s) => s.toggleSidebar);
  const resizeRef = useRef<HTMLDivElement>(null);
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      const newWidth = Math.min(Math.max(e.clientX - 48, 200), 400);
      setSidebarWidth(newWidth);
    };
    const handleMouseUp = () => setIsResizing(false);

    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    }
    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, setSidebarWidth]);

  const getActivityTitle = (activity: ActivityType): string => {
    switch (activity) {
      case "workspace":
        return t("ide.activities.workspace");
      case "tickets":
        return t("ide.activities.tickets");
      case "channels":
        return t("ide.activities.channels");
      case "mesh":
        return t("ide.activities.mesh");
      case "repositories":
        return t("ide.activities.repositories");
      case "runners":
        return t("ide.activities.runners");
      case "settings":
        return t("ide.activities.settings");
      default:
        return "";
    }
  };

  if (!sidebarOpen) {
    return (
      <aside className={cn("w-0 relative", className)}>
        <Button
          variant="ghost"
          size="sm"
          className="absolute left-2 top-2 z-10"
          onClick={toggleSidebar}
        >
          <PanelLeft className="w-4 h-4" />
        </Button>
      </aside>
    );
  }

  return (
    <aside
      className={cn(
        "bg-background rounded-xl border border-border/50 shadow-sm overflow-hidden flex flex-col relative",
        className,
      )}
      style={{ width: sidebarWidth }}
    >
      {activeActivity !== "settings" && (
        <div className="flex h-12 items-center justify-between gap-2 border-b border-border/60 px-3">
          <div className="flex min-w-0 items-center gap-1.5">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 flex-shrink-0 p-0"
              onClick={toggleSidebar}
              aria-label={t("ide.sidebar.collapse")}
            >
              <PanelLeftClose className="w-4 h-4" />
            </Button>
            <span className="truncate text-sm font-semibold text-foreground">
              {getActivityTitle(activeActivity)}
            </span>
          </div>
          {headerAction}
        </div>
      )}

      <div className="flex-1 overflow-y-auto">{children}</div>

      <div
        ref={resizeRef}
        className={cn(
          "absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/50 transition-colors",
          isResizing && "bg-primary/50",
        )}
        onMouseDown={() => setIsResizing(true)}
      />
    </aside>
  );
}

export default SideBar;

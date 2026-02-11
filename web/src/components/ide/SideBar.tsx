"use client";

import React, { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import {
  ChevronDown,
  ChevronRight,
  PanelLeftClose,
  PanelLeft,
} from "lucide-react";

interface SideBarProps {
  className?: string;
  children?: React.ReactNode;
}

export function SideBar({ className, children }: SideBarProps) {
  const router = useRouter();
  const t = useTranslations();
  const {
    activeActivity,
    sidebarOpen,
    sidebarWidth,
    setSidebarWidth,
    toggleSidebar,
  } = useIDEStore();
  const { currentOrg, organizations, setCurrentOrg } = useAuthStore();
  const [orgDropdownOpen, setOrgDropdownOpen] = useState(false);
  const orgDropdownRef = useRef<HTMLDivElement>(null);
  const resizeRef = useRef<HTMLDivElement>(null);
  const [isResizing, setIsResizing] = useState(false);

  // Close org dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        orgDropdownRef.current &&
        !orgDropdownRef.current.contains(event.target as Node)
      ) {
        setOrgDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Handle sidebar resize
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      const newWidth = Math.min(Math.max(e.clientX - 48, 200), 400); // 48px is activity bar width
      setSidebarWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    }

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, setSidebarWidth]);

  // Get title for current activity
  const getActivityTitle = (activity: ActivityType): string => {
    switch (activity) {
      case "workspace":
        return t("ide.activities.workspace");
      case "tickets":
        return t("ide.activities.tickets");
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
        {/* Show expand button when collapsed */}
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

  const handleOrgChange = (org: typeof currentOrg) => {
    if (org) {
      setCurrentOrg(org);
      setOrgDropdownOpen(false);
      // Navigate to same activity in new org
      router.push(`/${org.slug}/${activeActivity === "workspace" ? "workspace" : activeActivity}`);
    }
  };

  return (
    <aside
      className={cn(
        "bg-background border-r border-border flex flex-col relative",
        className
      )}
      style={{ width: sidebarWidth }}
    >
      {/* Header with organization selector */}
      <div className="h-12 flex items-center justify-between px-3 border-b border-border">
        <div className="flex-1 min-w-0" ref={orgDropdownRef}>
          <button
            className="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded hover:bg-muted truncate"
            onClick={() => setOrgDropdownOpen(!orgDropdownOpen)}
          >
            <div className="w-5 h-5 rounded bg-primary/10 flex items-center justify-center text-xs font-medium text-primary flex-shrink-0">
              {currentOrg?.name?.charAt(0)?.toUpperCase() || "O"}
            </div>
            <span className="font-medium truncate">{currentOrg?.name}</span>
            <ChevronDown
              className={cn(
                "w-4 h-4 text-muted-foreground flex-shrink-0 transition-transform",
                orgDropdownOpen && "rotate-180"
              )}
            />
          </button>

          {/* Organization dropdown */}
          {orgDropdownOpen && organizations.length > 0 && (
            <div className="absolute left-3 right-3 top-12 mt-1 bg-popover border border-border rounded-md shadow-lg z-50 max-h-64 overflow-y-auto">
              {organizations.map((org) => (
                <button
                  key={org.id}
                  className={cn(
                    "w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-muted text-left",
                    org.id === currentOrg?.id && "bg-muted/50"
                  )}
                  onClick={() => handleOrgChange(org)}
                >
                  <div className="w-5 h-5 rounded bg-primary/10 flex items-center justify-center text-xs font-medium text-primary">
                    {org.name.charAt(0).toUpperCase()}
                  </div>
                  <span className="flex-1 truncate">{org.name}</span>
                  {org.id === currentOrg?.id && (
                    <ChevronRight className="w-4 h-4 text-primary" />
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        <Button
          variant="ghost"
          size="sm"
          className="flex-shrink-0 ml-1"
          onClick={toggleSidebar}
        >
          <PanelLeftClose className="w-4 h-4" />
        </Button>
      </div>

      {/* Activity title - hide for settings */}
      {activeActivity !== "settings" && (
        <div className="h-10 flex items-center px-3 border-b border-border">
          <span className="text-xs font-semibold uppercase text-muted-foreground tracking-wider">
            {getActivityTitle(activeActivity)}
          </span>
        </div>
      )}

      {/* Content area - will be populated by activity-specific content */}
      <div className="flex-1 overflow-y-auto">
        {children}
      </div>

      {/* Resize handle */}
      <div
        ref={resizeRef}
        className={cn(
          "absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/50 transition-colors",
          isResizing && "bg-primary/50"
        )}
        onMouseDown={() => setIsResizing(true)}
      />
    </aside>
  );
}

export default SideBar;

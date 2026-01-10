"use client";

import React from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { Button } from "@/components/ui/button";
import { Menu, Network } from "lucide-react";

interface MobileHeaderProps {
  className?: string;
  title?: string;
  actions?: React.ReactNode;
}

/**
 * Get display title for activity
 */
function getActivityTitle(activity: ActivityType): string {
  switch (activity) {
    case "workspace":
      return "Workspace";
    case "tickets":
      return "Tickets";
    case "mesh":
      return "AgentMesh";
    case "repositories":
      return "Repositories";
    case "runners":
      return "Runners";
    case "settings":
      return "Settings";
    default:
      return "AgentMesh";
  }
}

export function MobileHeader({ className, title, actions }: MobileHeaderProps) {
  const { activeActivity, setMobileDrawerOpen } = useIDEStore();

  const displayTitle = title || getActivityTitle(activeActivity);

  return (
    <header
      className={cn(
        "h-14 bg-background border-b border-border flex items-center px-4 gap-3",
        className
      )}
    >
      {/* Hamburger menu button */}
      <Button
        variant="ghost"
        size="sm"
        className="p-2"
        onClick={() => setMobileDrawerOpen(true)}
      >
        <Menu className="w-5 h-5" />
      </Button>

      {/* Logo and title */}
      <Link href="/" className="flex items-center gap-2 flex-1 min-w-0">
        <div className="w-7 h-7 rounded-lg bg-primary flex items-center justify-center flex-shrink-0">
          <Network className="w-4 h-4 text-primary-foreground" />
        </div>
        <span className="font-semibold truncate">{displayTitle}</span>
      </Link>

      {/* Custom actions */}
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </header>
  );
}

export default MobileHeader;

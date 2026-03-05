"use client";

import React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Menu, PanelRight } from "lucide-react";
import { Logo } from "@/components/common";

interface MobileHeaderProps {
  className?: string;
  title?: string;
  actions?: React.ReactNode;
}

export function MobileHeader({ className, title, actions }: MobileHeaderProps) {
  const activeActivity = useIDEStore((s) => s.activeActivity);
  const setMobileDrawerOpen = useIDEStore((s) => s.setMobileDrawerOpen);
  const setMobileSidebarOpen = useIDEStore((s) => s.setMobileSidebarOpen);
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const params = useParams();
  const t = useTranslations();
  const orgSlug = currentOrg?.slug || (params.org as string) || "";

  // Get display title for activity
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
        return "AgentsMesh";
    }
  };

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
      <Link href={`/${orgSlug}/workspace`} className="flex items-center gap-2 flex-1 min-w-0">
        <div className="w-7 h-7 rounded-lg overflow-hidden flex-shrink-0">
          <Logo />
        </div>
        <span className="font-semibold truncate">{displayTitle}</span>
      </Link>

      {/* Custom actions and sidebar toggle */}
      <div className="flex items-center gap-1">
        {actions}
        <Button
          variant="ghost"
          size="sm"
          className="p-2"
          onClick={() => setMobileSidebarOpen(true)}
        >
          <PanelRight className="w-5 h-5" />
        </Button>
      </div>
    </header>
  );
}

export default MobileHeader;

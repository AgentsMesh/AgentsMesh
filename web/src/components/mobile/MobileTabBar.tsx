"use client";

import React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  useIDEStore,
  getMobileActivities,
  type ActivityType,
} from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import {
  Terminal,
  Ticket,
  Network,
  MessageSquare,
  FolderGit2,
  MoreHorizontal,
  type LucideIcon,
} from "lucide-react";

const ICON_MAP: Record<string, LucideIcon> = {
  terminal: Terminal,
  ticket: Ticket,
  network: Network,
  "message-square": MessageSquare,
  repository: FolderGit2,
};

interface MobileTabBarProps {
  className?: string;
}

export function MobileTabBar({ className }: MobileTabBarProps) {
  const { activeActivity, setActiveActivity, setMobileMoreMenuOpen } =
    useIDEStore();
  const { currentOrg } = useAuthStore();
  const params = useParams();
  const t = useTranslations();
  const orgSlug = currentOrg?.slug || (params.org as string) || "";

  const mobileActivities = getMobileActivities();

  // Map activity to route
  const getActivityRoute = (activity: ActivityType): string => {
    switch (activity) {
      case "workspace":
        return `/${orgSlug}/workspace`;
      case "tickets":
        return `/${orgSlug}/tickets`;
      case "channels":
        return `/${orgSlug}/channels`;
      case "mesh":
        return `/${orgSlug}/mesh`;
      case "loops":
        return `/${orgSlug}/loops`;
      case "repositories":
        return `/${orgSlug}/repositories`;
      default:
        return `/${orgSlug}/workspace`;
    }
  };

  return (
    <nav
      className={cn(
        "h-16 bg-background border-t border-border flex items-stretch px-2 safe-area-pb",
        className
      )}
    >
      {mobileActivities.map((activity) => {
        const Icon = ICON_MAP[activity.icon] || Terminal;
        const isActive = activeActivity === activity.id;

        return (
          <Link
            key={activity.id}
            href={getActivityRoute(activity.id)}
            className={cn(
              "flex-1 flex flex-col items-center justify-center gap-1 transition-colors",
              isActive
                ? "text-primary"
                : "text-muted-foreground active:text-foreground"
            )}
            onClick={() => setActiveActivity(activity.id)}
          >
            <Icon className="w-5 h-5" />
            <span className="text-[10px] font-medium">{t(`ide.activities.${activity.id}`)}</span>
          </Link>
        );
      })}

      {/* More button */}
      <button
        className={cn(
          "flex-1 flex flex-col items-center justify-center gap-1 transition-colors",
          "text-muted-foreground active:text-foreground"
        )}
        onClick={() => setMobileMoreMenuOpen(true)}
      >
        <MoreHorizontal className="w-5 h-5" />
        <span className="text-[10px] font-medium">{t("mobile.more")}</span>
      </button>
    </nav>
  );
}

export default MobileTabBar;

"use client";

import React from "react";
import Link from "next/link";
import { usePathname, useParams } from "next/navigation";
import {
  Tooltip,
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipTrigger,
} from "@radix-ui/react-tooltip";
import { cn } from "@/lib/utils";
import { useIDEStore, ACTIVITIES, type ActivityType } from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import {
  Terminal,
  Ticket,
  Network,
  FolderGit2,
  Server,
  Settings,
  Repeat,
  CircleHelp,
  type LucideIcon,
} from "lucide-react";
import { Logo } from "@/components/common";

const ICON_MAP: Record<string, LucideIcon> = {
  terminal: Terminal,
  ticket: Ticket,
  network: Network,
  repeat: Repeat,
  repository: FolderGit2,
  server: Server,
  settings: Settings,
};

interface ActivityBarProps {
  className?: string;
}

export function ActivityBar({ className }: ActivityBarProps) {
  const { activeActivity, setActiveActivity } = useIDEStore();
  const { currentOrg } = useAuthStore();
  const params = useParams();
  const pathname = usePathname();
  const orgSlug = currentOrg?.slug || (params.org as string) || "";
  const t = useTranslations();

  // Map activity to route
  const getActivityRoute = (activity: ActivityType): string => {
    switch (activity) {
      case "workspace":
        return `/${orgSlug}/workspace`;
      case "tickets":
        return `/${orgSlug}/tickets`;
      case "mesh":
        return `/${orgSlug}/mesh`;
      case "loops":
        return `/${orgSlug}/loops`;
      case "repositories":
        return `/${orgSlug}/repositories`;
      case "runners":
        return `/${orgSlug}/runners`;
      case "settings":
        return `/${orgSlug}/settings`;
      default:
        return `/${orgSlug}/workspace`;
    }
  };

  // Determine active activity from pathname
  React.useEffect(() => {
    if (pathname.includes("/workspace")) {
      setActiveActivity("workspace");
    } else if (pathname.includes("/tickets")) {
      setActiveActivity("tickets");
    } else if (pathname.includes("/mesh")) {
      setActiveActivity("mesh");
    } else if (pathname.includes("/loops")) {
      setActiveActivity("loops");
    } else if (pathname.includes("/repositories")) {
      setActiveActivity("repositories");
    } else if (pathname.includes("/runners")) {
      setActiveActivity("runners");
    } else if (pathname.includes("/settings")) {
      setActiveActivity("settings");
    }
  }, [pathname, setActiveActivity]);

  // Split activities into main and bottom (settings)
  const mainActivities = ACTIVITIES.filter((a) => a.id !== "settings");
  const bottomActivities = ACTIVITIES.filter((a) => a.id === "settings");

  return (
    <TooltipProvider delayDuration={300}>
      <aside
        className={cn(
          "w-12 bg-background border-r border-border flex flex-col",
          className
        )}
      >
        {/* Logo */}
        <div className="h-12 flex items-center justify-center border-b border-border">
          <Link href={`/${orgSlug}/workspace`} className="flex items-center justify-center">
            <div className="w-7 h-7 rounded-lg overflow-hidden">
              <Logo />
            </div>
          </Link>
        </div>

        {/* Main activities */}
        <nav className="flex-1 flex flex-col items-center py-2 gap-1">
          {mainActivities.map((activity) => {
            const Icon = ICON_MAP[activity.icon] || Terminal;
            const isActive = activeActivity === activity.id;

            return (
              <Tooltip key={activity.id}>
                <TooltipTrigger asChild>
                  <Link
                    href={getActivityRoute(activity.id)}
                    className={cn(
                      "w-10 h-10 flex items-center justify-center rounded-md transition-colors relative",
                      isActive
                        ? "text-foreground"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted"
                    )}
                    onClick={() => setActiveActivity(activity.id)}
                  >
                    {/* Active indicator */}
                    {isActive && (
                      <div className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-6 bg-primary rounded-r" />
                    )}
                    <Icon className="w-5 h-5" />
                  </Link>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent
                    side="right"
                    className="z-50 bg-popover text-popover-foreground px-2 py-1 text-sm rounded shadow-md border border-border"
                  >
                    {t(`ide.activities.${activity.id}`)}
                  </TooltipContent>
                </TooltipPortal>
              </Tooltip>
            );
          })}
        </nav>

        {/* Bottom activities (Help + Settings) */}
        <nav className="flex flex-col items-center py-2 gap-1 border-t border-border">
          {/* Help & Feedback */}
          <Tooltip>
            <TooltipTrigger asChild>
              <a
                href="https://discord.gg/XmnSebwHY"
                target="_blank"
                rel="noopener noreferrer"
                className="w-10 h-10 flex items-center justify-center rounded-md transition-colors text-muted-foreground hover:text-foreground hover:bg-muted"
              >
                <CircleHelp className="w-5 h-5" />
              </a>
            </TooltipTrigger>
            <TooltipPortal>
              <TooltipContent
                side="right"
                className="z-50 bg-popover text-popover-foreground px-2 py-1 text-sm rounded shadow-md border border-border"
              >
                {t("ide.activities.help")}
              </TooltipContent>
            </TooltipPortal>
          </Tooltip>

          {bottomActivities.map((activity) => {
            const Icon = ICON_MAP[activity.icon] || Settings;
            const isActive = activeActivity === activity.id;

            return (
              <Tooltip key={activity.id}>
                <TooltipTrigger asChild>
                  <Link
                    href={getActivityRoute(activity.id)}
                    className={cn(
                      "w-10 h-10 flex items-center justify-center rounded-md transition-colors relative",
                      isActive
                        ? "text-foreground"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted"
                    )}
                    onClick={() => setActiveActivity(activity.id)}
                  >
                    {isActive && (
                      <div className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-6 bg-primary rounded-r" />
                    )}
                    <Icon className="w-5 h-5" />
                  </Link>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent
                    side="right"
                    className="z-50 bg-popover text-popover-foreground px-2 py-1 text-sm rounded shadow-md border border-border"
                  >
                    {t(`ide.activities.${activity.id}`)}
                  </TooltipContent>
                </TooltipPortal>
              </Tooltip>
            );
          })}
        </nav>
      </aside>
    </TooltipProvider>
  );
}

export default ActivityBar;

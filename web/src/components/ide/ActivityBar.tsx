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
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTotalUnreadCount } from "@/stores/channelMessageStore";
import { useTranslations } from "next-intl";
import {
  Terminal,
  Ticket,
  Network,
  MessageSquare,
  FolderGit2,
  Server,
  Settings,
  Repeat,
  Blocks,
  Layers,
  LifeBuoy,
  CircleHelp,
  type LucideIcon,
} from "lucide-react";
import { OrgSwitcher } from "@/components/ide/OrgSwitcher";

const ICON_MAP: Record<string, LucideIcon> = {
  terminal: Terminal,
  ticket: Ticket,
  network: Network,
  "message-square": MessageSquare,
  repeat: Repeat,
  blocks: Blocks,
  repository: FolderGit2,
  server: Server,
  settings: Settings,
  layers: Layers,
};

interface ActivityBarProps {
  className?: string;
}

export function ActivityBar({ className }: ActivityBarProps) {
  const activeActivity = useIDEStore((s) => s.activeActivity);
  const setActiveActivity = useIDEStore((s) => s.setActiveActivity);
  const currentOrg = useCurrentOrg();
  const params = useParams();
  const pathname = usePathname();
  const orgSlug = currentOrg?.slug || (params.org as string) || "";
  const t = useTranslations();
  const totalChannelUnread = useTotalUnreadCount();

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
      case "blocks":
        return `/${orgSlug}/blocks`;
      case "infra":
        return `/${orgSlug}/infra`;
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

  React.useEffect(() => {
    if (pathname.includes("/workspace")) setActiveActivity("workspace");
    else if (pathname.includes("/tickets")) setActiveActivity("tickets");
    else if (pathname.includes("/channels")) setActiveActivity("channels");
    else if (pathname.includes("/mesh")) setActiveActivity("mesh");
    else if (pathname.includes("/loops")) setActiveActivity("loops");
    else if (pathname.includes("/blocks")) setActiveActivity("blocks");
    else if (pathname.includes("/infra")) setActiveActivity("infra");
    else if (pathname.includes("/repositories")) setActiveActivity("repositories");
    else if (pathname.includes("/runners")) setActiveActivity("runners");
    else if (pathname.includes("/settings")) setActiveActivity("settings");
  }, [pathname, setActiveActivity]);

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
        {/* Org switcher (replaces logo per design — design/pages/*.pastel activity_bar > org_switcher) */}
        <div className="flex h-12 items-center justify-center border-b border-border">
          <OrgSwitcher />
        </div>

        <nav className="flex-1 flex flex-col items-center py-2 gap-1">
          {mainActivities.map((activity, idx) => {
            const Icon = ICON_MAP[activity.icon] || Terminal;
            const isActive = activeActivity === activity.id;
            const showBadge = activity.id === "channels" && totalChannelUnread > 0;

            // Divider between core (channels/workspace/blocks) and orchestration.
            const prev = mainActivities[idx - 1];
            const isOrchestration = ["tickets", "loops", "mesh", "infra"].includes(activity.id);
            const prevIsCore = prev && ["channels", "workspace", "blocks"].includes(prev.id);
            const showDivider = isOrchestration && prevIsCore;

            return (
              <React.Fragment key={activity.id}>
                {showDivider && (
                  <div className="my-1 h-px w-6 bg-border" aria-hidden="true" />
                )}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Link
                      href={getActivityRoute(activity.id)}
                      className={cn(
                        "w-10 h-10 flex items-center justify-center rounded-md transition-colors relative",
                        isActive
                          ? "text-foreground"
                          : "text-muted-foreground hover:text-foreground hover:bg-muted",
                      )}
                      onClick={() => setActiveActivity(activity.id)}
                    >
                      {isActive && (
                        <div className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-6 bg-primary rounded-r" />
                      )}
                      <Icon className="w-5 h-5" />
                      {showBadge && (
                        <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-[16px] px-0.5 text-[9px] font-bold rounded-full bg-destructive text-destructive-foreground flex items-center justify-center leading-none">
                          {totalChannelUnread > 99 ? "99+" : totalChannelUnread}
                        </span>
                      )}
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
              </React.Fragment>
            );
          })}
        </nav>

        <nav className="flex flex-col items-center py-2 gap-1 border-t border-border">
          <Tooltip>
            <TooltipTrigger asChild>
              <Link
                href="/support"
                className={cn(
                  "w-10 h-10 flex items-center justify-center rounded-md transition-colors relative",
                  pathname.startsWith("/support")
                    ? "text-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                )}
              >
                {pathname.startsWith("/support") && (
                  <div className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-6 bg-primary rounded-r" />
                )}
                <LifeBuoy className="w-5 h-5" />
              </Link>
            </TooltipTrigger>
            <TooltipPortal>
              <TooltipContent
                side="right"
                className="z-50 bg-popover text-popover-foreground px-2 py-1 text-sm rounded shadow-md border border-border"
              >
                {t("support.title")}
              </TooltipContent>
            </TooltipPortal>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <a
                href="https://discord.gg/3RcX7VBbH9"
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

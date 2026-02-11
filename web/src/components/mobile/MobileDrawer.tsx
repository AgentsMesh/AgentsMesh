"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { Drawer } from "vaul";
import * as VisuallyHidden from "@radix-ui/react-visually-hidden";
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
  LogOut,
  ChevronRight,
  type LucideIcon,
} from "lucide-react";

const ICON_MAP: Record<string, LucideIcon> = {
  terminal: Terminal,
  ticket: Ticket,
  network: Network,
  repository: FolderGit2,
  server: Server,
  settings: Settings,
};

interface MobileDrawerProps {
  className?: string;
}

export function MobileDrawer({ className }: MobileDrawerProps) {
  const router = useRouter();
  const { activeActivity, setActiveActivity, mobileDrawerOpen, setMobileDrawerOpen } =
    useIDEStore();
  const { currentOrg, organizations, setCurrentOrg, user, logout } =
    useAuthStore();
  const t = useTranslations();
  const orgSlug = currentOrg?.slug || "";

  const getActivityRoute = (activity: ActivityType): string => {
    switch (activity) {
      case "workspace":
        return `/${orgSlug}/workspace`;
      case "tickets":
        return `/${orgSlug}/tickets`;
      case "mesh":
        return `/${orgSlug}/mesh`;
      case "repositories":
        return `/${orgSlug}/repositories`;
      case "runners":
        return `/${orgSlug}/runners`;
      case "settings":
        return `/${orgSlug}/settings`;
      default:
        return `/${orgSlug}`;
    }
  };

  const handleActivityClick = (activity: ActivityType) => {
    setActiveActivity(activity);
    setMobileDrawerOpen(false);
    router.push(getActivityRoute(activity));
  };

  const handleOrgChange = (org: typeof currentOrg) => {
    if (org) {
      setCurrentOrg(org);
      setMobileDrawerOpen(false);
      router.push(`/${org.slug}/workspace`);
    }
  };

  const handleLogout = () => {
    logout();
    setMobileDrawerOpen(false);
    router.push("/login");
  };

  return (
    <Drawer.Root
      open={mobileDrawerOpen}
      onOpenChange={setMobileDrawerOpen}
      direction="left"
    >
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/40 z-50" />
        <Drawer.Content
          className={cn(
            "fixed left-0 top-0 bottom-0 w-[280px] bg-background z-50 flex flex-col",
            className
          )}
          aria-describedby={undefined}
        >
          {/* Hidden title for accessibility */}
          <VisuallyHidden.Root>
            <Drawer.Title>{t("mobile.drawer.navigationMenu")}</Drawer.Title>
          </VisuallyHidden.Root>

          {/* User info */}
          <div className="p-4 border-b border-border">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                <span className="text-sm font-medium">
                  {user?.username?.[0]?.toUpperCase() || "U"}
                </span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="font-medium truncate">{user?.username}</p>
                <p className="text-xs text-muted-foreground truncate">
                  {user?.email}
                </p>
              </div>
            </div>
          </div>

          {/* Organization selector */}
          <div className="p-2 border-b border-border">
            <p className="px-3 py-1.5 text-xs font-semibold text-muted-foreground uppercase">
              {t("mobile.drawer.organization")}
            </p>
            {organizations.map((org) => (
              <button
                key={org.id}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left",
                  org.id === currentOrg?.id
                    ? "bg-primary/10 text-primary"
                    : "hover:bg-muted"
                )}
                onClick={() => handleOrgChange(org)}
              >
                <div className="w-6 h-6 rounded bg-primary/10 flex items-center justify-center text-xs font-medium text-primary">
                  {org.name.charAt(0).toUpperCase()}
                </div>
                <span className="flex-1 truncate text-sm">{org.name}</span>
                {org.id === currentOrg?.id && (
                  <ChevronRight className="w-4 h-4" />
                )}
              </button>
            ))}
          </div>

          {/* Navigation */}
          <nav className="flex-1 p-2 overflow-y-auto">
            <p className="px-3 py-1.5 text-xs font-semibold text-muted-foreground uppercase">
              {t("mobile.drawer.navigation")}
            </p>
            {ACTIVITIES.map((activity) => {
              const Icon = ICON_MAP[activity.icon] || Terminal;
              const isActive = activeActivity === activity.id;

              return (
                <button
                  key={activity.id}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left transition-colors",
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-muted text-foreground"
                  )}
                  onClick={() => handleActivityClick(activity.id)}
                >
                  <Icon className="w-5 h-5" />
                  <span className="text-sm">{t(`ide.activities.${activity.id}`)}</span>
                </button>
              );
            })}
          </nav>

          {/* Logout */}
          <div className="p-2 border-t border-border">
            <button
              className="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left hover:bg-muted text-muted-foreground"
              onClick={handleLogout}
            >
              <LogOut className="w-5 h-5" />
              <span className="text-sm">{t("mobile.drawer.signOut")}</span>
            </button>
          </div>
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

export default MobileDrawer;

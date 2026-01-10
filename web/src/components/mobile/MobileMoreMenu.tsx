"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import { Drawer } from "vaul";
import { cn } from "@/lib/utils";
import { useIDEStore, getMoreMenuActivities, type ActivityType } from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import {
  Server,
  Settings,
  User,
  GitBranch,
  Moon,
  Sun,
  Monitor,
  type LucideIcon,
} from "lucide-react";

const ICON_MAP: Record<string, LucideIcon> = {
  server: Server,
  settings: Settings,
};

interface MobileMoreMenuProps {
  className?: string;
}

export function MobileMoreMenu({ className }: MobileMoreMenuProps) {
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const { setActiveActivity, mobileMoreMenuOpen, setMobileMoreMenuOpen } =
    useIDEStore();
  const { currentOrg } = useAuthStore();
  const orgSlug = currentOrg?.slug || "";

  const moreActivities = getMoreMenuActivities();

  const getThemeIcon = () => {
    switch (theme) {
      case "light":
        return <Sun className="w-5 h-5" />;
      case "dark":
        return <Moon className="w-5 h-5" />;
      default:
        return <Monitor className="w-5 h-5" />;
    }
  };

  const cycleTheme = () => {
    if (theme === "light") setTheme("dark");
    else if (theme === "dark") setTheme("system");
    else setTheme("light");
  };

  const getActivityRoute = (activity: ActivityType): string => {
    switch (activity) {
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
    setMobileMoreMenuOpen(false);
    router.push(getActivityRoute(activity));
  };

  // Additional menu items
  const additionalItems = [
    {
      id: "git-connections",
      label: "Git Connections",
      icon: GitBranch,
      route: "/settings/git-connections",
    },
    {
      id: "profile",
      label: "Profile",
      icon: User,
      route: "/settings/profile",
    },
  ];

  return (
    <Drawer.Root
      open={mobileMoreMenuOpen}
      onOpenChange={setMobileMoreMenuOpen}
    >
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/40 z-50" />
        <Drawer.Content
          className={cn(
            "fixed bottom-0 left-0 right-0 bg-background rounded-t-2xl z-50",
            className
          )}
          aria-describedby={undefined}
        >
          {/* Handle */}
          <div className="flex justify-center pt-3 pb-2">
            <div className="w-10 h-1 rounded-full bg-muted" />
          </div>

          {/* Title - Required for accessibility */}
          <div className="px-4 pb-2">
            <Drawer.Title className="text-lg font-semibold">More</Drawer.Title>
          </div>

          {/* Menu items */}
          <div className="px-2 pb-safe">
            {/* Activity items */}
            {moreActivities.map((activity) => {
              const Icon = ICON_MAP[activity.icon] || Settings;

              return (
                <button
                  key={activity.id}
                  className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-muted active:bg-muted transition-colors"
                  onClick={() => handleActivityClick(activity.id)}
                >
                  <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                    <Icon className="w-5 h-5" />
                  </div>
                  <span className="text-sm font-medium">{activity.label}</span>
                </button>
              );
            })}

            {/* Divider */}
            <div className="h-px bg-border my-2 mx-4" />

            {/* Additional items */}
            {additionalItems.map((item) => {
              const Icon = item.icon;

              return (
                <button
                  key={item.id}
                  className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-muted active:bg-muted transition-colors"
                  onClick={() => {
                    setMobileMoreMenuOpen(false);
                    router.push(item.route);
                  }}
                >
                  <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                    <Icon className="w-5 h-5" />
                  </div>
                  <span className="text-sm font-medium">{item.label}</span>
                </button>
              );
            })}

            {/* Divider */}
            <div className="h-px bg-border my-2 mx-4" />

            {/* Theme toggle */}
            <button
              className="w-full flex items-center justify-between gap-4 px-4 py-3 rounded-lg hover:bg-muted active:bg-muted transition-colors"
              onClick={cycleTheme}
            >
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                  {getThemeIcon()}
                </div>
                <span className="text-sm font-medium">Theme</span>
              </div>
              <span className="text-xs text-muted-foreground capitalize">
                {theme || "system"}
              </span>
            </button>
          </div>

          {/* Safe area padding */}
          <div className="h-6" />
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

export default MobileMoreMenu;

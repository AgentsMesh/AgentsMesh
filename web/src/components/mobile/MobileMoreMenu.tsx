"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import { Drawer } from "vaul";
import { cn } from "@/lib/utils";
import { useIDEStore, getMoreMenuActivities, type ActivityType } from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "@/lib/i18n/client";
import {
  Server,
  Settings,
  Moon,
  Sun,
  Monitor,
  Palette,
  Check,
  type LucideIcon,
} from "lucide-react";
import { themeConfigs, type Theme } from "@/lib/theme";

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
  const t = useTranslations();
  const orgSlug = currentOrg?.slug || "";

  const moreActivities = getMoreMenuActivities();

  const [themeMenuOpen, setThemeMenuOpen] = React.useState(false);

  const themeIconMap = {
    sun: Sun,
    moon: Moon,
    monitor: Monitor,
    palette: Palette,
  };

  const getThemeIcon = () => {
    const config = themeConfigs.find((c) => c.id === theme);
    if (config) {
      const Icon = themeIconMap[config.icon];
      return <Icon className="w-5 h-5" />;
    }
    return <Monitor className="w-5 h-5" />;
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
            <Drawer.Title className="text-lg font-semibold">{t("mobile.more")}</Drawer.Title>
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
                  <span className="text-sm font-medium">{t(`ide.activities.${activity.id}`)}</span>
                </button>
              );
            })}

            {/* Divider */}
            <div className="h-px bg-border my-2 mx-4" />

            {/* Theme toggle */}
            <div className="relative">
              <button
                className="w-full flex items-center justify-between gap-4 px-4 py-3 rounded-lg hover:bg-muted active:bg-muted transition-colors"
                onClick={() => setThemeMenuOpen(!themeMenuOpen)}
              >
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                    {getThemeIcon()}
                  </div>
                  <span className="text-sm font-medium">{t("mobile.menu.theme")}</span>
                </div>
                <span className="text-xs text-muted-foreground">
                  {t(`mobile.menu.theme_${theme || "system"}`)}
                </span>
              </button>

              {/* Theme submenu */}
              {themeMenuOpen && (
                <div className="ml-14 mr-4 mb-2 bg-secondary rounded-lg overflow-hidden">
                  {themeConfigs.map((config) => {
                    const Icon = themeIconMap[config.icon];
                    const isActive = theme === config.id;

                    return (
                      <button
                        key={config.id}
                        className={cn(
                          "w-full flex items-center justify-between gap-3 px-4 py-2.5 text-sm hover:bg-muted transition-colors",
                          isActive && "bg-muted/50"
                        )}
                        onClick={() => {
                          setTheme(config.id as Theme);
                          setThemeMenuOpen(false);
                        }}
                      >
                        <span className="flex items-center gap-3">
                          <Icon className="w-4 h-4" />
                          {t(`mobile.menu.${config.nameKey}`)}
                        </span>
                        {isActive && <Check className="w-4 h-4 text-primary" />}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Safe area padding */}
          <div className="h-6" />
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

export default MobileMoreMenu;

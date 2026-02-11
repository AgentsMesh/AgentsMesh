"use client";

import React from "react";
import { useRouter, usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { GitBranch, Bell, User, Settings } from "lucide-react";

interface PersonalSettingsSidebarProps {
  className?: string;
}

export function PersonalSettingsSidebar({ className }: PersonalSettingsSidebarProps) {
  const router = useRouter();
  const pathname = usePathname();
  const t = useTranslations();

  // Personal settings tabs configuration
  const settingsTabs = [
    {
      id: "general",
      path: "/settings/general",
      labelKey: "settings.personal.tabs.general",
      icon: Settings,
      descKey: "settings.personal.tabs.generalDesc",
    },
    {
      id: "git",
      path: "/settings/git",
      labelKey: "settings.personal.tabs.git",
      icon: GitBranch,
      descKey: "settings.personal.tabs.gitDesc",
    },
    {
      id: "notifications",
      path: "/settings/notifications",
      labelKey: "settings.personal.tabs.notifications",
      icon: Bell,
      descKey: "settings.personal.tabs.notificationsDesc",
    },
  ];

  // Handle tab click
  const handleTabClick = (path: string) => {
    router.push(path);
  };

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="px-3 py-3 border-b border-border">
        <h3 className="text-sm font-semibold">{t("settings.personal.title")}</h3>
        <p className="text-xs text-muted-foreground mt-0.5">
          {t("settings.personal.description")}
        </p>
      </div>

      {/* Settings navigation */}
      <div className="flex-1 overflow-y-auto py-2">
        {settingsTabs.map((tab) => {
          const Icon = tab.icon;
          const isActive = pathname === tab.path || pathname.startsWith(tab.path + "/");

          return (
            <button
              key={tab.id}
              className={cn(
                "w-full flex items-start gap-3 px-3 py-2 text-left transition-colors",
                isActive
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              )}
              onClick={() => handleTabClick(tab.path)}
            >
              <Icon
                className={cn(
                  "w-4 h-4 mt-0.5 flex-shrink-0",
                  isActive && "text-primary"
                )}
              />
              <div className="flex-1 min-w-0">
                <p className={cn("text-sm truncate", isActive && "font-medium")}>
                  {t(tab.labelKey)}
                </p>
                <p className="text-xs text-muted-foreground truncate">
                  {t(tab.descKey)}
                </p>
              </div>
            </button>
          );
        })}
      </div>

      {/* User info */}
      <div className="border-t border-border px-3 py-3">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <User className="w-3 h-3" />
          <span>{t("settings.personal.userSettings")}</span>
        </div>
      </div>
    </div>
  );
}

export default PersonalSettingsSidebar;

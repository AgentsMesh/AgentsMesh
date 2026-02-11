"use client";

import * as React from "react";
import { useTheme } from "next-themes";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { themeConfigs, type Theme } from "@/lib/theme";
import { Moon, Sun, Monitor, Palette, Check } from "lucide-react";

const iconMap = {
  sun: Sun,
  moon: Moon,
  monitor: Monitor,
  palette: Palette,
};

export function ThemeSettings() {
  const { theme, setTheme } = useTheme();
  const t = useTranslations();
  const [mounted, setMounted] = React.useState(false);

  // Avoid hydration mismatch
  React.useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return (
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.theme.title")}</h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.theme.description")}
        </p>
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3">
          {themeConfigs.map((config) => (
            <div
              key={config.id}
              className="h-20 rounded-lg border border-border bg-muted/50 animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">{t("settings.theme.title")}</h2>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.theme.description")}
      </p>
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3">
        {themeConfigs.map((config) => {
          const Icon = iconMap[config.icon];
          const isActive = theme === config.id;

          return (
            <button
              key={config.id}
              className={cn(
                "relative flex flex-col items-center justify-center gap-2 p-4 rounded-lg border-2 transition-all",
                "hover:bg-muted/50",
                isActive
                  ? "border-primary bg-primary/5"
                  : "border-border"
              )}
              onClick={() => setTheme(config.id as Theme)}
            >
              {isActive && (
                <Check className="w-4 h-4 text-primary absolute top-2 right-2" />
              )}
              <div className={cn(
                "w-10 h-10 rounded-full flex items-center justify-center",
                isActive ? "bg-primary text-primary-foreground" : "bg-muted"
              )}>
                <Icon className="w-5 h-5" />
              </div>
              <span className={cn(
                "text-sm",
                isActive ? "font-medium" : "text-muted-foreground"
              )}>
                {t(`mobile.menu.${config.nameKey}`)}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

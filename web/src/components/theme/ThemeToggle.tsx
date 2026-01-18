"use client";

import * as React from "react";
import { useTheme } from "next-themes";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Moon, Sun, Monitor, Palette, Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { themeConfigs, type Theme } from "@/lib/theme";

interface ThemeToggleProps {
  className?: string;
}

const iconMap = {
  sun: Sun,
  moon: Moon,
  monitor: Monitor,
  palette: Palette,
};

export function ThemeToggle({ className }: ThemeToggleProps) {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const t = useTranslations("mobile.menu");
  const [mounted, setMounted] = React.useState(false);
  const [open, setOpen] = React.useState(false);
  const dropdownRef = React.useRef<HTMLDivElement>(null);

  // Avoid hydration mismatch
  React.useEffect(() => {
    setMounted(true);
  }, []);

  // Close dropdown when clicking outside
  React.useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const currentIconName = React.useMemo(() => {
    if (!mounted) return "monitor" as const;
    if (theme === "system") return "monitor" as const;

    const config = themeConfigs.find((c) => c.id === theme);
    if (config) return config.icon;

    // Fallback based on resolved theme
    return resolvedTheme === "dark" || resolvedTheme === "solarized-dark" ? "moon" as const : "sun" as const;
  }, [mounted, theme, resolvedTheme]);

  const CurrentIcon = iconMap[currentIconName];

  return (
    <div className={cn("relative", className)} ref={dropdownRef}>
      <Button
        variant="ghost"
        size="sm"
        className="w-9 h-9 p-0"
        onClick={() => setOpen(!open)}
        aria-label="Toggle theme"
      >
        <CurrentIcon className="w-4 h-4" />
      </Button>

      {open && (
        <div className="absolute right-0 top-full mt-1 py-1 bg-popover border border-border rounded-md shadow-lg z-50 min-w-40">
          {themeConfigs.map((config) => {
            const Icon = iconMap[config.icon];
            const isActive = theme === config.id;

            return (
              <button
                key={config.id}
                className={cn(
                  "w-full flex items-center justify-between gap-2 px-3 py-1.5 text-sm hover:bg-muted text-left",
                  isActive && "bg-muted/50"
                )}
                onClick={() => {
                  setTheme(config.id as Theme);
                  setOpen(false);
                }}
              >
                <span className="flex items-center gap-2">
                  <Icon className="w-4 h-4" />
                  {t(config.nameKey)}
                </span>
                {isActive && <Check className="w-4 h-4 text-primary" />}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default ThemeToggle;

"use client";

import * as React from "react";
import { useTheme } from "next-themes";
import { Button } from "@/components/ui/button";
import { Moon, Sun, Monitor } from "lucide-react";
import { cn } from "@/lib/utils";

interface ThemeToggleProps {
  className?: string;
  variant?: "default" | "compact" | "dropdown";
}

export function ThemeToggle({ className, variant = "default" }: ThemeToggleProps) {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const [mounted, setMounted] = React.useState(false);

  // Avoid hydration mismatch
  React.useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return (
      <Button variant="ghost" size="sm" className={cn("w-9 h-9 p-0", className)}>
        <Monitor className="w-4 h-4" />
      </Button>
    );
  }

  if (variant === "compact") {
    return (
      <Button
        variant="ghost"
        size="sm"
        className={cn("w-9 h-9 p-0", className)}
        onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        title={`Switch to ${resolvedTheme === "dark" ? "light" : "dark"} mode`}
      >
        {resolvedTheme === "dark" ? (
          <Sun className="w-4 h-4" />
        ) : (
          <Moon className="w-4 h-4" />
        )}
      </Button>
    );
  }

  if (variant === "dropdown") {
    return (
      <div className={cn("relative group", className)}>
        <Button variant="ghost" size="sm" className="w-9 h-9 p-0">
          {theme === "system" ? (
            <Monitor className="w-4 h-4" />
          ) : resolvedTheme === "dark" ? (
            <Moon className="w-4 h-4" />
          ) : (
            <Sun className="w-4 h-4" />
          )}
        </Button>
        <div className="absolute right-0 top-full mt-1 py-1 bg-popover border border-border rounded-md shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50 min-w-32">
          <button
            className={cn(
              "w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-muted text-left",
              theme === "light" && "bg-muted/50"
            )}
            onClick={() => setTheme("light")}
          >
            <Sun className="w-4 h-4" />
            Light
          </button>
          <button
            className={cn(
              "w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-muted text-left",
              theme === "dark" && "bg-muted/50"
            )}
            onClick={() => setTheme("dark")}
          >
            <Moon className="w-4 h-4" />
            Dark
          </button>
          <button
            className={cn(
              "w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-muted text-left",
              theme === "system" && "bg-muted/50"
            )}
            onClick={() => setTheme("system")}
          >
            <Monitor className="w-4 h-4" />
            System
          </button>
        </div>
      </div>
    );
  }

  // Default: three-way toggle buttons
  return (
    <div className={cn("flex items-center gap-1 p-1 bg-muted rounded-lg", className)}>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "h-7 px-2",
          theme === "light" && "bg-background shadow-sm"
        )}
        onClick={() => setTheme("light")}
        title="Light mode"
      >
        <Sun className="w-4 h-4" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "h-7 px-2",
          theme === "dark" && "bg-background shadow-sm"
        )}
        onClick={() => setTheme("dark")}
        title="Dark mode"
      >
        <Moon className="w-4 h-4" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "h-7 px-2",
          theme === "system" && "bg-background shadow-sm"
        )}
        onClick={() => setTheme("system")}
        title="System preference"
      >
        <Monitor className="w-4 h-4" />
      </Button>
    </div>
  );
}

export default ThemeToggle;

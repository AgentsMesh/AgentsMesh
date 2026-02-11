"use client";

import { useLocale } from "next-intl";
import { useSetLocale } from "@/lib/i18n/locale-switcher";
import { locales, localeNames, type Locale } from "@/lib/i18n/config";
import { Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useState } from "react";

interface LanguageSwitcherProps {
  variant?: "icon" | "full" | "select";
  className?: string;
}

export function LanguageSwitcher({ variant = "icon", className }: LanguageSwitcherProps) {
  const locale = useLocale() as Locale;
  const setLocale = useSetLocale();
  const [open, setOpen] = useState(false);

  const handleLocaleChange = (newLocale: Locale) => {
    setLocale(newLocale);
    setOpen(false);
  };

  // Select variant - simple dropdown for settings page
  if (variant === "select") {
    return (
      <select
        value={locale}
        onChange={(e) => setLocale(e.target.value as Locale)}
        className={`border border-border rounded-md px-3 py-2 bg-background text-sm ${className}`}
      >
        {locales.map((loc) => (
          <option key={loc} value={loc}>
            {localeNames[loc]}
          </option>
        ))}
      </select>
    );
  }

  // Full variant - shows current language name
  if (variant === "full") {
    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="ghost" size="sm" className={className}>
            <Globe className="h-4 w-4 mr-2" />
            {localeNames[locale]}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-40 p-1" align="end">
          <div className="flex flex-col">
            {locales.map((loc) => (
              <button
                key={loc}
                onClick={() => handleLocaleChange(loc)}
                className={`flex items-center px-3 py-2 text-sm rounded-md hover:bg-muted transition-colors ${
                  locale === loc ? "bg-muted font-medium" : ""
                }`}
              >
                {localeNames[loc]}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
    );
  }

  // Icon variant (default) - just shows globe icon
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className={className}>
          <Globe className="h-4 w-4" />
          <span className="sr-only">Switch language</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-40 p-1" align="end">
        <div className="flex flex-col">
          {locales.map((loc) => (
            <button
              key={loc}
              onClick={() => handleLocaleChange(loc)}
              className={`flex items-center px-3 py-2 text-sm rounded-md hover:bg-muted transition-colors ${
                locale === loc ? "bg-muted font-medium" : ""
              }`}
            >
              {localeNames[loc]}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

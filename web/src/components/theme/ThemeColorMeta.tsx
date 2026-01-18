"use client";

import { useTheme } from "next-themes";
import { useEffect } from "react";
import { themeColors } from "@/lib/theme";

export function ThemeColorMeta() {
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    const color = themeColors[resolvedTheme || "light"] || "#ffffff";
    const metaTag = document.querySelector('meta[name="theme-color"]');
    if (metaTag) {
      metaTag.setAttribute("content", color);
    }
  }, [resolvedTheme]);

  return null;
}

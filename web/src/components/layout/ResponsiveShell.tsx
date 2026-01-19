"use client";

import React from "react";
import { useBreakpoint } from "./useBreakpoint";
import { IDEShell } from "@/components/ide";
import { MobileShell } from "@/components/mobile";

interface ResponsiveShellProps {
  children: React.ReactNode;
  sidebarContent?: React.ReactNode;
  mobileTitle?: string;
  mobileHeaderActions?: React.ReactNode;
  hideMobileTabBar?: boolean;
}

/**
 * ResponsiveShell - Automatically switches between IDE and Mobile layouts
 *
 * - Desktop (≥1024px): IDE-style layout with activity bar, sidebar, bottom panel
 * - Tablet (768-1024px): Compact IDE layout
 * - Mobile (<768px): Mobile layout with header, bottom tab bar, drawers
 */
export function ResponsiveShell({
  children,
  sidebarContent,
  mobileTitle,
  mobileHeaderActions,
  hideMobileTabBar = false,
}: ResponsiveShellProps) {
  const { isMobile } = useBreakpoint();

  // Mobile layout
  if (isMobile) {
    return (
      <MobileShell
        title={mobileTitle}
        headerActions={mobileHeaderActions}
        hideTabBar={hideMobileTabBar}
      >
        {children}
      </MobileShell>
    );
  }

  // Desktop and Tablet use IDE layout
  return (
    <IDEShell sidebarContent={sidebarContent}>
      {children}
    </IDEShell>
  );
}

export default ResponsiveShell;

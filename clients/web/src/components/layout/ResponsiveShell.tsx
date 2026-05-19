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

export function ResponsiveShell({
  children,
  sidebarContent,
  mobileTitle,
  mobileHeaderActions,
  hideMobileTabBar = false,
}: ResponsiveShellProps) {
  const { isMobile } = useBreakpoint();

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

  return (
    <IDEShell sidebarContent={sidebarContent}>
      {children}
    </IDEShell>
  );
}

export default ResponsiveShell;

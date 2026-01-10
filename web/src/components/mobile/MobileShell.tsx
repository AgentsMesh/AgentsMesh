"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { MobileHeader } from "./MobileHeader";
import { MobileTabBar } from "./MobileTabBar";
import { MobileDrawer } from "./MobileDrawer";
import { MobileMoreMenu } from "./MobileMoreMenu";
import { useIDEStore } from "@/stores/ide";

interface MobileShellProps {
  children: React.ReactNode;
  title?: string;
  headerActions?: React.ReactNode;
  hideTabBar?: boolean;
  className?: string;
}

/**
 * MobileShell - Mobile layout
 *
 * Layout structure:
 * ┌─────────────────────────────────┐
 * │ Header (hamburger + title)      │
 * ├─────────────────────────────────┤
 * │                                 │
 * │       Main Content Area         │
 * │                                 │
 * ├─────────────────────────────────┤
 * │   Bottom Tab Bar (5 items)      │
 * └─────────────────────────────────┘
 */
export function MobileShell({
  children,
  title,
  headerActions,
  hideTabBar = false,
  className,
}: MobileShellProps) {
  const { _hasHydrated } = useIDEStore();

  // Show loading state while hydrating
  if (!_hasHydrated) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col h-screen bg-background", className)}>
      {/* Header */}
      <MobileHeader title={title} actions={headerActions} />

      {/* Main content */}
      <main className="flex-1 overflow-auto">{children}</main>

      {/* Bottom Tab Bar */}
      {!hideTabBar && <MobileTabBar />}

      {/* Drawer */}
      <MobileDrawer />

      {/* More Menu */}
      <MobileMoreMenu />
    </div>
  );
}

export default MobileShell;

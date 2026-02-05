"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { MobileHeader } from "./MobileHeader";
import { MobileTabBar } from "./MobileTabBar";
import { MobileDrawer } from "./MobileDrawer";
import { MobileMoreMenu } from "./MobileMoreMenu";
import { MobileSidebar } from "./MobileSidebar";
import { useIDEStore } from "@/stores/ide";
import { CenteredSpinner } from "@/components/ui/spinner";

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
      <CenteredSpinner className="h-screen bg-background" />
    );
  }

  return (
    <div className={cn("app-shell flex flex-col h-screen bg-background overflow-hidden", className)} style={{ height: '100dvh' }}>
      {/* Header */}
      <MobileHeader title={title} actions={headerActions} />

      {/* Main content */}
      <main className="flex-1 overflow-auto">{children}</main>

      {/* Bottom Tab Bar */}
      {!hideTabBar && <MobileTabBar />}

      {/* Drawer (left side navigation) */}
      <MobileDrawer />

      {/* Sidebar (right side content panel) */}
      <MobileSidebar />

      {/* More Menu */}
      <MobileMoreMenu />
    </div>
  );
}

export default MobileShell;

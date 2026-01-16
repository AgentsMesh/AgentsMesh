"use client";

import React from "react";
import { Drawer } from "vaul";
import * as VisuallyHidden from "@radix-ui/react-visually-hidden";
import { cn } from "@/lib/utils";
import { useIDEStore, type ActivityType } from "@/stores/ide";
import { useAuthStore } from "@/stores/auth";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";

// Import sidebar content components
import { WorkspaceSidebarContent } from "@/components/ide/sidebar/WorkspaceSidebarContent";
import { TicketsSidebarContent } from "@/components/ide/sidebar/TicketsSidebarContent";
import { MeshSidebarContent } from "@/components/ide/sidebar/MeshSidebarContent";
import { RepositoriesSidebarContent } from "@/components/ide/sidebar/RepositoriesSidebarContent";
import { RunnersSidebarContent } from "@/components/ide/sidebar/RunnersSidebarContent";
import { SettingsSidebarContent } from "@/components/ide/sidebar/SettingsSidebarContent";

interface MobileSidebarProps {
  className?: string;
}

/**
 * Get display title for activity
 */
function getActivityTitle(activity: ActivityType): string {
  switch (activity) {
    case "workspace":
      return "Workspace";
    case "tickets":
      return "Tickets";
    case "mesh":
      return "AgentsMesh";
    case "repositories":
      return "Repositories";
    case "runners":
      return "Runners";
    case "settings":
      return "Settings";
    default:
      return "AgentsMesh";
  }
}

/**
 * Get sidebar content based on current activity
 */
function getSidebarContent(activity: ActivityType): React.ReactNode {
  switch (activity) {
    case "workspace":
      return <WorkspaceSidebarContent />;
    case "tickets":
      return <TicketsSidebarContent />;
    case "mesh":
      return <MeshSidebarContent />;
    case "repositories":
      return <RepositoriesSidebarContent />;
    case "runners":
      return <RunnersSidebarContent />;
    case "settings":
      return <SettingsSidebarContent />;
    default:
      return null;
  }
}

/**
 * MobileSidebar - Right-side drawer containing activity-specific sidebar content
 *
 * This provides mobile users access to the same sidebar functionality
 * available on desktop (e.g., ticket lists, channel lists, etc.)
 */
export function MobileSidebar({ className }: MobileSidebarProps) {
  const { activeActivity, mobileSidebarOpen, setMobileSidebarOpen } = useIDEStore();
  const { currentOrg } = useAuthStore();

  const title = getActivityTitle(activeActivity);
  const content = getSidebarContent(activeActivity);

  return (
    <Drawer.Root
      open={mobileSidebarOpen}
      onOpenChange={setMobileSidebarOpen}
      direction="right"
    >
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/40 z-50" />
        <Drawer.Content
          className={cn(
            "fixed right-0 top-0 bottom-0 w-[300px] bg-background z-50 flex flex-col",
            className
          )}
          aria-describedby={undefined}
        >
          {/* Hidden title for accessibility */}
          <VisuallyHidden.Root>
            <Drawer.Title>{title} Panel</Drawer.Title>
          </VisuallyHidden.Root>

          {/* Header */}
          <div className="h-14 flex items-center justify-between px-4 border-b border-border">
            <div className="flex items-center gap-2 min-w-0">
              {currentOrg && (
                <div className="w-6 h-6 rounded bg-primary/10 flex items-center justify-center text-xs font-medium text-primary flex-shrink-0">
                  {currentOrg.name.charAt(0).toUpperCase()}
                </div>
              )}
              <span className="font-semibold truncate">{title}</span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="p-2 flex-shrink-0"
              onClick={() => setMobileSidebarOpen(false)}
            >
              <X className="w-5 h-5" />
            </Button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto">
            {content}
          </div>
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

export default MobileSidebar;

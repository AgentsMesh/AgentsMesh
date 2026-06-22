import React from "react";
import { type ActivityType } from "@/stores/ide";
import { WorkspaceSidebarContent } from "./sidebar/WorkspaceSidebarContent";
import { TicketsSidebarContent } from "./sidebar/TicketsSidebarContent";
import { RepositoriesSidebarContent } from "./sidebar/RepositoriesSidebarContent";
import { RunnersSidebarContent } from "./sidebar/RunnersSidebarContent";
import { InfraSidebarContent } from "./sidebar/InfraSidebarContent";
import { MeshSidebarContent } from "./sidebar/MeshSidebarContent";
import { ChannelsSidebarContent } from "./sidebar/ChannelsSidebarContent";
import { LoopsSidebarContent } from "./sidebar/LoopsSidebarContent";
import { SettingsSidebarContent } from "./sidebar/SettingsSidebarContent";
import { BlocksSidebar } from "@/components/blocks/BlocksSidebar";

export interface SidebarCallbacks {
  onCreatePod?: () => void;
  onAddRunner?: () => void;
  onImportRepo?: () => void;
}

export function getSidebarContent(
  activity: ActivityType,
  callbacks: SidebarCallbacks,
): React.ReactNode {
  switch (activity) {
    case "workspace":
      return <WorkspaceSidebarContent onCreatePod={callbacks.onCreatePod} />;
    case "tickets":
      return <TicketsSidebarContent />;
    case "channels":
      return <ChannelsSidebarContent />;
    case "mesh":
      return <MeshSidebarContent />;
    case "loops":
      return <LoopsSidebarContent />;
    case "blocks":
      return <BlocksSidebar />;
    case "infra":
      return (
        <InfraSidebarContent
          onImportRepo={callbacks.onImportRepo}
          onAddRunner={callbacks.onAddRunner}
        />
      );
    case "repositories":
      return <RepositoriesSidebarContent onImportRepo={callbacks.onImportRepo} />;
    case "runners":
      return <RunnersSidebarContent onAddRunner={callbacks.onAddRunner} />;
    case "settings":
      return <SettingsSidebarContent />;
    default:
      return null;
  }
}

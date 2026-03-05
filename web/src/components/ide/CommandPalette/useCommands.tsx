"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { useIDEStore } from "@/stores/ide";
import {
  Terminal,
  Ticket,
  Network,
  FolderGit2,
  Server,
  Settings,
  Plus,
  LogOut,
} from "lucide-react";
import type { CommandItemData } from "./types";

/**
 * Custom hook for defining navigation and action commands
 */
export function useCommands(t: (key: string) => string): {
  navigationCommands: CommandItemData[];
  actionCommands: CommandItemData[];
} {
  const router = useRouter();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const logout = useAuthStore((s) => s.logout);
  const setActiveActivity = useIDEStore((s) => s.setActiveActivity);
  const orgSlug = currentOrg?.slug || "";

  const navigationCommands: CommandItemData[] = useMemo(
    () => [
      {
        id: "nav-workspace",
        category: "navigation",
        label: t("commandPalette.goToWorkspace"),
        description: t("commandPalette.workspaceDescription"),
        icon: <Terminal className="w-4 h-4" />,
        keywords: ["terminal", "pods", "workspace"],
        action: () => {
          setActiveActivity("workspace");
          router.push(`/${orgSlug}/workspace`);
        },
      },
      {
        id: "nav-tickets",
        category: "navigation",
        label: t("commandPalette.goToTickets"),
        description: t("commandPalette.ticketsDescription"),
        icon: <Ticket className="w-4 h-4" />,
        keywords: ["issues", "tasks", "kanban"],
        action: () => {
          setActiveActivity("tickets");
          router.push(`/${orgSlug}/tickets`);
        },
      },
      {
        id: "nav-mesh",
        category: "navigation",
        label: t("commandPalette.goToMesh"),
        description: t("commandPalette.meshDescription"),
        icon: <Network className="w-4 h-4" />,
        keywords: ["topology", "network", "agents"],
        action: () => {
          setActiveActivity("mesh");
          router.push(`/${orgSlug}/mesh`);
        },
      },
      {
        id: "nav-repositories",
        category: "navigation",
        label: t("commandPalette.goToRepositories"),
        description: t("commandPalette.repositoriesDescription"),
        icon: <FolderGit2 className="w-4 h-4" />,
        keywords: ["git", "repos", "code"],
        action: () => {
          setActiveActivity("repositories");
          router.push(`/${orgSlug}/repositories`);
        },
      },
      {
        id: "nav-runners",
        category: "navigation",
        label: t("commandPalette.goToRunners"),
        description: t("commandPalette.runnersDescription"),
        icon: <Server className="w-4 h-4" />,
        keywords: ["compute", "resources", "agents"],
        action: () => {
          setActiveActivity("runners");
          router.push(`/${orgSlug}/runners`);
        },
      },
      {
        id: "nav-settings",
        category: "navigation",
        label: t("commandPalette.goToSettings"),
        description: t("commandPalette.settingsDescription"),
        icon: <Settings className="w-4 h-4" />,
        keywords: ["config", "preferences"],
        action: () => {
          setActiveActivity("settings");
          router.push(`/${orgSlug}/settings`);
        },
      },
    ],
    [orgSlug, router, setActiveActivity, t]
  );

  const actionCommands: CommandItemData[] = useMemo(
    () => [
      {
        id: "action-new-pod",
        category: "actions",
        label: t("commandPalette.createNewPod"),
        description: t("commandPalette.createPodDescription"),
        icon: <Plus className="w-4 h-4" />,
        keywords: ["new", "create", "pod", "terminal"],
        action: () => {
          router.push(`/${orgSlug}/workspace`);
        },
      },
      {
        id: "action-new-ticket",
        category: "actions",
        label: t("commandPalette.createNewTicket"),
        description: t("commandPalette.createTicketDescription"),
        icon: <Plus className="w-4 h-4" />,
        keywords: ["new", "create", "ticket", "issue"],
        action: () => {
          router.push(`/${orgSlug}/tickets`);
        },
      },
      {
        id: "action-logout",
        category: "actions",
        label: t("commandPalette.signOut"),
        description: t("commandPalette.signOutDescription"),
        icon: <LogOut className="w-4 h-4" />,
        keywords: ["logout", "signout", "exit"],
        action: () => {
          logout();
          router.push("/login");
        },
      },
    ],
    [orgSlug, router, logout, t]
  );

  return { navigationCommands, actionCommands };
}

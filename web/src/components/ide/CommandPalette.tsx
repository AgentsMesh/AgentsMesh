"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useIDEStore, type ActivityType, ACTIVITIES } from "@/stores/ide";
import { useWorkspaceStore } from "@/stores/workspace";
import { podApi, ticketApi, repositoryApi } from "@/lib/api";
import {
  Terminal,
  Ticket,
  Network,
  FolderGit2,
  Server,
  Settings,
  Search,
  Plus,
  Moon,
  Sun,
  LogOut,
  User,
  Building,
  Command as CommandIcon,
  ArrowRight,
  Keyboard,
} from "lucide-react";

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type CommandCategory = "navigation" | "actions" | "search" | "recent";

interface CommandItem {
  id: string;
  category: CommandCategory;
  label: string;
  description?: string;
  icon: React.ReactNode;
  keywords?: string[];
  action: () => void | Promise<void>;
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();
  const { currentOrg, logout } = useAuthStore();
  const { setActiveActivity } = useIDEStore();
  const { addPane } = useWorkspaceStore();
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);

  // Search results
  const [pods, setPods] = useState<Array<{ pod_key: string; status: string }>>([]);
  const [tickets, setTickets] = useState<Array<{ identifier: string; title: string }>>([]);
  const [repositories, setRepositories] = useState<Array<{ id: number; full_path: string }>>([]);

  const orgSlug = currentOrg?.slug || "";

  // Load search results when search changes
  useEffect(() => {
    if (!search || search.length < 2) {
      setPods([]);
      setTickets([]);
      setRepositories([]);
      return;
    }

    const loadSearchResults = async () => {
      setLoading(true);
      try {
        const [podsRes, ticketsRes, reposRes] = await Promise.all([
          podApi.list().catch(() => ({ pods: [] })),
          ticketApi.list().catch(() => ({ tickets: [] })),
          repositoryApi.list().catch(() => ({ repositories: [] })),
        ]);

        // Filter by search term
        const searchLower = search.toLowerCase();
        setPods(
          (podsRes.pods || [])
            .filter((p: any) => p.pod_key.toLowerCase().includes(searchLower))
            .slice(0, 5)
        );
        setTickets(
          (ticketsRes.tickets || [])
            .filter(
              (t: any) =>
                t.identifier.toLowerCase().includes(searchLower) ||
                t.title.toLowerCase().includes(searchLower)
            )
            .slice(0, 5)
        );
        setRepositories(
          (reposRes.repositories || [])
            .filter((r: any) => r.full_path.toLowerCase().includes(searchLower))
            .slice(0, 5)
        );
      } catch (error) {
        console.error("Search error:", error);
      } finally {
        setLoading(false);
      }
    };

    const debounce = setTimeout(loadSearchResults, 300);
    return () => clearTimeout(debounce);
  }, [search]);

  // Navigation commands
  const navigationCommands: CommandItem[] = useMemo(
    () => [
      {
        id: "nav-workspace",
        category: "navigation",
        label: "Go to Workspace",
        description: "Terminal workspace",
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
        label: "Go to Tickets",
        description: "View and manage tickets",
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
        label: "Go to AgentMesh",
        description: "Network topology",
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
        label: "Go to Repositories",
        description: "Manage repositories",
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
        label: "Go to Runners",
        description: "Manage compute resources",
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
        label: "Go to Settings",
        description: "Organization settings",
        icon: <Settings className="w-4 h-4" />,
        keywords: ["config", "preferences"],
        action: () => {
          setActiveActivity("settings");
          router.push(`/${orgSlug}/settings`);
        },
      },
    ],
    [orgSlug, router, setActiveActivity]
  );

  // Action commands
  const actionCommands: CommandItem[] = useMemo(
    () => [
      {
        id: "action-new-pod",
        category: "actions",
        label: "Create New Pod",
        description: "Start a new agent pod",
        icon: <Plus className="w-4 h-4" />,
        keywords: ["new", "create", "pod", "terminal"],
        action: () => {
          router.push(`/${orgSlug}/workspace`);
          // The workspace page will handle showing the create modal
        },
      },
      {
        id: "action-new-ticket",
        category: "actions",
        label: "Create New Ticket",
        description: "Create a new task or issue",
        icon: <Plus className="w-4 h-4" />,
        keywords: ["new", "create", "ticket", "issue"],
        action: () => {
          router.push(`/${orgSlug}/tickets`);
        },
      },
      {
        id: "action-logout",
        category: "actions",
        label: "Sign Out",
        description: "Log out of your account",
        icon: <LogOut className="w-4 h-4" />,
        keywords: ["logout", "signout", "exit"],
        action: () => {
          logout();
          router.push("/login");
        },
      },
    ],
    [orgSlug, router, logout]
  );

  // All static commands
  const allCommands = useMemo(
    () => [...navigationCommands, ...actionCommands],
    [navigationCommands, actionCommands]
  );

  // Handle keyboard shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd+K or Ctrl+K
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        onOpenChange(!open);
      }
      // Escape
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onOpenChange]);

  // Reset search when closing
  useEffect(() => {
    if (!open) {
      setSearch("");
    }
  }, [open]);

  const handleSelect = useCallback(
    async (item: CommandItem) => {
      onOpenChange(false);
      await item.action();
    },
    [onOpenChange]
  );

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
      />

      {/* Command Dialog */}
      <div className="absolute inset-x-4 top-[20%] mx-auto max-w-xl">
        <Command
          className="bg-popover border border-border rounded-lg shadow-2xl overflow-hidden"
          loop
        >
          {/* Search Input */}
          <div className="flex items-center px-4 border-b border-border">
            <Search className="w-4 h-4 text-muted-foreground mr-2" />
            <Command.Input
              placeholder="Type a command or search..."
              className="flex-1 py-3 bg-transparent text-foreground placeholder:text-muted-foreground outline-none"
              value={search}
              onValueChange={setSearch}
            />
            <kbd className="px-2 py-1 text-xs bg-muted rounded text-muted-foreground">
              esc
            </kbd>
          </div>

          {/* Command List */}
          <Command.List className="max-h-80 overflow-y-auto p-2">
            {loading && (
              <Command.Loading className="px-4 py-2 text-sm text-muted-foreground">
                Searching...
              </Command.Loading>
            )}

            <Command.Empty className="px-4 py-6 text-center text-sm text-muted-foreground">
              No results found.
            </Command.Empty>

            {/* Search Results - Pods */}
            {pods.length > 0 && (
              <Command.Group heading="Pods">
                {pods.map((pod) => (
                  <Command.Item
                    key={pod.pod_key}
                    value={`pod-${pod.pod_key}`}
                    className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
                    onSelect={() => {
                      addPane(pod.pod_key, `Pod ${pod.pod_key.substring(0, 8)}`);
                      router.push(`/${orgSlug}/workspace`);
                      onOpenChange(false);
                    }}
                  >
                    <Terminal className="w-4 h-4 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm truncate">
                        <code>{pod.pod_key.substring(0, 12)}...</code>
                      </div>
                      <div className="text-xs text-muted-foreground">{pod.status}</div>
                    </div>
                    <ArrowRight className="w-3 h-3 text-muted-foreground" />
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {/* Search Results - Tickets */}
            {tickets.length > 0 && (
              <Command.Group heading="Tickets">
                {tickets.map((ticket) => (
                  <Command.Item
                    key={ticket.identifier}
                    value={`ticket-${ticket.identifier}`}
                    className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
                    onSelect={() => {
                      router.push(`/${orgSlug}/tickets/${ticket.identifier}`);
                      onOpenChange(false);
                    }}
                  >
                    <Ticket className="w-4 h-4 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm truncate">{ticket.title}</div>
                      <div className="text-xs text-muted-foreground">{ticket.identifier}</div>
                    </div>
                    <ArrowRight className="w-3 h-3 text-muted-foreground" />
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {/* Search Results - Repositories */}
            {repositories.length > 0 && (
              <Command.Group heading="Repositories">
                {repositories.map((repo) => (
                  <Command.Item
                    key={repo.id}
                    value={`repo-${repo.id}`}
                    className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
                    onSelect={() => {
                      router.push(`/${orgSlug}/repositories/${repo.id}`);
                      onOpenChange(false);
                    }}
                  >
                    <FolderGit2 className="w-4 h-4 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm truncate">{repo.full_path}</div>
                    </div>
                    <ArrowRight className="w-3 h-3 text-muted-foreground" />
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {/* Navigation */}
            <Command.Group heading="Navigation">
              {navigationCommands.map((cmd) => (
                <Command.Item
                  key={cmd.id}
                  value={cmd.label}
                  keywords={cmd.keywords}
                  className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
                  onSelect={() => handleSelect(cmd)}
                >
                  <span className="text-muted-foreground">{cmd.icon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm">{cmd.label}</div>
                    {cmd.description && (
                      <div className="text-xs text-muted-foreground">{cmd.description}</div>
                    )}
                  </div>
                </Command.Item>
              ))}
            </Command.Group>

            {/* Actions */}
            <Command.Group heading="Actions">
              {actionCommands.map((cmd) => (
                <Command.Item
                  key={cmd.id}
                  value={cmd.label}
                  keywords={cmd.keywords}
                  className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
                  onSelect={() => handleSelect(cmd)}
                >
                  <span className="text-muted-foreground">{cmd.icon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm">{cmd.label}</div>
                    {cmd.description && (
                      <div className="text-xs text-muted-foreground">{cmd.description}</div>
                    )}
                  </div>
                </Command.Item>
              ))}
            </Command.Group>
          </Command.List>

          {/* Footer */}
          <div className="px-4 py-2 border-t border-border flex items-center justify-between text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-muted rounded">↑↓</kbd>
                Navigate
              </span>
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-muted rounded">↵</kbd>
                Select
              </span>
            </div>
            <span className="flex items-center gap-1">
              <CommandIcon className="w-3 h-3" />K to open
            </span>
          </div>
        </Command>
      </div>
    </div>
  );
}

export default CommandPalette;

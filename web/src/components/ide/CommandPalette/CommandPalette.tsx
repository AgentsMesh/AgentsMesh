"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { Search, Command as CommandIcon } from "lucide-react";
import { useTranslations } from "next-intl";
import { useCommandPaletteSearch } from "./useCommandPaletteSearch";
import { useCommands } from "./useCommands";
import { SearchResultGroups } from "./SearchResultGroups";
import { CommandGroups } from "./CommandGroups";
import type { CommandPaletteProps, CommandItemData, PodSearchResult, TicketSearchResult, RepositorySearchResult } from "./types";

/**
 * CommandPalette - Global command palette for quick navigation and actions
 *
 * Supports keyboard navigation (Cmd+K / Ctrl+K) and provides:
 * - Search for pods, tickets, and repositories
 * - Navigation to different sections
 * - Quick actions (create, logout, etc.)
 */
export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();
  const t = useTranslations();
  const currentOrg = useAuthStore((s) => s.currentOrg);
  const addPane = useWorkspaceStore((s) => s.addPane);
  const [search, setSearch] = useState("");

  const orgSlug = currentOrg?.slug || "";

  // Wrap onOpenChange to reset search when closing
  const handleOpenChange = useCallback((newOpen: boolean) => {
    if (!newOpen) {
      setSearch("");
    }
    onOpenChange(newOpen);
  }, [onOpenChange]);

  // Search results
  const { pods, tickets, repositories, loading } = useCommandPaletteSearch(search);

  // Commands
  const { navigationCommands, actionCommands } = useCommands(t);

  // Handle keyboard shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        handleOpenChange(!open);
      }
      if (e.key === "Escape" && open) {
        handleOpenChange(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, handleOpenChange]);


  const handleSelect = useCallback(
    async (item: CommandItemData) => {
      handleOpenChange(false);
      await item.action();
    },
    [handleOpenChange]
  );

  const handleSelectPod = useCallback(
    (pod: PodSearchResult) => {
      addPane(pod.pod_key);
      router.push(`/${orgSlug}/workspace`);
      handleOpenChange(false);
    },
    [addPane, router, orgSlug, handleOpenChange]
  );

  const handleSelectTicket = useCallback(
    (ticket: TicketSearchResult) => {
      router.push(`/${orgSlug}/tickets/${ticket.slug}`);
      handleOpenChange(false);
    },
    [router, orgSlug, handleOpenChange]
  );

  const handleSelectRepository = useCallback(
    (repo: RepositorySearchResult) => {
      router.push(`/${orgSlug}/repositories/${repo.id}`);
      handleOpenChange(false);
    },
    [router, orgSlug, handleOpenChange]
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
              placeholder={t("commandPalette.placeholder")}
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
                {t("commandPalette.searching")}
              </Command.Loading>
            )}

            <Command.Empty className="px-4 py-6 text-center text-sm text-muted-foreground">
              {t("commandPalette.noResults")}
            </Command.Empty>

            {/* Search Results */}
            <SearchResultGroups
              pods={pods}
              tickets={tickets}
              repositories={repositories}
              orgSlug={orgSlug}
              onSelectPod={handleSelectPod}
              onSelectTicket={handleSelectTicket}
              onSelectRepository={handleSelectRepository}
            />

            {/* Commands */}
            <CommandGroups
              navigationCommands={navigationCommands}
              actionCommands={actionCommands}
              onSelect={handleSelect}
              t={t}
            />
          </Command.List>

          {/* Footer */}
          <div className="px-4 py-2 border-t border-border flex items-center justify-between text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-muted rounded">↑↓</kbd>
                {t("commandPalette.navigate")}
              </span>
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-muted rounded">↵</kbd>
                {t("commandPalette.select")}
              </span>
            </div>
            <span className="flex items-center gap-1">
              <CommandIcon className="w-3 h-3" />K {t("commandPalette.toOpen")}
            </span>
          </div>
        </Command>
      </div>
    </div>
  );
}

export default CommandPalette;

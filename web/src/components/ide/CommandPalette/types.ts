import React from "react";

/**
 * Props for CommandPalette component
 */
export interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Command category types
 */
export type CommandCategory = "navigation" | "actions" | "search" | "recent";

/**
 * Command item interface
 */
export interface CommandItemData {
  id: string;
  category: CommandCategory;
  label: string;
  description?: string;
  icon: React.ReactNode;
  keywords?: string[];
  action: () => void | Promise<void>;
}

/**
 * Search result types
 */
export interface PodSearchResult {
  pod_key: string;
  status: string;
}

export interface TicketSearchResult {
  slug: string;
  title: string;
}

export interface RepositorySearchResult {
  id: number;
  slug: string;
}

/**
 * Search results state
 */
export interface SearchResults {
  pods: PodSearchResult[];
  tickets: TicketSearchResult[];
  repositories: RepositorySearchResult[];
  loading: boolean;
}

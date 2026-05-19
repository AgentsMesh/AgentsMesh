import React from "react";

export interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export type CommandCategory = "navigation" | "actions" | "search" | "recent";

export interface CommandItemData {
  id: string;
  category: CommandCategory;
  label: string;
  description?: string;
  icon: React.ReactNode;
  keywords?: string[];
  action: () => void | Promise<void>;
}

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

export interface SearchResults {
  pods: PodSearchResult[];
  tickets: TicketSearchResult[];
  repositories: RepositorySearchResult[];
  loading: boolean;
}

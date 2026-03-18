"use client";

import { cn } from "@/lib/utils";

export type FilterType = "mine" | "running" | "completed";

interface WorkspaceFiltersProps {
  filter: FilterType;
  onFilterChange: (filter: FilterType) => void;
  t: (key: string) => string;
}

/**
 * Filter tabs for pod list
 */
export function WorkspaceFilters({ filter, onFilterChange, t }: WorkspaceFiltersProps) {
  const filters: FilterType[] = ["mine", "running", "completed"];

  return (
    <div className="flex items-center gap-1 px-2 py-1 border-y border-border">
      {filters.map((f) => (
        <button
          key={f}
          className={cn(
            "px-2 py-1 text-xs rounded transition-colors",
            filter === f
              ? "bg-muted text-foreground font-medium"
              : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
          )}
          onClick={() => onFilterChange(f)}
        >
          {t(`workspace.filters.${f}`)}
        </button>
      ))}
    </div>
  );
}

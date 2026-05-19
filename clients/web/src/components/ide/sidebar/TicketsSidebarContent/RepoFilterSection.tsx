"use client";

import { useEffect, useMemo, useState } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { GitBranch, CircleOff } from "lucide-react";
import { useRepositories, useRepositoryStore } from "@/stores/repository";
import type { Ticket } from "@/stores/ticket";
import { FilterSection } from "./FilterSection";

const INITIAL_VISIBLE_COUNT = 5;

export const NO_REPOSITORY_ID = 0;

interface RepoFilterSectionProps {
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  selectedRepositoryIds: number[];
  onToggleRepository: (id: number) => void;
  allTickets?: Ticket[];
  t: (key: string) => string;
}

export function RepoFilterSection({
  expanded,
  onExpandedChange,
  selectedRepositoryIds,
  onToggleRepository,
  allTickets,
  t,
}: RepoFilterSectionProps) {
  const allRepos = useRepositories();
  const loading = useRepositoryStore((s) => s.isLoading);
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);
  const [showAll, setShowAll] = useState(false);

  useEffect(() => { fetchRepositories(); }, [fetchRepositories]);

  const repositories = useMemo(() => allRepos.filter((r) => r.is_active), [allRepos]);

  const repoCounts = useMemo(() => {
    if (!allTickets) return {};
    const counts: Record<number, number> = {};
    for (const ticket of allTickets) {
      const key = ticket.repository_id ?? NO_REPOSITORY_ID;
      counts[key] = (counts[key] || 0) + 1;
    }
    return counts;
  }, [allTickets]);

  const sortedRepos = useMemo(() => {
    return [...repositories].sort((a, b) => {
      const countDiff = (repoCounts[b.id] || 0) - (repoCounts[a.id] || 0);
      if (countDiff !== 0) return countDiff;
      return a.name.localeCompare(b.name);
    });
  }, [repositories, repoCounts]);

  const visibleRepos = showAll ? sortedRepos : sortedRepos.slice(0, INITIAL_VISIBLE_COUNT);
  const hasMore = sortedRepos.length > INITIAL_VISIBLE_COUNT;
  const noRepoCount = repoCounts[NO_REPOSITORY_ID];

  if (loading || repositories.length === 0) return null;

  return (
    <FilterSection
      title={t("tickets.filters.repository")}
      expanded={expanded}
      onExpandedChange={onExpandedChange}
      selectedCount={selectedRepositoryIds.length}
      showBorder
    >
      {/* "No repository" option */}
      <label className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded">
        <Checkbox
          checked={selectedRepositoryIds.includes(NO_REPOSITORY_ID)}
          onCheckedChange={() => onToggleRepository(NO_REPOSITORY_ID)}
          className="h-3.5 w-3.5"
        />
        <CircleOff className="w-3 h-3 text-muted-foreground shrink-0" />
        <span className="flex-1 truncate text-muted-foreground">
          {t("tickets.filters.noRepository")}
        </span>
        {noRepoCount !== undefined && (
          <span className="text-muted-foreground/60 font-mono">{noRepoCount}</span>
        )}
      </label>

      {/* Repository list */}
      {visibleRepos.map((repo) => {
        const count = repoCounts[repo.id];
        return (
          <label
            key={repo.id}
            className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/50 px-1 py-0.5 rounded"
          >
            <Checkbox
              checked={selectedRepositoryIds.includes(repo.id)}
              onCheckedChange={() => onToggleRepository(repo.id)}
              className="h-3.5 w-3.5"
            />
            <GitBranch className="w-3 h-3 text-muted-foreground shrink-0" />
            <span className="flex-1 truncate">{repo.name}</span>
            {count !== undefined && (
              <span className="text-muted-foreground/60 font-mono">{count}</span>
            )}
          </label>
        );
      })}
      {hasMore && (
        <button
          type="button"
          className="text-xs text-primary hover:underline px-1 pt-1"
          onClick={() => setShowAll(!showAll)}
        >
          {showAll ? t("tickets.filters.showLess") : t("tickets.filters.showMore")}
        </button>
      )}
    </FilterSection>
  );
}

"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { GitBranch, CircleOff } from "lucide-react";
import { repositoryApi, type RepositoryData } from "@/lib/api";
import type { Ticket } from "@/stores/ticket";
import { FilterSection } from "./FilterSection";

/** Max repos shown before "show more" */
const INITIAL_VISIBLE_COUNT = 5;

/** Sentinel value representing tickets with no repository */
export const NO_REPOSITORY_ID = 0;

interface RepoFilterSectionProps {
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  selectedRepositoryIds: number[];
  onToggleRepository: (id: number) => void;
  allTickets?: Ticket[];
  t: (key: string) => string;
}

/**
 * RepoFilterSection - Repository checkbox filter with show more/less.
 * Loads repository list from API on mount, derives ticket counts from allTickets.
 */
export function RepoFilterSection({
  expanded,
  onExpandedChange,
  selectedRepositoryIds,
  onToggleRepository,
  allTickets,
  t,
}: RepoFilterSectionProps) {
  const [repositories, setRepositories] = useState<RepositoryData[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAll, setShowAll] = useState(false);

  const loadRepos = useCallback(async () => {
    try {
      const res = await repositoryApi.list();
      setRepositories((res.repositories || []).filter((r) => r.is_active));
    } catch {
      // Silently fail — filter section simply won't render
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadRepos(); }, [loadRepos]);

  // Count tickets per repository (0 = no repository)
  const repoCounts = useMemo(() => {
    if (!allTickets) return {};
    const counts: Record<number, number> = {};
    for (const ticket of allTickets) {
      const key = ticket.repository_id ?? NO_REPOSITORY_ID;
      counts[key] = (counts[key] || 0) + 1;
    }
    return counts;
  }, [allTickets]);

  // Sort repos: those with tickets first (descending count), then alphabetical
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

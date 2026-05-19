"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useRepositories, useRepositoryStore, type Repository } from "@/stores/repository";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  FolderGit2,
  Loader2,
  Plus,
  Search,
  RefreshCw,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { RepositoryItem } from "./RepositoryItem";

interface RepositoriesSidebarContentProps {
  className?: string;
  /** Callback when "Import Repository" is clicked. If provided, opens modal; otherwise navigates to repositories page */
  onImportRepo?: () => void;
}

const PROVIDER_FILTER_VALUES = ["all", "github", "gitlab", "gitee"] as const;

export function RepositoriesSidebarContent({ className, onImportRepo }: RepositoriesSidebarContentProps) {
  const t = useTranslations();
  const router = useRouter();
  const searchParams = useSearchParams();
  const currentOrg = useCurrentOrg();

  // Store-backed list (SSOT). Local UI state only for filters/search.
  const repositories = useRepositories();
  const loading = useRepositoryStore((s) => s.isLoading);
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);
  const [refreshing, setRefreshing] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProvider, setSelectedProvider] = useState("all");
  const [expandedRepos, setExpandedRepos] = useState<Set<number>>(new Set());

  const selectedRepoId = useMemo(() => {
    const raw = searchParams.get("id");
    if (!raw) return null;
    const n = Number(raw);
    return Number.isNaN(n) ? null : n;
  }, [searchParams]);

  useEffect(() => {
    if (currentOrg) fetchRepositories();
  }, [currentOrg, fetchRepositories]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await fetchRepositories();
    } finally {
      setRefreshing(false);
    }
  }, [fetchRepositories]);

  const filteredRepositories = useMemo(() => repositories.filter((repo) => {
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesName = repo.name.toLowerCase().includes(query);
      const matchesPath = repo.slug.toLowerCase().includes(query);
      if (!matchesName && !matchesPath) return false;
    }

    if (selectedProvider !== "all" && repo.provider_type !== selectedProvider) {
      return false;
    }

    return true;
  }), [repositories, searchQuery, selectedProvider]);

  const handleRepoClick = (repo: Repository) => {
    router.push(`/${currentOrg?.slug}/infra?tab=repositories&id=${repo.id}`);
  };

  const toggleRepoExpand = (repoId: number, e: React.MouseEvent) => {
    e.stopPropagation();
    setExpandedRepos(prev => {
      const next = new Set(prev);
      if (next.has(repoId)) {
        next.delete(repoId);
      } else {
        next.add(repoId);
      }
      return next;
    });
  };

  const handleImportRepo = () => {
    if (onImportRepo) {
      onImportRepo();
    } else {
      router.push(`/${currentOrg?.slug}/infra?tab=repositories`);
    }
  };

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <div className="px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder={t("repositories.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 text-sm"
          />
        </div>
      </div>

      <div className="flex items-center gap-1 px-2 pb-2">
        <Button
          size="sm"
          variant="outline"
          className="flex-1 h-8 text-xs"
          onClick={handleImportRepo}
        >
          <Plus className="w-3 h-3 mr-1" />
          {t("repositories.import")}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          className="h-8 w-8 p-0"
          onClick={handleRefresh}
          disabled={refreshing}
        >
          <RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
        </Button>
      </div>

      <div className="px-2 pb-2">
        <div className="flex items-center gap-1 flex-wrap">
          {PROVIDER_FILTER_VALUES.map((value) => (
            <button
              key={value}
              className={cn(
                "px-2 py-1 text-xs rounded transition-colors",
                selectedProvider === value
                  ? "bg-muted text-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
              )}
              onClick={() => setSelectedProvider(value)}
            >
              {t(`repositories.filters.${value}`)}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto border-t border-border">
        <div className="px-3 py-2 text-xs text-muted-foreground border-b border-border">
          {filteredRepositories.length} {t("repositories.repoCount")}
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : filteredRepositories.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <FolderGit2 className="w-8 h-8 mx-auto mb-2 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              {searchQuery || selectedProvider !== "all"
                ? t("repositories.emptyState.noMatch")
                : t("repositories.emptyState.title")}
            </p>
            {!searchQuery && selectedProvider === "all" && (
              <Button
                size="sm"
                variant="outline"
                className="mt-3"
                onClick={handleImportRepo}
              >
                {t("repositories.import")}
              </Button>
            )}
          </div>
        ) : (
          <div className="py-1">
            {filteredRepositories.map((repo) => (
              <RepositoryItem
                key={repo.id}
                repo={repo}
                isSelected={selectedRepoId === repo.id}
                isExpanded={expandedRepos.has(repo.id)}
                onClick={() => handleRepoClick(repo)}
                onToggleExpand={(e) => toggleRepoExpand(repo.id, e)}
                t={t}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default RepositoriesSidebarContent;

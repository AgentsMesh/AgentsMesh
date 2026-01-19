"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { repositoryApi, RepositoryData } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  FolderGit2,
  GitBranch,
  Loader2,
  Plus,
  Search,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Github,
  Globe,
} from "lucide-react";
// Collapsible imports reserved for future use
import { useTranslations } from "@/lib/i18n/client";

interface RepositoriesSidebarContentProps {
  className?: string;
}

// Provider icons
const providerIcons: Record<string, React.ReactNode> = {
  github: <Github className="w-3.5 h-3.5" />,
  gitlab: <FolderGit2 className="w-3.5 h-3.5" />,
  gitee: <FolderGit2 className="w-3.5 h-3.5" />,
  generic: <Globe className="w-3.5 h-3.5" />,
};

// Provider filter values - labels will be translated
const PROVIDER_FILTER_VALUES = ["all", "github", "gitlab", "gitee"] as const;

export function RepositoriesSidebarContent({ className }: RepositoriesSidebarContentProps) {
  const t = useTranslations();
  const router = useRouter();
  const { currentOrg } = useAuthStore();

  // State
  const [repositories, setRepositories] = useState<RepositoryData[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProvider, setSelectedProvider] = useState("all");
  const [selectedRepoId, setSelectedRepoId] = useState<number | null>(null);
  const [expandedRepos, setExpandedRepos] = useState<Set<number>>(new Set());

  // Load repositories on mount
  useEffect(() => {
    if (currentOrg) {
      loadRepositories();
    }
  }, [currentOrg]);

  const loadRepositories = async () => {
    try {
      const response = await repositoryApi.list();
      setRepositories(response.repositories || []);
    } catch (error) {
      console.error("Failed to load repositories:", error);
    } finally {
      setLoading(false);
    }
  };

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await loadRepositories();
    } finally {
      setRefreshing(false);
    }
  }, []);

  // Filter repositories
  const filteredRepositories = repositories.filter((repo) => {
    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesName = repo.name.toLowerCase().includes(query);
      const matchesPath = repo.full_path.toLowerCase().includes(query);
      if (!matchesName && !matchesPath) return false;
    }

    // Provider filter
    if (selectedProvider !== "all" && repo.provider_type !== selectedProvider) {
      return false;
    }

    return true;
  });

  // Handle repository click
  const handleRepoClick = (repo: RepositoryData) => {
    setSelectedRepoId(repo.id);
    router.push(`/${currentOrg?.slug}/repositories/${repo.id}`);
  };

  // Toggle repository expansion
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

  // Navigate to import page
  const handleImportRepo = () => {
    router.push(`/${currentOrg?.slug}/repositories?import=true`);
  };

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Search */}
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

      {/* Action buttons */}
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

      {/* Provider filter */}
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

      {/* Repository list */}
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
            {filteredRepositories.map((repo) => {
              const isSelected = selectedRepoId === repo.id;
              const isExpanded = expandedRepos.has(repo.id);
              const providerIcon = providerIcons[repo.provider_type] || providerIcons.generic;

              return (
                <div key={repo.id}>
                  <div
                    className={cn(
                      "group flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer",
                      isSelected && "bg-muted/30"
                    )}
                    onClick={() => handleRepoClick(repo)}
                  >
                    {/* Expand button */}
                    <button
                      className="p-0.5 hover:bg-muted rounded"
                      onClick={(e) => toggleRepoExpand(repo.id, e)}
                    >
                      {isExpanded ? (
                        <ChevronDown className="w-3 h-3 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="w-3 h-3 text-muted-foreground" />
                      )}
                    </button>

                    {/* Provider icon */}
                    <span className="text-muted-foreground">
                      {providerIcon}
                    </span>

                    {/* Repo info */}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate font-medium">{repo.name}</p>
                      <p className="text-xs text-muted-foreground truncate">
                        {repo.full_path}
                      </p>
                    </div>

                    {/* Active indicator */}
                    {repo.is_active && (
                      <span className="w-1.5 h-1.5 rounded-full bg-green-500 flex-shrink-0" />
                    )}
                  </div>

                  {/* Expanded content - branch info */}
                  {isExpanded && (
                    <div className="pl-10 pr-3 pb-2">
                      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                        <GitBranch className="w-3 h-3" />
                        <span>{repo.default_branch}</span>
                        <span className="text-muted-foreground/50">({t("repositories.repository.default")})</span>
                      </div>
                      {repo.ticket_prefix && (
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-1">
                          <span className="font-mono bg-muted px-1 rounded">
                            {repo.ticket_prefix}
                          </span>
                          <span>{t("repositories.repository.ticketPrefix")}</span>
                        </div>
                      )}
                      <div className="flex items-center gap-2 mt-2">
                        <a
                          href={repo.clone_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-primary hover:underline flex items-center gap-1"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <ExternalLink className="w-3 h-3" />
                          {t("repositories.repository.viewOnProvider", { provider: repo.provider_type })}
                        </a>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

export default RepositoriesSidebarContent;

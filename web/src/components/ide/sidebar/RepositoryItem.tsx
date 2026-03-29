"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { RepositoryData } from "@/lib/api";
import {
  FolderGit2,
  GitBranch,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Github,
  Globe,
} from "lucide-react";

// Provider icons
const providerIcons: Record<string, React.ReactNode> = {
  github: <Github className="w-3.5 h-3.5" />,
  gitlab: <FolderGit2 className="w-3.5 h-3.5" />,
  gitee: <FolderGit2 className="w-3.5 h-3.5" />,
  generic: <Globe className="w-3.5 h-3.5" />,
};

interface RepositoryItemProps {
  repo: RepositoryData;
  isSelected: boolean;
  isExpanded: boolean;
  onClick: () => void;
  onToggleExpand: (e: React.MouseEvent) => void;
  t: (key: string, params?: Record<string, string>) => string;
}

/**
 * Single repository item in the sidebar list
 */
export function RepositoryItem({
  repo,
  isSelected,
  isExpanded,
  onClick,
  onToggleExpand,
  t,
}: RepositoryItemProps) {
  const providerIcon = providerIcons[repo.provider_type] || providerIcons.generic;

  return (
    <div>
      <div
        className={cn(
          "group flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer",
          isSelected && "bg-muted/30"
        )}
        onClick={onClick}
      >
        {/* Expand button */}
        <button
          className="p-0.5 hover:bg-muted rounded"
          onClick={onToggleExpand}
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
            {repo.slug}
          </p>
        </div>

        {/* Active indicator */}
        {repo.is_active && (
          <span className="w-1.5 h-1.5 rounded-full bg-green-500 flex-shrink-0" />
        )}
      </div>

      {/* Expanded content - branch info */}
      {isExpanded && (
        <RepositoryExpandedContent repo={repo} t={t} />
      )}
    </div>
  );
}

/**
 * Expanded content showing branch and other details
 */
function RepositoryExpandedContent({
  repo,
  t,
}: {
  repo: RepositoryData;
  t: (key: string, params?: Record<string, string>) => string;
}) {
  return (
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
          href={`${repo.provider_base_url.replace(/\/$/, "")}/${repo.slug}`}
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
  );
}

export default RepositoryItem;

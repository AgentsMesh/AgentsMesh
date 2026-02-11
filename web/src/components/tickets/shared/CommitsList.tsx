"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { TicketCommit } from "@/lib/api";
import { GitCommit } from "lucide-react";
import { cn } from "@/lib/utils";

interface CommitsListProps {
  commits: TicketCommit[];
  /** Link to view all commits (for compact mode) */
  viewAllLink?: string;
  /** Maximum commits to show in compact mode */
  maxItems?: number;
  /** Compact style for panel view */
  compact?: boolean;
  className?: string;
}

/**
 * Shared commits list component used by both TicketDetail and TicketDetailPane
 */
export function CommitsList({
  commits,
  viewAllLink,
  maxItems = 5,
  compact = false,
  className,
}: CommitsListProps) {
  const t = useTranslations();

  if (commits.length === 0) return null;

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const displayCommits = compact ? commits.slice(0, maxItems) : commits;
  const hasMore = compact && commits.length > maxItems;

  if (compact) {
    return (
      <div className={cn("space-y-2", className)}>
        <label className="text-[11px] font-medium text-muted-foreground/70 uppercase tracking-wider flex items-center gap-1">
          <GitCommit className="h-3 w-3" />
          {t("tickets.detail.commits")}
        </label>
        <div className="space-y-1">
          {displayCommits.map((commit) => (
            <div key={commit.id} className="px-2.5 py-1.5 rounded-md hover:bg-muted/30 transition-colors">
              <div className="flex items-start gap-2">
                <code className="font-mono text-[10px] text-muted-foreground shrink-0">
                  {commit.commit_sha.substring(0, 7)}
                </code>
                <div className="flex-1 min-w-0">
                  <p className="truncate text-sm">{commit.commit_message}</p>
                  <p className="text-[11px] text-muted-foreground/70 mt-0.5">
                    {commit.author_name} • {commit.committed_at ? formatDate(commit.committed_at) : "N/A"}
                  </p>
                </div>
              </div>
            </div>
          ))}
          {hasMore && viewAllLink && (
            <Link
              href={viewAllLink}
              className="block text-xs text-primary hover:underline px-2.5 py-1"
            >
              {t("common.viewAll")} ({commits.length})
            </Link>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("mb-6", className)}>
      <h3 className="font-medium mb-3 flex items-center gap-2">
        <svg className="w-4 h-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        {t("tickets.detail.commits")} ({commits.length})
      </h3>
      <div className="border border-border rounded-lg divide-y divide-border">
        {commits.map((commit) => (
          <div key={commit.id} className="px-4 py-3">
            <div className="flex items-start gap-3">
              <code className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                {commit.commit_sha.substring(0, 7)}
              </code>
              <div className="flex-1 min-w-0">
                <p className="truncate">{commit.commit_message}</p>
                <p className="text-xs text-muted-foreground mt-1">
                  {commit.author_name} • {commit.committed_at ? new Date(commit.committed_at).toLocaleDateString() : "N/A"}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default CommitsList;

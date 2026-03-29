"use client";

import { Command } from "cmdk";
import { Terminal, Ticket, FolderGit2, ArrowRight } from "lucide-react";
import { getPodDisplayName } from "@/lib/pod-utils";
import type { PodSearchResult, TicketSearchResult, RepositorySearchResult } from "./types";

interface SearchResultGroupsProps {
  pods: PodSearchResult[];
  tickets: TicketSearchResult[];
  repositories: RepositorySearchResult[];
  orgSlug: string;
  onSelectPod: (pod: PodSearchResult) => void;
  onSelectTicket: (ticket: TicketSearchResult) => void;
  onSelectRepository: (repo: RepositorySearchResult) => void;
}

/**
 * SearchResultGroups - Renders search result groups for pods, tickets, and repositories
 */
export function SearchResultGroups({
  pods,
  tickets,
  repositories,
  onSelectPod,
  onSelectTicket,
  onSelectRepository,
}: SearchResultGroupsProps) {
  return (
    <>
      {/* Search Results - Pods */}
      {pods.length > 0 && (
        <Command.Group heading="Pods">
          {pods.map((pod) => (
            <Command.Item
              key={pod.pod_key}
              value={`pod-${pod.pod_key}`}
              className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
              onSelect={() => onSelectPod(pod)}
            >
              <Terminal className="w-4 h-4 text-muted-foreground" />
              <div className="flex-1 min-w-0">
                <div className="text-sm truncate">
                  <code>{getPodDisplayName(pod)}</code>
                </div>
                <div className="text-xs text-muted-foreground">{pod.status}</div>
              </div>
              <ArrowRight className="w-3 h-3 text-muted-foreground" />
            </Command.Item>
          ))}
        </Command.Group>
      )}

      {/* Search Results - Tickets */}
      {tickets.length > 0 && (
        <Command.Group heading="Tickets">
          {tickets.map((ticket) => (
            <Command.Item
              key={ticket.slug}
              value={`ticket-${ticket.slug}`}
              className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
              onSelect={() => onSelectTicket(ticket)}
            >
              <Ticket className="w-4 h-4 text-muted-foreground" />
              <div className="flex-1 min-w-0">
                <div className="text-sm truncate">{ticket.title}</div>
                <div className="text-xs text-muted-foreground">{ticket.slug}</div>
              </div>
              <ArrowRight className="w-3 h-3 text-muted-foreground" />
            </Command.Item>
          ))}
        </Command.Group>
      )}

      {/* Search Results - Repositories */}
      {repositories.length > 0 && (
        <Command.Group heading="Repositories">
          {repositories.map((repo) => (
            <Command.Item
              key={repo.id}
              value={`repo-${repo.id}`}
              className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
              onSelect={() => onSelectRepository(repo)}
            >
              <FolderGit2 className="w-4 h-4 text-muted-foreground" />
              <div className="flex-1 min-w-0">
                <div className="text-sm truncate">{repo.slug}</div>
              </div>
              <ArrowRight className="w-3 h-3 text-muted-foreground" />
            </Command.Item>
          ))}
        </Command.Group>
      )}
    </>
  );
}

export default SearchResultGroups;

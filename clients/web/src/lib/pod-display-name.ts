interface PodDisplayInfo {
  pod_key: string;
  alias?: string | null;
  title?: string | null;
  ticket?: {
    slug?: string;
    title?: string;
  };
  loop?: {
    name?: string;
  };
  agent?: {
    name?: string;
  };
}

/**
 * Get the display name for a Pod.
 *
 * Priority:
 * 1. Alias (user-defined display name)
 * 2. Ticket title (if associated with a ticket)
 * 3. Loop name (if created by a loop job)
 * 4. OSC title (set by terminal applications like Claude Code)
 * 5. Ticket slug fallback
 * 6. Agent name + truncated pod_key
 * 7. Truncated pod_key
 *
 * @param pod - Pod data with optional alias, title, ticket, and loop
 * @param maxLength - Maximum length before truncation (default: 20)
 * @returns Display name string
 */
export function getPodDisplayName(
  pod: PodDisplayInfo,
  maxLength: number = 20
): string {
  const alias = pod.alias?.trim();
  if (alias) {
    if (alias.length > maxLength) {
      return alias.substring(0, maxLength - 3) + "...";
    }
    return alias;
  }

  // Priority 2: Ticket title
  // This takes precedence over OSC title because agents (e.g., Claude Code)
  // overwrite the terminal title with their own name, losing the ticket context.
  const ticketTitle = pod.ticket?.title?.trim();
  if (ticketTitle) {
    if (ticketTitle.length > maxLength) {
      return ticketTitle.substring(0, maxLength - 3) + "...";
    }
    return ticketTitle;
  }

  const loopName = pod.loop?.name?.trim();
  if (loopName) {
    if (loopName.length > maxLength) {
      return loopName.substring(0, maxLength - 3) + "...";
    }
    return loopName;
  }

  const title = pod.title?.trim();
  if (title) {
    if (title.length > maxLength) {
      return title.substring(0, maxLength - 3) + "...";
    }
    return title;
  }

  // Priority 5: Ticket slug fallback
  if (pod.ticket?.slug) {
    return pod.ticket.slug;
  }

  // Priority 6: Agent name + truncated pod_key
  const keyPrefix = getShortPodKey(pod.pod_key);
  const agentName = pod.agent?.name?.trim();
  if (agentName) {
    return `${agentName} (${keyPrefix})`;
  }

  // Priority 7: Fallback to truncated pod_key
  return `${keyPrefix}...`;
}

export function getShortPodKey(podKey: string): string {
  return podKey.substring(0, 8);
}

export function getMentionSafeName(pod: PodDisplayInfo): string {
  if (pod.alias) {
    return pod.alias.replace(/\s+/g, "_");
  }
  if (pod.ticket?.slug) return pod.ticket.slug;
  return getShortPodKey(pod.pod_key);
}

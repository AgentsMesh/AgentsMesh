/**
 * Utility functions for Pod display
 */

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
  // Priority 1: Alias (user-defined display name)
  if (pod.alias) {
    if (pod.alias.length > maxLength) {
      return pod.alias.substring(0, maxLength - 3) + "...";
    }
    return pod.alias;
  }

  // Priority 2: Ticket title
  // This takes precedence over OSC title because agents (e.g., Claude Code)
  // overwrite the terminal title with their own name, losing the ticket context.
  if (pod.ticket?.title) {
    if (pod.ticket.title.length > maxLength) {
      return pod.ticket.title.substring(0, maxLength - 3) + "...";
    }
    return pod.ticket.title;
  }

  // Priority 3: Loop name
  if (pod.loop?.name) {
    if (pod.loop.name.length > maxLength) {
      return pod.loop.name.substring(0, maxLength - 3) + "...";
    }
    return pod.loop.name;
  }

  // Priority 4: OSC title (set by terminal applications)
  if (pod.title) {
    if (pod.title.length > maxLength) {
      return pod.title.substring(0, maxLength - 3) + "...";
    }
    return pod.title;
  }

  // Priority 5: Ticket slug fallback
  if (pod.ticket?.slug) {
    return pod.ticket.slug;
  }

  // Priority 6: Agent name + truncated pod_key
  const keyPrefix = getShortPodKey(pod.pod_key);
  if (pod.agent?.name) {
    return `${pod.agent.name} (${keyPrefix})`;
  }

  // Priority 7: Fallback to truncated pod_key
  return `${keyPrefix}...`;
}

/**
 * Get a standardized short form of a pod key (first 8 characters).
 */
export function getShortPodKey(podKey: string): string {
  return podKey.substring(0, 8);
}

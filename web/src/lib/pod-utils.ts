/**
 * Utility functions for Pod display
 */

interface PodDisplayInfo {
  pod_key: string;
  title?: string | null;
  ticket?: {
    slug?: string;
  };
  agent_type?: {
    name?: string;
  };
}

/**
 * Get the display name for a Pod.
 *
 * Priority:
 * 1. OSC title (set by terminal applications like Claude Code)
 * 2. Ticket slug (if associated with a ticket)
 * 3. Agent type name + truncated pod_key
 *
 * @param pod - Pod data with optional title and ticket
 * @param maxLength - Maximum length before truncation (default: 20)
 * @returns Display name string
 */
export function getPodDisplayName(
  pod: PodDisplayInfo,
  maxLength: number = 20
): string {
  // Priority 1: OSC title
  if (pod.title) {
    if (pod.title.length > maxLength) {
      return pod.title.substring(0, maxLength - 3) + "...";
    }
    return pod.title;
  }

  // Priority 2: Ticket slug
  if (pod.ticket?.slug) {
    return pod.ticket.slug;
  }

  // Priority 3: Agent type + truncated pod_key
  const keyPrefix = pod.pod_key.substring(0, 8);
  if (pod.agent_type?.name) {
    return `${pod.agent_type.name} (${keyPrefix})`;
  }

  // Fallback: just the truncated pod_key
  return `${keyPrefix}...`;
}

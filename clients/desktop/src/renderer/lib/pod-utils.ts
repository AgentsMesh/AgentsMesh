
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

export function getPodDisplayName(
  pod: PodDisplayInfo,
  maxLength: number = 20
): string {
  if (pod.alias) {
    if (pod.alias.length > maxLength) {
      return pod.alias.substring(0, maxLength - 3) + "...";
    }
    return pod.alias;
  }

  // Ticket title overrides OSC title: agents like Claude Code overwrite terminal title, losing ticket context.
  if (pod.ticket?.title) {
    if (pod.ticket.title.length > maxLength) {
      return pod.ticket.title.substring(0, maxLength - 3) + "...";
    }
    return pod.ticket.title;
  }

  if (pod.loop?.name) {
    if (pod.loop.name.length > maxLength) {
      return pod.loop.name.substring(0, maxLength - 3) + "...";
    }
    return pod.loop.name;
  }

  if (pod.title) {
    if (pod.title.length > maxLength) {
      return pod.title.substring(0, maxLength - 3) + "...";
    }
    return pod.title;
  }

  if (pod.ticket?.slug) {
    return pod.ticket.slug;
  }

  const keyPrefix = getShortPodKey(pod.pod_key);
  if (pod.agent?.name) {
    return `${pod.agent.name} (${keyPrefix})`;
  }

  return `${keyPrefix}...`;
}

export function getShortPodKey(podKey: string): string {
  return podKey.substring(0, 8);
}

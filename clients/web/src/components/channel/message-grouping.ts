import type { TransformedMessage } from "./types";

const MERGE_WINDOW_MS = 5 * 60_000;

function senderKey(m: TransformedMessage): string {
  if (m.pod) return `pod:${m.pod.podKey}`;
  if (m.user) return `user:${m.user.id}`;
  return "unknown";
}

function sameDay(a: string, b: string): boolean {
  return new Date(a).toDateString() === new Date(b).toDateString();
}

/**
 * Feishu-style consecutive-sender grouping: a message starts a new group (shows
 * avatar + header) unless it follows another message from the same sender,
 * within a 5-minute window on the same day, with neither being a system or
 * tool-call row. Returns isFirstInGroup keyed by message id.
 */
export function computeGroupFlags(messages: TransformedMessage[]): Map<number, boolean> {
  const flags = new Map<number, boolean>();
  let prev: TransformedMessage | null = null;
  for (const m of messages) {
    const mergeable =
      prev !== null &&
      m.messageType !== "system" &&
      prev.messageType !== "system" &&
      m.content?.kind !== "tool_call" &&
      prev.content?.kind !== "tool_call" &&
      senderKey(m) === senderKey(prev) &&
      sameDay(m.createdAt, prev.createdAt) &&
      new Date(m.createdAt).getTime() - new Date(prev.createdAt).getTime() < MERGE_WINDOW_MS;
    flags.set(m.id, !mergeable);
    prev = m;
  }
  return flags;
}

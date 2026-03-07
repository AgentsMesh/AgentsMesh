/**
 * Pure utility functions for channel message processing.
 * Extracted from MessageInput.tsx to enable reuse and testing.
 */

import type { MentionItem } from "@/hooks/useMentionCandidates";

/** Pod mention info resolved at send time */
export interface MentionedPod {
  podKey: string;
  mentionText: string;
}

/**
 * Parse @mentions from text and match against known pod candidates.
 * Returns deduplicated list of mentioned pods with their full pod keys.
 */
export function parsePodMentions(
  text: string,
  candidates: MentionItem[]
): MentionedPod[] {
  const podCandidates = candidates.filter((c) => c.type === "pod");
  if (podCandidates.length === 0) return [];

  const mentionRegex = /@([\w.\-]+)/g;
  const result: MentionedPod[] = [];
  const seen = new Set<string>();

  let match;
  while ((match = mentionRegex.exec(text)) !== null) {
    const mentionText = match[1];
    const pod = podCandidates.find((p) => p.mentionText === mentionText);
    if (pod && !seen.has(pod.id)) {
      seen.add(pod.id);
      result.push({
        podKey: pod.id.replace("pod:", ""),
        mentionText: pod.mentionText,
      });
    }
  }

  return result;
}

/**
 * Extract the prompt text by stripping pod @mentions from the message.
 * User @mentions are preserved as they may be part of the natural language prompt.
 */
export function extractPromptFromMention(
  content: string,
  mentionedPods: MentionedPod[]
): string {
  let prompt = content;
  for (const pod of mentionedPods) {
    const escaped = pod.mentionText.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    prompt = prompt.replace(new RegExp(`@${escaped}\\s*`, "g"), "");
  }
  return prompt.trim();
}

/**
 * Build a context-aware prompt for a pod, wrapping the raw prompt with
 * channel origin and reply instruction.
 */
export function buildChannelPrompt(
  rawPrompt: string,
  channelName: string
): string {
  return [
    `Message from channel(#${channelName}): ${rawPrompt}`,
    "",
    "If you finish it, please reply to this channel.",
  ].join("\n");
}

/**
 * Extract the @ query at the cursor position.
 * Returns the query string (text after @) and its start index, or null if not in a mention.
 */
export function getMentionQuery(
  text: string,
  cursorPos: number
): { query: string; startIndex: number } | null {
  const textBeforeCursor = text.slice(0, cursorPos);
  const atIndex = textBeforeCursor.lastIndexOf("@");

  if (atIndex === -1) return null;

  // '@' must be at start or preceded by whitespace
  if (atIndex > 0 && !/\s/.test(textBeforeCursor[atIndex - 1])) return null;

  // Extract query: text between '@' and cursor (must not contain whitespace)
  const query = textBeforeCursor.slice(atIndex + 1);
  if (/\s/.test(query)) return null;

  return { query, startIndex: atIndex };
}

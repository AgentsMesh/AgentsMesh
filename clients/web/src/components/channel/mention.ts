/**
 * Pure utility functions for channel mention autocomplete.
 * Structured mention data is collected by MessageInput and validated server-side
 * by MentionValidatorHook — no client-side regex parsing needed.
 */

export function getMentionQuery(
  text: string,
  cursorPos: number
): { query: string; startIndex: number } | null {
  const textBeforeCursor = text.slice(0, cursorPos);
  const atIndex = textBeforeCursor.lastIndexOf("@");

  if (atIndex === -1) return null;

  if (atIndex > 0 && !/\s/.test(textBeforeCursor[atIndex - 1])) return null;

  const query = textBeforeCursor.slice(atIndex + 1);
  if (/\s/.test(query)) return null;

  return { query, startIndex: atIndex };
}

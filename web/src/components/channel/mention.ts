/**
 * Pure utility functions for channel mention autocomplete.
 * Structured mention data is collected by MessageInput and validated server-side
 * by MentionValidatorHook — no client-side regex parsing needed.
 */

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

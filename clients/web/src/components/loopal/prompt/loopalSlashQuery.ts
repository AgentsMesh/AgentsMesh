// Detect a slash-command name being typed at input start. Returns the partial
// name (after `/`) when the cursor sits in a leading `/word` token with no
// space yet — once a space is typed it's an argument (e.g. "/goal ship it"),
// not a command query, so the menu closes.
export function getSlashQuery(
  text: string,
  cursor: number,
): { query: string; startIndex: number } | null {
  if (!text.startsWith("/")) return null;
  const head = text.slice(0, cursor);
  const m = head.match(/^\/(\S*)$/);
  return m ? { query: m[1], startIndex: 0 } : null;
}


export function safeRedirectPath(raw: string | null | undefined): string | null {
  if (typeof raw !== "string" || raw.length === 0) return null;
  if (raw[0] !== "/") return null;
  if (raw.length > 1 && (raw[1] === "/" || raw[1] === "\\")) return null;
  return raw;
}

export function loginUrlWithRedirect(target: string): string {
  const safe = safeRedirectPath(target);
  return safe ? `/login?redirect=${encodeURIComponent(safe)}` : "/login";
}

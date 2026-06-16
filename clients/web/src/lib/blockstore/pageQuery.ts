export function buildPageQuery(
  current: URLSearchParams,
  pageID: string,
  rootBlockID: string | null,
): string {
  const next = new URLSearchParams(Array.from(current.entries()));
  if (pageID === rootBlockID) next.delete("page");
  else next.set("page", pageID);
  const qs = next.toString();
  return qs ? `?${qs}` : "?";
}

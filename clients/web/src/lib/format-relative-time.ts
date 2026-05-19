export function formatRelativeShort(timestamp?: string | null, now = Date.now()): string {
  if (!timestamp) return "";
  const t = new Date(timestamp).getTime();
  if (Number.isNaN(t)) return "";
  const diff = now - t;
  if (diff < 60_000) return "now";
  const min = Math.floor(diff / 60_000);
  if (min < 60) return `${min}m`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h`;
  const day = Math.floor(hr / 24);
  if (day < 7) return `${day}d`;
  const wk = Math.floor(day / 7);
  if (wk < 4) return `${wk}w`;
  const mo = Math.floor(day / 30);
  return `${mo}mo`;
}

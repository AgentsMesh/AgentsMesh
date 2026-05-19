// eslint-disable-next-line @typescript-eslint/no-explicit-any
type TFunc = (key: string, values?: any) => string;

export function formatTimeAgo(dateStr: string, t: TFunc): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return t("common.justNow");
  if (diffMin < 60) return t("common.timeAgoMinutes", { count: diffMin });
  const diffHours = Math.floor(diffMin / 60);
  if (diffHours < 24) return t("common.timeAgoHours", { count: diffHours });
  const diffDays = Math.floor(diffHours / 24);
  return t("common.timeAgoDays", { count: diffDays });
}

export function formatTimeUntil(dateStr: string, t: TFunc): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  if (diffMs < 0) return t("common.overdue");
  if (diffMs < 60000) return t("common.justNow");
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 60) return t("common.timeInMinutes", { count: diffMin });
  const diffHours = Math.floor(diffMin / 60);
  if (diffHours < 24) return t("common.timeInHours", { count: diffHours });
  const diffDays = Math.floor(diffHours / 24);
  return t("common.timeInDays", { count: diffDays });
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const hours = Math.floor(seconds / 3600);
  const min = Math.floor((seconds % 3600) / 60);
  const sec = seconds % 60;
  if (hours > 0) {
    return min > 0 ? `${hours}h ${min}m` : `${hours}h`;
  }
  return sec > 0 ? `${min}m ${sec}s` : `${min}m`;
}

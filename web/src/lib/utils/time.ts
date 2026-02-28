/**
 * Shared time formatting utilities.
 *
 * These functions accept an i18n translate function `t` so that
 * all user-facing strings go through next-intl.
 * The `t` parameter is typed as a simple string-returning function
 * to avoid coupling to a specific next-intl version.
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type TFunc = (key: string, values?: any) => string;

/**
 * Format a past date as a relative time string (e.g. "3m ago", "2h ago").
 * All strings are i18n'd via the t() function.
 */
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

/**
 * Format a future date as a relative time string (e.g. "in 3m", "in 2h").
 * All strings are i18n'd via the t() function.
 */
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

/**
 * Format a duration in seconds as a human-readable string (e.g. "3m 12s").
 */
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

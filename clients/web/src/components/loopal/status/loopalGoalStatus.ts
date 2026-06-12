// Goal status → status-bar tone. Mirrors loopal TUI's goal indicator colors
// (active / paused / complete / infeasible).
const GOAL_STATUS_TONE: Record<string, string> = {
  active: "text-green-600",
  paused: "text-yellow-600",
  complete: "text-blue-600",
  infeasible: "text-red-600",
};

export function goalStatusTone(status: string): string {
  return GOAL_STATUS_TONE[status] ?? "text-muted-foreground";
}

const DESTROYED_POD_STATUSES = new Set(["terminated", "completed", "failed", "error"]);

export function isDestroyedPodStatus(status: string): boolean {
  return DESTROYED_POD_STATUSES.has(status);
}

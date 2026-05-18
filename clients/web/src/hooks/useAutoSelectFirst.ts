"use client";

import { useEffect } from "react";

/**
 * Auto-select the first item of a master list when the URL is missing an id.
 *
 * Caller owns the navigation target — typically `(id) => router.replace(buildUrl(id))`.
 * `onNavigate` MUST be stable (wrap in `useCallback`) — otherwise the effect re-fires
 * every render.
 */
export function useAutoSelectFirst(opts: {
  firstId: number | null;
  idMissing: boolean;
  loading: boolean;
  onNavigate: (id: number) => void;
}): void {
  const { firstId, idMissing, loading, onNavigate } = opts;
  useEffect(() => {
    if (!idMissing || loading || firstId == null) return;
    onNavigate(firstId);
  }, [firstId, idMissing, loading, onNavigate]);
}

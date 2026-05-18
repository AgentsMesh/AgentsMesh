"use client";

import { useEffect } from "react";

/**
 * Auto-select the first item of a master list when the URL is missing an id.
 *
 * Caller owns the navigation target — typically `(id) => router.replace(buildUrl(id))`.
 * `onNavigate` MUST be stable (wrap in `useCallback`) — otherwise the effect re-fires
 * every render.
 *
 * `fetched` gates navigation: until the store has completed at least one
 * fetch, `firstId` may reflect stale cache (e.g. desktop adapter cache
 * carried into the next render before the in-flight fetch resolves). Acting
 * on that pre-fetch value can race past the empty-state branch and jump to
 * a detail page that no longer exists. Wait for the first fetch.
 */
export function useAutoSelectFirst(opts: {
  firstId: number | null;
  idMissing: boolean;
  loading: boolean;
  fetched: boolean;
  onNavigate: (id: number) => void;
}): void {
  const { firstId, idMissing, loading, fetched, onNavigate } = opts;
  useEffect(() => {
    if (!idMissing || loading || !fetched || firstId == null) return;
    onNavigate(firstId);
  }, [firstId, idMissing, loading, fetched, onNavigate]);
}

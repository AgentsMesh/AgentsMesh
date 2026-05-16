import { useEffect, useState } from "react";

const DEFAULT_GRACE_MS = 30_000;

/**
 * Grace window between "this device is registered" and "we're confident the
 * backend runner list reflects that registration". Use to delay orphan-style
 * UI ("Server doesn't recognize this runner") until the local→backend
 * heartbeat has had time to settle.
 *
 * Returns `true` once the grace window elapses; resets whenever `isRegistered`
 * flips back to false (e.g. user logs out / un-registers).
 */
export function useOrphanGrace(
  isRegistered: boolean,
  graceMs: number = DEFAULT_GRACE_MS,
): boolean {
  const [graceExpired, setGraceExpired] = useState(false);

  useEffect(() => {
    if (!isRegistered) {
      setGraceExpired(false);
      return;
    }
    setGraceExpired(false);
    const timer = setTimeout(() => setGraceExpired(true), graceMs);
    return () => clearTimeout(timer);
  }, [isRegistered, graceMs]);

  return graceExpired;
}

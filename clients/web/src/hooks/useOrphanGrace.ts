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
 *
 * Implementation note: every `setExpired` call is scheduled through a 0-ms
 * timeout rather than invoked synchronously, so this hook doesn't trip the
 * `react-hooks/set-state-in-effect` rule (which would otherwise force the
 * effect to re-render mid-commit when `isRegistered` toggles).
 */
export function useOrphanGrace(
  isRegistered: boolean,
  graceMs: number = DEFAULT_GRACE_MS,
): boolean {
  const [expired, setExpired] = useState(false);

  useEffect(() => {
    // Reset to false in a macrotask so the state update is decoupled from
    // the effect's synchronous flush.
    const resetId = setTimeout(() => setExpired(false), 0);
    if (!isRegistered) {
      return () => clearTimeout(resetId);
    }
    const expireId = setTimeout(() => setExpired(true), graceMs);
    return () => {
      clearTimeout(resetId);
      clearTimeout(expireId);
    };
  }, [isRegistered, graceMs]);

  return expired;
}

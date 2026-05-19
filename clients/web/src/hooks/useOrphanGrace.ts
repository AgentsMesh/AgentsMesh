import { useEffect, useState } from "react";

const DEFAULT_GRACE_MS = 30_000;

export function useOrphanGrace(
  isRegistered: boolean,
  graceMs: number = DEFAULT_GRACE_MS,
): boolean {
  const [expired, setExpired] = useState(false);

  useEffect(() => {
    // setExpired through a 0-ms timeout to avoid react-hooks/set-state-in-effect.
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

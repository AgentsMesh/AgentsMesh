export function relayAbortError(): DOMException {
  return new DOMException("Relay subscription aborted", "AbortError");
}

// Endpoint discovery has no cancellation surface. Settle the adapter wait on
// abort, but keep observing the underlying promise so a late rejection is never
// reported as unhandled.
export function waitForRelayWithAbort<T>(
  pending: Promise<T>,
  signal: AbortSignal,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let settled = false;
    function finish(complete: () => void): void {
      if (settled) return;
      settled = true;
      signal.removeEventListener("abort", onAbort);
      complete();
    }
    function onAbort(): void {
      finish(() => reject(relayAbortError()));
    }

    if (signal.aborted) {
      onAbort();
    } else {
      signal.addEventListener("abort", onAbort, { once: true });
      // Defend independently of the caller against an abort racing listener
      // registration (including non-browser AbortSignal implementations).
      if (signal.aborted) onAbort();
    }

    pending.then(
      (value) => finish(() => resolve(value)),
      (error: unknown) => finish(() => reject(error)),
    );
  });
}

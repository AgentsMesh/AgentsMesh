// Node's global fetch (undici) reuses pooled keep-alive sockets with no idle
// health-check: after OS sleep / a network switch the next reuse of a now-dead
// socket throws `TypeError: fetch failed`, and with no timeout a hung connect
// blocks forever. Both surfaced in the renderer as a failed Connect-RPC that
// cascaded into a blank workspace. One retry forces undici onto a fresh socket
// — the actual fix for the stale-socket case — and an abort timeout bounds the
// hang. Shared by both main-process Connect callers (connectCall + the legacy
// JSON aliases) so the retry/timeout policy can't drift between them.

const DEFAULT_TIMEOUT_MS = 30_000;
const RETRY_BACKOFF_MS = 300;

// HTTP 4xx/5xx never reach here — callers check `res.ok` and throw themselves —
// so a TypeError (undici transport failure) or AbortError is the only thing
// that lands in catch, and re-attempting it can never replay a real response.
function isTransientNetworkError(err: unknown): boolean {
  if (err instanceof TypeError) return true;
  return err instanceof Error && err.name === "AbortError";
}

export async function connectFetch(
  url: string,
  init: RequestInit,
  timeoutMs = DEFAULT_TIMEOUT_MS,
): Promise<Response> {
  let lastErr: unknown;
  for (let attempt = 0; attempt < 2; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      return await fetch(url, { ...init, signal: controller.signal });
    } catch (err) {
      lastErr = err;
      if (attempt === 0 && isTransientNetworkError(err)) {
        await new Promise((r) => setTimeout(r, RETRY_BACKOFF_MS));
        continue;
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }
  }
  throw lastErr;
}

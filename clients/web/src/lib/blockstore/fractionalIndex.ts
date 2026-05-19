
const BASE_CHARS = "0123456789abcdefghijklmnopqrstuvwxyz";
const MIN_CHAR = BASE_CHARS[0];
const MAX_CHAR = BASE_CHARS[BASE_CHARS.length - 1];

function indexOfChar(ch: string): number {
  const i = BASE_CHARS.indexOf(ch);
  if (i >= 0) return i;
  // Chars outside BASE_CHARS can appear in legacy or Agent-written keys. Pin
  // them to the nearest endpoint in BASE_CHARS using ASCII order so the
  // fractional walk still produces a key that sorts correctly in Postgres.
  return ch < BASE_CHARS[0] ? 0 : BASE_CHARS.length - 1;
}

export function keyBetween(a: string | null, b: string | null): string {
  if (a !== null && b !== null && a >= b) {
    throw new Error(`keyBetween: expected a < b, got ${a} >= ${b}`);
  }

  const result: string[] = [];
  let i = 0;
  while (true) {
    const lo = a !== null && i < a.length ? indexOfChar(a[i]) : 0;
    const hi = b !== null && i < b.length ? indexOfChar(b[i]) : BASE_CHARS.length - 1;
    if (lo === hi) {
      result.push(BASE_CHARS[lo]);
      i++;
      continue;
    }
    if (hi - lo > 1) {
      const mid = Math.floor((lo + hi) / 2);
      result.push(BASE_CHARS[mid]);
      return result.join("");
    }
    result.push(BASE_CHARS[lo]);
    a = a !== null && i + 1 <= a.length ? a.substring(i + 1) : "";
    b = null;
    i = 0;
  }
}

export function keyAfter(last: string | null): string {
  return keyBetween(last, null);
}

export function keyBefore(first: string | null): string {
  return keyBetween(null, first);
}

export const ORDER_KEY_CHARS = BASE_CHARS;
export const ORDER_KEY_MIN = MIN_CHAR;
export const ORDER_KEY_MAX = MAX_CHAR;

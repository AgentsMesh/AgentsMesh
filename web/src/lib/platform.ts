/**
 * Platform capability detection utilities.
 */

/**
 * Returns true when the primary pointing device is coarse (touch).
 * Used to gate mobile-specific behaviors like manual textarea positioning
 * for IME support in xterm.js.
 *
 * On desktop (mouse/trackpad), xterm.js internally handles IME textarea
 * positioning via its CompositionHelper — no manual sync needed.
 */
export function isTouchPrimaryInput(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(pointer: coarse)").matches;
}

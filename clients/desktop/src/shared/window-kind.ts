// Single source for per-kind window policy (election, reload, layout storage,
// notification seed) — modules read this table instead of re-deriving from
// scattered `kind === "..."` tests. Adding a kind means editing one row.

export type WindowKind = "primary" | "secondary" | "terminal" | "channel";

export interface WindowTraits {
  primaryEligible: boolean;
  reloadOnConfigEdit: boolean;
  // sessionStorage when true — non-primary windows share the file:// localStorage origin.
  ephemeralLayout: boolean;
  seedsNotificationOwner: boolean;
}

const WINDOW_TRAITS: Record<WindowKind, WindowTraits> = {
  primary: { primaryEligible: true, reloadOnConfigEdit: true, ephemeralLayout: false, seedsNotificationOwner: true },
  secondary: { primaryEligible: true, reloadOnConfigEdit: true, ephemeralLayout: true, seedsNotificationOwner: false },
  terminal: { primaryEligible: false, reloadOnConfigEdit: false, ephemeralLayout: true, seedsNotificationOwner: false },
  channel: { primaryEligible: false, reloadOnConfigEdit: true, ephemeralLayout: true, seedsNotificationOwner: false },
};

export function windowTraits(kind: WindowKind): WindowTraits {
  // Fallback guards preload's `as WindowKind` cast — a bogus argv must not crash preload.
  return WINDOW_TRAITS[kind] ?? WINDOW_TRAITS.primary;
}

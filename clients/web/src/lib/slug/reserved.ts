// Reserved slug words that conflict with built-in routes, public pages, or
// system endpoints. MUST stay in sync with backend/pkg/slug/reserved.go.
// See backend/pkg/slug/reserved.go for category breakdown.
export const RESERVED_SLUGS = new Set<string>([
  "auth",
  "forgot-password",
  "invite",
  "login",
  "logout",
  "onboarding",
  "register",
  "reset-password",
  "verify-email",
  "admin",
  "billing",
  "dashboard",
  "settings",
  "support",
  "about",
  "blog",
  "careers",
  "changelog",
  "demo",
  "docs",
  "enterprise",
  "mock-checkout",
  "offline",
  "popout",
  "privacy",
  "terms",
  "api",
  "app",
  "www",
  "organizations",
  "orgs",
  "personal",
  "runners",
  "agents",
  "me",
  "new",
  "null",
  "true",
  "false",
  "undefined",
]);

export function isReservedSlug(s: string): boolean {
  return RESERVED_SLUGS.has(s);
}

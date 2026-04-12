const MOBILE_BREAKPOINT = 768;

export function getDefaultRoute(orgSlug: string): string {
  const isMobile =
    typeof window !== "undefined" && window.innerWidth < MOBILE_BREAKPOINT;
  return `/${orgSlug}/${isMobile ? "channels" : "workspace"}`;
}

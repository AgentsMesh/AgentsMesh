// Light-mode replacement for lib/auth/post-login.ts. Decides where the
// browser should land after login / OAuth / accept-invite / etc, WITHOUT
// touching wasm. The dashboard's wasm bootstrap re-fetches organizations
// when it loads, so we just need to pick a sensible destination.
// Order of preference:
//   1. ?redirect=<safe path> (deep-link restoration; matches login/page.tsx)
//   2. First org's default route (fetched via REST with the Bearer token
//      we just persisted)
//   3. /onboarding (no orgs yet)

import { lightFetch } from "./api-fetch";
import { getDefaultRoute } from "@/lib/default-route";
import { safeRedirectPath } from "@/lib/auth/redirect";

interface Organization {
  id: number;
  name: string;
  slug: string;
}

interface ListOrgsResponse {
  organizations: Organization[];
}

export async function fetchFirstOrgSlug(): Promise<string | null> {
  try {
    const resp = await lightFetch<ListOrgsResponse>(
      "/api/v1/orgs",
      { authenticated: true },
    );
    const orgs = resp?.organizations ?? [];
    return orgs.length > 0 ? orgs[0].slug : null;
  } catch {
    return null;
  }
}

export async function resolvePostLoginUrlLight(opts: {
  redirectParam?: string | null;
}): Promise<string> {
  const redirect = safeRedirectPath(opts.redirectParam ?? null);
  if (redirect) return redirect;
  const slug = await fetchFirstOrgSlug();
  if (slug) return getDefaultRoute(slug);
  return "/onboarding";
}

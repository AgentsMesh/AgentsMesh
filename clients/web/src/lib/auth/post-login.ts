import { getAuthManager, getOrgApiService } from "@/lib/wasm-core";
import { getDefaultRoute } from "@/lib/default-route";
import { safeRedirectPath } from "./redirect";

interface Organization {
  id: number;
  name: string;
  slug: string;
}

export interface ResolvePostLoginUrlOptions {
  redirectParam: string | null;
  setOrganizations: (orgs: Organization[]) => Promise<void> | void;
}

// Single source of truth for "where do we land after a successful login".
// Order of preference:
//   1. `?redirect=<safe path>` from the URL — restores deep-link / popout
//      flows that bounced through /login.
//   2. First org's default route — fetched fresh from OrgApiService and
//      cached via setOrganizations so the destination renders immediately.
//   3. /onboarding — fallback when the user has no org yet.
//
// Org cache priming runs on the redirect path too: the destination
// (e.g. /popout/terminal/...) likely relies on currentOrg being populated.
async function primeOrganizations(
  setOrganizations: (orgs: Organization[]) => Promise<void> | void
): Promise<Organization[]> {
  try {
    const orgsResponse = JSON.parse(await getOrgApiService().list());
    const orgs: Organization[] = orgsResponse.organizations || [];
    if (orgs.length > 0) {
      await setOrganizations(orgs);
      try { await getAuthManager().fetch_organizations(); } catch { /* best effort */ }
    }
    return orgs;
  } catch {
    return [];
  }
}

export async function resolvePostLoginUrl(
  opts: ResolvePostLoginUrlOptions
): Promise<string> {
  const redirect = safeRedirectPath(opts.redirectParam);
  if (redirect) {
    await primeOrganizations(opts.setOrganizations);
    return redirect;
  }
  const orgs = await primeOrganizations(opts.setOrganizations);
  if (orgs.length > 0) return getDefaultRoute(orgs[0].slug);
  return "/onboarding";
}

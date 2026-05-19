import { readCurrentOrg, readOrganizations } from "@/stores/auth";
import { safeRedirectPath } from "@/lib/auth/redirect";

// Reads Rust SSOT cache directly (AuthManager.login populated it via fetch_organizations).
// Desktop service shim doesn't expose OrgApiService — web counterpart at clients/web/src/lib/auth/post-login.ts.
export function navigateAfterLogin(
  push: (url: string) => void,
  redirect: string | null
): void {
  const safe = safeRedirectPath(redirect);
  if (safe) { push(safe); return; }
  const org = readCurrentOrg() ?? readOrganizations()[0];
  push(org ? `/${org.slug}/workspace` : "/onboarding");
}

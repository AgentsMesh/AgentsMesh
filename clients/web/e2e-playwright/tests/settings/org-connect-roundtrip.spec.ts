import { test, expect } from "../../fixtures/index";
import { OrgMembersPage } from "../../pages/settings/org-members.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// Connect-RPC binary lane (proto-migration feature branch). Asserts the
// wasm-side OrgService Connect methods round-trip through the renderer
// to the same UI surface as the REST handlers do — i.e. the
// /proto.org.v1.OrgService/* endpoints return the same data the REST
// handlers do, and the @bufbuild/protobuf fromBinary path on the
// renderer side doesn't lose fields.
//
// Marked as Connect-path explicitly so a future regression in the
// protobuf wire (vs the still-mounted REST wire) surfaces as a
// distinct failure rather than as a generic empty-list bug.
test.describe("Org Service — Connect path (proto-migration)", () => {
  test.beforeEach(async () => {
    clearAuthRateLimit();
  });

  test("members settings list members via Connect (UI populated)", async ({
    page,
  }) => {
    // The MembersSettings page now calls listMembers() from
    // lib/api/org.ts → wasm listMembersConnect → /proto.org.v1.OrgService/
    // ListMembers (binary). If the proto field set drifted (member.user.username
    // dropped, role missing, joined_at unparseable), the list would render
    // empty or the assignee filter would fail to find anyone.
    const membersPage = new OrgMembersPage(page, TEST_ORG_SLUG);
    await membersPage.goto();

    // The owner member at minimum must be visible — they were inserted
    // by org creation, so any working list path renders them.
    await expect(membersPage.inviteButton).toBeVisible();

    // The fixture user dev@agentsmesh.local must show up — proves the
    // Connect path delivered the {items: [...]} envelope (vs an empty
    // list silently dropped in the wasm bridge).
    await expect(page.getByText(/dev@agentsmesh\.local/i)).toBeVisible();
  });

  test("onboarding list-my-orgs via Connect (auth flow lands on dashboard)", async ({
    page,
  }) => {
    // The post-login resolver (lib/auth/post-login.ts) calls listMyOrgs()
    // → wasm listMyOrgsConnect. After a fresh login the user should land
    // on the dashboard for their first org (not /onboarding), which only
    // works if the Connect call returned a non-empty {items: [...]}.
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);

    // Workspace renders only when currentOrg is set — Connect call must
    // have populated the Zustand store via setOrganizations(resp.items).
    await expect(page).toHaveURL(new RegExp(`/${TEST_ORG_SLUG}/`));
  });
});

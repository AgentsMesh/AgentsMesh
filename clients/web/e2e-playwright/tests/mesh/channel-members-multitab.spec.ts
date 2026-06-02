// Multi-tab UI propagation for channel:member_added + :member_removed.
//
// Both tabs open the same channel; we invite user 3 via Connect-RPC and
// verify the member-count badge in the channel header advances in both
// tabs without manual refresh. Then remove and verify it decrements.
//
// Wire-level coverage in tests/realtime/channel-events-wire.spec.ts;
// this spec exercises handler → useChannelStore.patchChannelMemberCount
// → React subscriber chain.
import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { TEST_ORG_SLUG } from "../../helpers/env";

test.describe("Channel members · multi-tab UI propagation", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("tab A invite/remove → tab B member count badge updates", async ({ context, api }) => {
    const cc = await api.connect();

    const stamp = Date.now().toString(36);
    const ch = (await cc.channel.createChannel({
      orgSlug: TEST_ORG_SLUG,
      name: `e2e-members-${stamp}`,
    })) as { id: bigint | number };
    const channelId = ch.id;
    const channelIdStr = String(channelId);

    // dev-org seed has users 1 (dev@) and 3 (dev2@); 1 is the creator,
    // 3 is the invitee for member churn.
    const inviteeId = 3n;

    const tabA = await context.newPage();
    const tabB = await context.newPage();
    await Promise.all([
      tabA.goto(`/${TEST_ORG_SLUG}/channels`),
      tabB.goto(`/${TEST_ORG_SLUG}/channels`),
    ]);

    const channelSelector = `[data-testid="channel-list-item"][data-channel-id="${channelIdStr}"]`;
    await Promise.all([
      expect(tabA.locator(channelSelector)).toHaveCount(1, { timeout: 30_000 }),
      expect(tabB.locator(channelSelector)).toHaveCount(1, { timeout: 30_000 }),
    ]);

    await tabA.locator(channelSelector).scrollIntoViewIfNeeded();
    await tabB.locator(channelSelector).scrollIntoViewIfNeeded();
    await Promise.all([
      tabA.locator(channelSelector).click(),
      tabB.locator(channelSelector).click(),
    ]);

    // member count badge ("1" creator-only baseline) on the channel header.
    const memberBadge = `[data-testid="channel-header-members"]`;
    await Promise.all([
      expect(tabA.locator(memberBadge)).toContainText("1", { timeout: 15_000 }),
      expect(tabB.locator(memberBadge)).toContainText("1", { timeout: 15_000 }),
    ]);

    // EventSubscriptionManager bootstrap: even after the badge renders,
    // the WASM-side Connect-RPC stream needs time to handshake before
    // subscribeAll is live. Events published earlier have no replay
    // buffer. 5000ms covers the slowest CI runners we've observed.
    await tabA.waitForTimeout(5000);

    await cc.channel.inviteChannelMembers({
      orgSlug: TEST_ORG_SLUG, id: channelId, userIds: [inviteeId],
    });

    await Promise.all([
      expect(tabA.locator(memberBadge)).toContainText("2", { timeout: 10_000 }),
      expect(tabB.locator(memberBadge)).toContainText("2", { timeout: 10_000 }),
    ]);

    await cc.channel.removeChannelMember({
      orgSlug: TEST_ORG_SLUG, id: channelId, userId: inviteeId,
    });

    await Promise.all([
      expect(tabA.locator(memberBadge)).toContainText("1", { timeout: 10_000 }),
      expect(tabB.locator(memberBadge)).toContainText("1", { timeout: 10_000 }),
    ]);

    await tabA.close();
    await tabB.close();
    await cc.channel.archiveChannel({ orgSlug: TEST_ORG_SLUG, id: channelId }).catch(() => undefined);
  });
});

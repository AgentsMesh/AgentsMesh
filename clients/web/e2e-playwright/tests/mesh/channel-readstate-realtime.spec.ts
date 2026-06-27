// Multi-user realtime channel read-state: a DIFFERENT user's message drives the
// viewer's sidebar unread badge + @me label without a refresh, and entering the
// channel reveals the "new messages" divider at the first unread.
//
// NOTE: authored-but-unverified — needs the live dev stack:
//   bazel run //deploy/dev:up   then   bazel test //clients/web:e2e
// Two-actor pattern (mirrors channel-members-multitab): the browser tab is the
// primary user dev@ (id 1); a SECOND Connect client logged in as dev2@ (id 3,
// seed.sql) plays the other sender. Live-verify assumptions: the
// SendChannelMessageRequest.mentions map → stored mention, and the
// EventSubscription handshake timing (no replay buffer, hence the 5s settle).
import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { TEST_ORG_SLUG } from "../../helpers/env";

const DEV2 = { email: "dev2@agentsmesh.local", password: "devpass123" };
const DEV2_ID = 3n;
const SELF_USER_ID = "1"; // dev@ — the browser tab + mention target
const BADGE = '[data-testid="channel-unread-badge"]';

async function secondActor(api: { connect: () => Promise<unknown>; login: (e: string, p: string) => Promise<{ token: string }>; connectWithToken: (t: string) => unknown }) {
  const cc = (await api.connect()) as { channel: Record<string, (req: unknown) => Promise<unknown>> };
  const login2 = await api.login(DEV2.email, DEV2.password); // token-switch; `cc` keeps user-1's captured token
  const cc2 = api.connectWithToken(login2.token) as { channel: Record<string, (req: unknown) => Promise<unknown>> };
  return { cc, cc2 };
}

test.describe("Channel read-state · multi-user realtime", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  // NOTE: this asserts badge + @me (the realtime read-state surface). The
  // "new messages" divider is NOT asserted here: it anchors on a known last_read
  // cursor, which a never-opened channel doesn't have yet (the divider LOGIC is
  // covered by useChannelEntryAnchor.test.tsx). Divider-on-realtime for a
  // never-opened channel is a documented edge (no cursor to anchor on).
  test("dev2's @mention raises the unread badge + @me label", async ({ api, page }) => {
    const { cc, cc2 } = await secondActor(api);
    const stamp = Date.now().toString(36);
    const ch = (await cc.channel.createChannel({ orgSlug: TEST_ORG_SLUG, name: `e2e-unread-${stamp}` })) as { id: bigint | number };
    const channelId = ch.id;
    await cc.channel.inviteChannelMembers({ orgSlug: TEST_ORG_SLUG, id: channelId, userIds: [DEV2_ID] });

    // Tab parked on the channel LIST (not inside the channel → stays unread).
    await page.goto(`/${TEST_ORG_SLUG}/channels`);
    const row = `[data-testid="channel-list-item"][data-channel-id="${String(channelId)}"]`;
    await expect(page.locator(row)).toHaveCount(1, { timeout: 30_000 });
    await page.waitForTimeout(5000); // WASM event-stream handshake (no replay buffer)

    await cc2.channel.sendChannelMessage({
      orgSlug: TEST_ORG_SLUG,
      channelId,
      source: "@dev please review",
      mentions: { dev: { entityType: "user", entityKey: SELF_USER_ID } },
    });

    await expect(page.locator(row).locator(BADGE)).toHaveText("1", { timeout: 15_000 });
    await expect(page.locator(row).getByText("@me")).toBeVisible({ timeout: 5_000 });

    await cc.channel.archiveChannel({ orgSlug: TEST_ORG_SLUG, id: channelId }).catch(() => undefined);
  });

  test("dev2's plain messages raise a numeric unread badge", async ({ api, page }) => {
    const { cc, cc2 } = await secondActor(api);
    const stamp = Date.now().toString(36);
    const ch = (await cc.channel.createChannel({ orgSlug: TEST_ORG_SLUG, name: `e2e-unread2-${stamp}` })) as { id: bigint | number };
    const channelId = ch.id;
    await cc.channel.inviteChannelMembers({ orgSlug: TEST_ORG_SLUG, id: channelId, userIds: [DEV2_ID] });

    await page.goto(`/${TEST_ORG_SLUG}/channels`);
    const row = `[data-testid="channel-list-item"][data-channel-id="${String(channelId)}"]`;
    await expect(page.locator(row)).toHaveCount(1, { timeout: 30_000 });
    await page.waitForTimeout(5000);

    await cc2.channel.sendChannelMessage({ orgSlug: TEST_ORG_SLUG, channelId, source: "hello from dev2" });
    await cc2.channel.sendChannelMessage({ orgSlug: TEST_ORG_SLUG, channelId, source: "second one" });

    await expect(page.locator(row).locator(BADGE)).toHaveText("2", { timeout: 15_000 });

    await cc.channel.archiveChannel({ orgSlug: TEST_ORG_SLUG, id: channelId }).catch(() => undefined);
  });
});

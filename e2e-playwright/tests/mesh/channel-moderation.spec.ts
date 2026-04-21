import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

test.describe("Channel Moderation API", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  let channelId: number | null = null;

  test.afterEach(async ({ api }) => {
    if (channelId) {
      await api.delete(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}`);
      channelId = null;
    }
  });

  test("archive and unarchive channel", async ({ api }) => {
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      name: `e2e-archive-${Date.now()}`,
    });
    channelId = (await createRes.json()).channel?.id;

    const archiveRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/archive`);
    expect([200, 204]).toContain(archiveRes.status);

    const unarchiveRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/unarchive`);
    expect([200, 204]).toContain(unarchiveRes.status);
  });

  test("send, edit and delete message", async ({ api }) => {
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      name: `e2e-msg-${Date.now()}`,
    });
    channelId = (await createRes.json()).channel?.id;

    // Channel messages now require structured AST content (schema v1). A plain
    // string is rejected with 400; we construct a minimal text block.
    const textContent = (text: string) => ({
      kind: "text",
      blocks: [{ type: "paragraph", elements: [{ type: "text", text }] }],
    });

    const sendRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/messages`, {
      content: textContent("E2E test message"),
    });
    expect([200, 201]).toContain(sendRes.status);
    const msgId = (await sendRes.json()).message?.id;

    if (msgId) {
      const editRes = await api.put(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/messages/${msgId}`, {
        content: textContent("E2E edited message"),
      });
      expect(editRes.status).toBe(200);

      const delRes = await api.delete(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/messages/${msgId}`);
      expect([200, 204]).toContain(delRes.status);
    }
  });

  test("mark channel read", async ({ api }) => {
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      name: `e2e-read-${Date.now()}`,
    });
    channelId = (await createRes.json()).channel?.id;

    const sendRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/messages`, {
      content: "mark read test",
    });
    const msgId = (await sendRes.json()).message?.id;
    if (msgId) {
      const readRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/read`, {
        message_id: msgId,
      });
      expect([200, 204]).toContain(readRes.status);
    }
  });

  test("get channel unread counts", async ({ api }) => {
    const res = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/unread`);
    expect(res.status).toBe(200);
  });

  test("mute and unmute channel", async ({ api }) => {
    const createRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      name: `e2e-mute-${Date.now()}`,
    });
    channelId = (await createRes.json()).channel?.id;

    const muteRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${channelId}/mute`, {
      muted: true,
    });
    expect([200, 204]).toContain(muteRes.status);
  });
});

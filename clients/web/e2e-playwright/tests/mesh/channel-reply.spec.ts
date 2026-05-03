import { test, expect } from "../../fixtures/index";
import { getApiBaseUrl, TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { textContent } from "../../helpers/test-data";

// Reply-to chains are the only way conversations stay readable past 50+
// messages. Backend stores reply_to in messages.reply_to → ParentMessageID,
// and the REST handler accepts it on send. This spec proves the round-trip:
// send A → send B with reply_to=A → list messages and assert B.reply_to == A.id.
// Without it, a reply field rename or DB-column drop would silently turn
// every reply into a top-level message and nobody would notice until users
// reported "lost context" two weeks later.

const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;

test.describe("Channel reply-to chain", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("reply_to round-trips through both send and list endpoints", async ({ api }) => {
    // Walks the full chain: send parent → send child with reply_to → list
    // messages → assert child.reply_to points back at parent.id. Drop this
    // and a column rename or json-tag drop turns every reply into a
    // top-level message, silently breaking thread context.
    const createRes = await api.post(CHANNELS, {
      name: "E2E ReplyRT " + Date.now(),
      visibility: "public",
    });
    expect(createRes.status).toBe(201);
    const { channel } = await createRes.json();
    try {
      const parentRes = await api.post(`${CHANNELS}/${channel.id}/messages`, {
        content: textContent("the parent"),
      });
      const { message: parent } = await parentRes.json();
      expect(parent.reply_to == null).toBe(true);

      const childRes = await api.post(`${CHANNELS}/${channel.id}/messages`, {
        content: textContent("the reply"),
        reply_to: parent.id,
      });
      const { message: child } = await childRes.json();
      expect(child.reply_to).toBe(parent.id);

      const listRes = await api.get(`${CHANNELS}/${channel.id}/messages`);
      const { messages } = await listRes.json();
      const fetchedChild = (messages as Array<{ id: number; reply_to: number | null }>)
        .find((m) => m.id === child.id);
      expect(fetchedChild?.reply_to).toBe(parent.id);
    } finally {
      await api.post(`${CHANNELS}/${channel.id}/archive`, {});
    }
  });
});

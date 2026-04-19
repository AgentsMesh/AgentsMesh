import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { textContent } from "../../helpers/test-data";

const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;

// Second user in the same org (from seed data)
const SECOND_USER = { email: "dev2@agentsmesh.local", password: "devpass123" };

test.describe("Channel IM Group Model", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  // ── Visibility & Membership ──

  test("create public channel — creator is auto-member", async ({ api }) => {
    const res = await api.post(CHANNELS, {
      name: "E2E Public " + Date.now(),
      visibility: "public",
    });
    expect(res.status).toBe(201);
    const { channel } = await res.json();
    expect(channel.visibility).toBe("public");
    expect(channel.is_member).toBe(true);
    expect(channel.member_count).toBeGreaterThanOrEqual(1);

    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("create private channel with initial members", async ({ api }) => {
    // Get org members to find a second user ID
    const membersRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/members`);
    const { members } = await membersRes.json();
    const otherUser = members?.find((m: { user?: { email: string } }) =>
      m.user?.email !== "dev@agentsmesh.local"
    );

    const body: Record<string, unknown> = {
      name: "E2E Private " + Date.now(),
      visibility: "private",
    };
    if (otherUser?.user_id) {
      body.member_ids = [otherUser.user_id];
    }

    const res = await api.post(CHANNELS, body);
    expect(res.status).toBe(201);
    const { channel } = await res.json();
    expect(channel.visibility).toBe("private");
    expect(channel.is_member).toBe(true);

    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("private channel invisible to non-member", async ({ api }) => {
    // Create private channel as dev user
    const createRes = await api.post(CHANNELS, {
      name: "E2E Invisible " + Date.now(),
      visibility: "private",
    });
    const { channel } = await createRes.json();

    // Login as admin user (different user)
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);

    // Admin should NOT see this private channel in list
    const listRes = await api.getWithToken(CHANNELS, adminToken);
    const { channels } = await listRes.json();
    const found = channels?.find((c: { id: number }) => c.id === channel.id);
    expect(found).toBeUndefined();

    // Admin should get 403 on direct access
    const getRes = await api.getWithToken(`${CHANNELS}/${channel.id}`, adminToken);
    expect(getRes.status).toBe(403);

    // Cleanup — re-login as dev
    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  // ── Join / Leave ──

  test("join public channel", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E Join " + Date.now(),
      visibility: "public",
    });
    const { channel } = await createRes.json();

    // Login as admin
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);

    // Admin joins
    const joinRes = await api.postWithToken(`${CHANNELS}/${channel.id}/join`, {}, adminToken);
    expect(joinRes.status).toBe(200);

    // Verify membership
    const getRes = await api.getWithToken(`${CHANNELS}/${channel.id}`, adminToken);
    const data = await getRes.json();
    expect(data.channel.is_member).toBe(true);

    // Cleanup
    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("cannot join private channel", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E NoJoin " + Date.now(),
      visibility: "private",
    });
    const { channel } = await createRes.json();

    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    const joinRes = await api.postWithToken(`${CHANNELS}/${channel.id}/join`, {}, adminToken);
    expect(joinRes.status).toBe(403);

    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("leave channel", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E Leave " + Date.now(),
      visibility: "public",
    });
    const { channel } = await createRes.json();

    // Admin joins then leaves
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    await api.postWithToken(`${CHANNELS}/${channel.id}/join`, {}, adminToken);
    const leaveRes = await api.postWithToken(`${CHANNELS}/${channel.id}/leave`, {}, adminToken);
    expect(leaveRes.status).toBe(200);

    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("creator cannot leave channel", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E CreatorLeave " + Date.now(),
    });
    const { channel } = await createRes.json();

    const leaveRes = await api.post(`${CHANNELS}/${channel.id}/leave`, {});
    expect(leaveRes.status).toBe(403);

    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  // ── Invite / Remove Members ──

  test("invite and remove member", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E Invite " + Date.now(),
      visibility: "private",
    });
    const { channel } = await createRes.json();

    // Get admin user ID
    const membersRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/members`);
    const { members } = await membersRes.json();
    const admin = members?.find((m: { user?: { email: string } }) =>
      m.user?.email === SECOND_USER.email
    );
    if (!admin?.user_id) { test.skip(); return; }

    // Invite
    const inviteRes = await api.post(`${CHANNELS}/${channel.id}/members`, {
      user_ids: [admin.user_id],
    });
    expect(inviteRes.status).toBe(200);

    // Verify admin can now access
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    const getRes = await api.getWithToken(`${CHANNELS}/${channel.id}`, adminToken);
    expect(getRes.status).toBe(200);

    // Remove (as creator)
    await api.login();
    const removeRes = await api.delete(`${CHANNELS}/${channel.id}/members/${admin.user_id}`);
    expect(removeRes.status).toBe(200);

    // Verify admin can no longer access
    const getRes2 = await api.getWithToken(`${CHANNELS}/${channel.id}`, adminToken);
    expect(getRes2.status).toBe(403);

    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("non-creator cannot remove members", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E NoRemove " + Date.now(),
      visibility: "public",
    });
    const { channel } = await createRes.json();

    // Get creator user ID from members list
    const chMembersRes = await api.get(`${CHANNELS}/${channel.id}/members`);
    const { members } = await chMembersRes.json();
    const creatorId = members?.[0]?.user_id;

    // Admin joins
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    await api.postWithToken(`${CHANNELS}/${channel.id}/join`, {}, adminToken);

    // Admin tries to remove creator — should fail
    const rmRes = await fetch(
      `${(api as any).baseUrl || ""}${CHANNELS}/${channel.id}/members/${creatorId}`,
      { method: "DELETE", headers: { Authorization: `Bearer ${adminToken}` } }
    );
    expect(rmRes.status).toBe(403);

    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  // ── Messages in Private Channel ──

  test("non-member cannot send message to private channel", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E PrivMsg " + Date.now(),
      visibility: "private",
    });
    const { channel } = await createRes.json();

    // Admin is not a member
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    const msgRes = await api.postWithToken(
      `${CHANNELS}/${channel.id}/messages`,
      { content: textContent("should fail") },
      adminToken,
    );
    expect(msgRes.status).toBe(403);

    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  test("member can send and read messages", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E MemberMsg " + Date.now(),
    });
    const { channel } = await createRes.json();

    // Send
    const sendRes = await api.post(`${CHANNELS}/${channel.id}/messages`, {
      content: textContent("Hello from E2E"),
    });
    expect(sendRes.status).toBe(201);
    const { message } = await sendRes.json();
    expect(message.body).toBe("Hello from E2E");

    // Read
    const listRes = await api.get(`${CHANNELS}/${channel.id}/messages`);
    expect(listRes.status).toBe(200);
    const { messages } = await listRes.json();
    expect(messages.length).toBeGreaterThanOrEqual(1);

    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  // ── Edit / Delete Messages ──

  test("edit and delete own message", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E EditDel " + Date.now(),
    });
    const { channel } = await createRes.json();

    const sendRes = await api.post(`${CHANNELS}/${channel.id}/messages`, {
      content: textContent("original"),
    });
    const { message } = await sendRes.json();

    // Edit
    const editRes = await api.put(
      `${CHANNELS}/${channel.id}/messages/${message.id}`,
      { content: textContent("edited") },
    );
    expect(editRes.status).toBe(200);
    const edited = await editRes.json();
    expect(edited.message.body).toBe("edited");

    // Delete
    const delRes = await api.delete(`${CHANNELS}/${channel.id}/messages/${message.id}`);
    expect(delRes.status).toBe(200);

    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  // ── Unread Counts & Mark Read ──

  test("unread counts and mark read", async ({ api }) => {
    const createRes = await api.post(CHANNELS, {
      name: "E2E Unread " + Date.now(),
      visibility: "public",
    });
    const { channel } = await createRes.json();

    // Admin joins
    const adminToken = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    await api.postWithToken(`${CHANNELS}/${channel.id}/join`, {}, adminToken);

    // Creator sends messages
    await api.login();
    const m1 = await api.post(`${CHANNELS}/${channel.id}/messages`, { content: textContent("msg1") });
    const msg1 = await m1.json();
    await api.post(`${CHANNELS}/${channel.id}/messages`, { content: textContent("msg2") });

    // Admin checks unread
    const unreadRes = await api.getWithToken(`${CHANNELS}/unread`, adminToken);
    expect(unreadRes.status).toBe(200);
    const { unread } = await unreadRes.json();
    const count = unread?.[String(channel.id)] || 0;
    expect(count).toBeGreaterThanOrEqual(2);

    // Admin marks read up to msg1
    await api.postWithToken(
      `${CHANNELS}/${channel.id}/read`,
      { message_id: msg1.message.id },
      adminToken,
    );

    // Unread should decrease
    const unread2Res = await api.getWithToken(`${CHANNELS}/unread`, adminToken);
    const { unread: unread2 } = await unread2Res.json();
    const count2 = unread2?.[String(channel.id)] || 0;
    expect(count2).toBeLessThan(count);

    await api.login();
    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });

  // ── ListChannels returns visibility + membership ──

  test("list channels includes visibility and is_member", async ({ api }) => {
    const name = "E2E ListVis " + Date.now();
    const createRes = await api.post(CHANNELS, { name, visibility: "public" });
    const { channel } = await createRes.json();

    const listRes = await api.get(CHANNELS);
    const { channels } = await listRes.json();
    const found = channels?.find((c: { id: number }) => c.id === channel.id);
    expect(found).toBeTruthy();
    expect(found.visibility).toBe("public");
    expect(found.is_member).toBe(true);
    expect(typeof found.member_count).toBe("number");

    await api.post(`${CHANNELS}/${channel.id}/archive`, {});
  });
});

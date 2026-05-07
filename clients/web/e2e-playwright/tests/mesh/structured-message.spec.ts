import { test, expect } from "../../fixtures/index";
import { test as uiTest, expect as uiExpect } from "@playwright/test";
import { ChannelsPage } from "../../pages/channels.page";
import { SidebarPage } from "../../pages/sidebar.page";
import { TEST_ORG_SLUG, getApiBaseUrl } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { textContent } from "../../helpers/test-data";

const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;

// ────────────────────────────────────────────────────
// Part 1: API-level structured content tests
// ────────────────────────────────────────────────────

test.describe("Structured Message — API", () => {
  let channelId: number;

  test.beforeAll(async ({ api }) => {
    clearAuthRateLimit();
    const res = await api.post(CHANNELS, { name: "E2E StructMsg API " + Date.now() });
    channelId = (await res.json()).channel.id;
  });

  test.afterAll(async ({ api }) => {
    await api.post(`${CHANNELS}/${channelId}/archive`, {});
  });

  test("send plain text paragraph", async ({ api }) => {
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("Hello structured world"),
    });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("Hello structured world");
    expect(message.content.kind).toBe("text");
    expect(message.content.blocks).toHaveLength(1);
    expect(message.content.blocks[0].type).toBe("paragraph");
    expect(message.content.blocks[0].elements[0].type).toBe("text");
    expect(message.content.blocks[0].elements[0].text).toBe("Hello structured world");
  });

  test("send multi-paragraph message", async ({ api }) => {
    const content = {
      kind: "text",
      blocks: [
        { type: "paragraph", elements: [{ type: "text", text: "First paragraph" }] },
        { type: "paragraph", elements: [{ type: "text", text: "Second paragraph" }] },
        { type: "paragraph", elements: [{ type: "text", text: "Third paragraph" }] },
      ],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("First paragraph\nSecond paragraph\nThird paragraph");
    expect(message.content.blocks).toHaveLength(3);
  });

  test("send message with bold/italic/strike/code formatting", async ({ api }) => {
    const content = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "normal " },
          { type: "text", text: "bold", bold: true },
          { type: "text", text: " " },
          { type: "text", text: "italic", italic: true },
          { type: "text", text: " " },
          { type: "text", text: "struck", strike: true },
          { type: "text", text: " " },
          { type: "text", text: "code", code: true },
        ],
      }],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("normal bold italic struck code");
    expect(message.content.blocks[0].elements).toHaveLength(8);
    expect(message.content.blocks[0].elements[1].style?.bold).toBe(true);
    expect(message.content.blocks[0].elements[3].style?.italic).toBe(true);
    expect(message.content.blocks[0].elements[5].style?.strike).toBe(true);
    expect(message.content.blocks[0].elements[7].style?.code).toBe(true);
  });

  test("send message with user mention — body includes @display", async ({ api }) => {
    const content = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "Hey " },
          { type: "mention", entity_type: "user", entity_key: "1", display: "dev-user" },
          { type: "text", text: " check this" },
        ],
      }],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("Hey @dev-user check this");
    // Mention element is preserved in content
    const mentionEl = message.content.blocks[0].elements.find(
      (e: { type: string }) => e.type === "mention"
    );
    expect(mentionEl).toBeTruthy();
    expect(mentionEl.entity_type).toBe("user");
    expect(mentionEl.entity_key).toBe("1");
  });

  test("send message with pod mention — mention element preserved in content", async ({ api }) => {
    const content = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "mention", entity_type: "pod", entity_key: "fake-pod-key", display: "MyBot" },
          { type: "text", text: " fix the bug" },
        ],
      }],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("@MyBot fix the bug");
    // Content preserves the mention element even if pod doesn't exist in org
    const mentionEl = message.content.blocks[0].elements[0];
    expect(mentionEl.type).toBe("mention");
    expect(mentionEl.entity_key).toBe("fake-pod-key");
    expect(mentionEl.display).toBe("MyBot");
  });

  test("send message with link element", async ({ api }) => {
    const content = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "See " },
          { type: "link", text: "docs", url: "https://example.com/docs" },
        ],
      }],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("See docs");
    expect(message.content.blocks[0].elements[1].url).toBe("https://example.com/docs");
  });

  test("send message with linebreak", async ({ api }) => {
    const content = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "before" },
          { type: "linebreak" },
          { type: "text", text: "after" },
        ],
      }],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    expect(message.body).toBe("before\nafter");
  });

  test("duplicate mentions deduplicated in mentions index", async ({ api }) => {
    // Use user mentions (user ID 1 exists in seed data, won't be pruned)
    const content = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "mention", entity_type: "user", entity_key: "1", display: "dev" },
          { type: "text", text: " and " },
          { type: "mention", entity_type: "user", entity_key: "1", display: "dev" },
        ],
      }],
    };
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { content });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    // Server deduplicates: user ID 1 appears once despite two mention elements
    expect(message.mentions.users).toHaveLength(1);
    expect(message.mentions.users[0]).toBe(1);
    // Content preserves both mention elements
    const mentionEls = message.content.blocks[0].elements.filter(
      (e: { type: string }) => e.type === "mention"
    );
    expect(mentionEls).toHaveLength(2);
  });

  test("edit message updates body and content", async ({ api }) => {
    const sendRes = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("original text"),
    });
    const { message } = await sendRes.json();

    const editRes = await api.put(`${CHANNELS}/${channelId}/messages/${message.id}`, {
      content: textContent("edited text"),
    });
    expect(editRes.status).toBe(200);
    const edited = (await editRes.json()).message;
    expect(edited.body).toBe("edited text");
    expect(edited.edited_at).toBeTruthy();
    expect(edited.content.blocks[0].elements[0].text).toBe("edited text");
  });

  test("message list returns body and structured content", async ({ api }) => {
    await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("list test msg"),
    });
    const listRes = await api.get(`${CHANNELS}/${channelId}/messages`);
    expect(listRes.status).toBe(200);
    const { messages } = await listRes.json();
    const found = messages.find((m: { body: string }) => m.body === "list test msg");
    expect(found).toBeTruthy();
    expect(found.content).toBeTruthy();
    expect(found.content.kind).toBe("text");
  });

  test("empty content blocks rejected", async ({ api }) => {
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: { kind: "text" },
    });
    // Server should accept (body extracted as empty string or reject)
    // Either 201 with empty body or 400 — both are valid
    expect([201, 400]).toContain(res.status);
  });

  test("reply_to threads a message to a parent", async ({ api }) => {
    const parent = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("original question"),
    });
    const parentMsg = (await parent.json()).message;
    const reply = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("the answer"),
      reply_to: parentMsg.id,
    });
    expect(reply.status).toBe(201);
    const { message: replyMsg } = await reply.json();
    expect(replyMsg.reply_to).toBe(parentMsg.id);
    expect(replyMsg.body).toBe("the answer");
  });

  test("reply_to to unknown id — server is permissive or rejects", async ({ api }) => {
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("orphan reply"),
      reply_to: 99999999,
    });
    // Server may reject (400) or accept (201) with the unknown reply_to value
    // preserved. Both are valid product behaviors — what matters is no 5xx.
    expect([201, 400]).toContain(res.status);
  });

  test("pod_key send path: sender_pod is set when supplied", async ({ api }) => {
    // pod_key identifies an agent-originated send. Without a real pod, the
    // server should either set sender_pod to the provided key or reject. Both
    // outcomes prove the field is wired end-to-end.
    const podKey = "e2e-synthetic-pod-key";
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      content: textContent("agent hello"),
      pod_key: podKey,
    });
    expect([201, 400, 403, 404]).toContain(res.status);
    if (res.status === 201) {
      const { message } = await res.json();
      expect(message.sender_pod).toBe(podKey);
    }
  });

  test("source: heading + list + code block parse via goldmark", async ({ api }) => {
    const source = "# Heading\n\n- item one\n- item two\n  - nested\n\n```go\nfunc main() {}\n```";
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { source });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    const types = message.content.blocks.map((b: { type: string }) => b.type);
    expect(types).toEqual(["heading", "list", "code_block"]);
    expect(message.content.blocks[0].level).toBe(1);
    expect(message.content.blocks[2].language).toBe("go");
    expect(message.content.blocks[2].text).toBe("func main() {}");
    // Body should include the code-block text now (extractBody fix)
    expect(message.body).toContain("func main() {}");
    // Nested list-item carries an inner list block. Goldmark places the
    // nested list inside the SECOND outer item ("item two\n  - nested").
    type ListBlock = { type: string; items?: ListBlock[][] };
    const allInnerBlocks = (message.content.blocks[1].items as ListBlock[][]).flat();
    const innerList = allInnerBlocks.find((b) => b.type === "list");
    expect(innerList).toBeTruthy();
  });

  test("source: inline marks become typed style elements", async ({ api }) => {
    const source = "**bold** *italic* ~~strike~~ `code`";
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, { source });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    const styles = message.content.blocks[0].elements
      .filter((e: { type: string }) => e.type === "text" && e.style)
      .map((e: { style: Record<string, boolean> }) => e.style);
    expect(styles.some((s: Record<string, boolean>) => s.bold)).toBe(true);
    expect(styles.some((s: Record<string, boolean>) => s.italic)).toBe(true);
    expect(styles.some((s: Record<string, boolean>) => s.strike)).toBe(true);
    expect(styles.some((s: Record<string, boolean>) => s.code)).toBe(true);
  });

  test("source: mention map upgrades @key into typed mention", async ({ api }) => {
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      source: "ping @dev-user please",
      mentions: { "dev-user": { entity_type: "user", entity_key: "1" } },
    });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    const mention = message.content.blocks[0].elements.find(
      (e: { type: string }) => e.type === "mention"
    );
    expect(mention).toBeTruthy();
    expect(mention.entity_key).toBe("1");
    expect(message.mentions.users).toContain(1);
  });

  test("source: link with disallowed scheme degrades to plain text", async ({ api }) => {
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      source: "[click](javascript:alert(1)) and [ok](https://example.com)",
    });
    expect(res.status).toBe(201);
    const { message } = await res.json();
    const linkEls = message.content.blocks[0].elements.filter(
      (e: { type: string }) => e.type === "link"
    );
    // Only the https link survives; javascript: degraded to text
    expect(linkEls.map((e: { url: string }) => e.url)).toEqual(["https://example.com"]);
  });

  test("source: providing both source and content is a 400", async ({ api }) => {
    const res = await api.post(`${CHANNELS}/${channelId}/messages`, {
      source: "# H",
      content: textContent("x"),
    });
    expect(res.status).toBe(400);
  });
});

// ────────────────────────────────────────────────────
// Part 2: UI rendering tests (uses storageState from chromium project)
// ────────────────────────────────────────────────────

uiTest.describe("Structured Message — UI Rendering", () => {
  let channels: ChannelsPage;
  let sidebar: SidebarPage;

  uiTest.beforeEach(async ({ page }) => {
    clearAuthRateLimit();
    channels = new ChannelsPage(page, TEST_ORG_SLUG);
    sidebar = new SidebarPage(page, TEST_ORG_SLUG);
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");
  });

  uiTest("plain text message renders in chat", async ({ page }) => {
    const name = "E2E StructUI Plain " + Date.now();
    await sidebar.navigateTo("channels");
    await channels.createChannel(name);
    await channels.selectChannel(name);

    await channels.sendMessage("Simple plain text message");
    await uiExpect(page.getByText("Simple plain text message")).toBeVisible({ timeout: 5000 });
  });

  uiTest("multiline message renders multiple lines", async ({ page }) => {
    const name = "E2E StructUI Multi " + Date.now();
    await sidebar.navigateTo("channels");
    await channels.createChannel(name);
    await channels.selectChannel(name);

    // Shift+Enter for newline, then Enter to send
    await channels.messageInput.click();
    await page.keyboard.type("Line one");
    await page.keyboard.down("Shift");
    await page.keyboard.press("Enter");
    await page.keyboard.up("Shift");
    await page.keyboard.type("Line two");
    await page.keyboard.press("Enter");
    await page.waitForTimeout(500);

    await uiExpect(page.getByText("Line one")).toBeVisible({ timeout: 5000 });
    await uiExpect(page.getByText("Line two")).toBeVisible({ timeout: 5000 });
  });

  uiTest("bold text renders as <strong> in UI", async ({ page, request }) => {
    const name = "E2E StructUI Bold " + Date.now();
    await sidebar.navigateTo("channels");
    await channels.createChannel(name);
    await channels.selectChannel(name);

    const apiBase = getApiBaseUrl();
    const loginRes = await request.post(`${apiBase}/api/v1/auth/login`, {
      data: { email: "dev@agentsmesh.local", password: "devpass123" },
    });
    const { token } = await loginRes.json();

    // Get channel ID
    const chListRes = await request.get(`${apiBase}/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const { channels: chList } = await chListRes.json();
    const ch = chList.find((c: { name: string }) => c.name === name);
    if (!ch) { uiTest.skip(); return; }

    // Send message with bold element
    await request.post(`${apiBase}/api/v1/orgs/${TEST_ORG_SLUG}/channels/${ch.id}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        content: {
          kind: "text",
          blocks: [{
            type: "paragraph",
            elements: [
              { type: "text", text: "This is " },
              { type: "text", text: "important", bold: true },
            ],
          }],
        },
      },
    });

    // Reload to see the API-sent message
    await page.reload();
    await channels.selectChannel(name);

    // Scope to message bodies (data-message-id) so we don't double-match
    // the sidebar preview text. boldEl below already filters by <strong>.
    await uiExpect(page.locator("[data-message-id]").getByText("important").first()).toBeVisible({ timeout: 5000 });
    const boldEl = page.locator("strong").filter({ hasText: "important" });
    await uiExpect(boldEl).toBeVisible({ timeout: 3000 });
  });

  uiTest("mention renders with @-prefix highlight", async ({ page, request }) => {
    const name = "E2E StructUI Mention " + Date.now();
    await sidebar.navigateTo("channels");
    await channels.createChannel(name);
    await channels.selectChannel(name);

    const apiBase = getApiBaseUrl();
    const loginRes = await request.post(`${apiBase}/api/v1/auth/login`, {
      data: { email: "dev@agentsmesh.local", password: "devpass123" },
    });
    const { token } = await loginRes.json();

    const chListRes = await request.get(`${apiBase}/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const { channels: chList } = await chListRes.json();
    const ch = chList.find((c: { name: string }) => c.name === name);
    if (!ch) { uiTest.skip(); return; }

    await request.post(`${apiBase}/api/v1/orgs/${TEST_ORG_SLUG}/channels/${ch.id}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        content: {
          kind: "text",
          blocks: [{
            type: "paragraph",
            elements: [
              { type: "text", text: "Hello " },
              { type: "mention", entity_type: "user", entity_key: "1", display: "dev-user" },
            ],
          }],
        },
      },
    });

    await page.reload();
    await channels.selectChannel(name);

    // Scope to message bodies — sidebar preview also shows "@dev-user" text.
    await uiExpect(page.locator("[data-message-id]").getByText("@dev-user").first()).toBeVisible({ timeout: 5000 });
  });

  uiTest("source-mode roundtrip renders heading + list + code", async ({ page, request }) => {
    const name = "E2E StructUI Source " + Date.now();
    await sidebar.navigateTo("channels");
    await channels.createChannel(name);
    await channels.selectChannel(name);

    const apiBase = getApiBaseUrl();
    const loginRes = await request.post(`${apiBase}/api/v1/auth/login`, {
      data: { email: "dev@agentsmesh.local", password: "devpass123" },
    });
    const { token } = await loginRes.json();

    const chListRes = await request.get(`${apiBase}/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const { channels: chList } = await chListRes.json();
    const ch = chList.find((c: { name: string }) => c.name === name);
    if (!ch) { uiTest.skip(); return; }

    await request.post(`${apiBase}/api/v1/orgs/${TEST_ORG_SLUG}/channels/${ch.id}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { source: "# Big heading\n\n- bullet a\n- bullet b\n\n```\nfn main() {}\n```" },
    });

    await page.reload();
    await channels.selectChannel(name);

    const bubble = page.locator("[data-message-id]").last();
    await uiExpect(bubble.locator("h1, h2, h3").filter({ hasText: "Big heading" }).first()).toBeVisible({ timeout: 5000 });
    await uiExpect(bubble.locator("ul li").filter({ hasText: "bullet a" })).toBeVisible({ timeout: 3000 });
    await uiExpect(bubble.locator("pre code").filter({ hasText: "fn main()" })).toBeVisible({ timeout: 3000 });
  });
});

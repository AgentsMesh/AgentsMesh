import { test, expect } from "../../fixtures/index";
import { getApiBaseUrl, TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { textContent } from "../../helpers/test-data";

// End-to-end realtime contract for channel messages: when user A sends a
// message via REST, every other channel member must see a `channel:message`
// event on their /ws/events socket within a short window. Without this proof
// the production "messages appear instantly" UX is just a hope — the
// hook_event → eventbus → hub → ws pipeline can break silently and the only
// failure mode is "users say chat feels stale", which we don't catch in CI.

const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;

const SECOND_USER = { email: "dev2@agentsmesh.local", password: "devpass123" };

interface ServerEvent {
  type?: string;
  data?: unknown;
}

function wsEventsURL(token: string): string {
  // Backend's auth middleware accepts the JWT via `?token=` query param when
  // the Authorization header isn't usable (the browser WebSocket API can't
  // set custom headers, so the production client uses the same query-param
  // path — see useRealtimeEvents.ts).
  const base = getApiBaseUrl().replace(/^http/, "ws");
  return `${base}/api/v1/orgs/${TEST_ORG_SLUG}/ws/events?token=${encodeURIComponent(token)}`;
}

async function openSocket(token: string): Promise<{
  socket: WebSocket;
  events: ServerEvent[];
  ready: Promise<void>;
}> {
  const sock = new WebSocket(wsEventsURL(token));
  const events: ServerEvent[] = [];
  const ready = new Promise<void>((resolve, reject) => {
    sock.addEventListener("error", (e) => reject(e instanceof Error ? e : new Error("ws error")));
    sock.addEventListener("message", (m) => {
      try {
        // node's WebSocket gives string for text frames; if it's a Blob/
        // ArrayBuffer the test setup is wrong (server only sends text).
        const ev = JSON.parse(typeof m.data === "string" ? m.data : String(m.data)) as ServerEvent;
        events.push(ev);
        // Resolve on `connected` not `open` so we don't race the hub's
        // client-registration step against the first publish.
        if (ev.type === "connected") resolve();
      } catch {
        /* ignore non-json frames */
      }
    });
  });
  return { socket: sock, events, ready };
}

async function waitFor<T>(
  pred: () => T | undefined,
  timeoutMs: number,
  label: string,
): Promise<T> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const v = pred();
    if (v !== undefined) return v;
    await new Promise((r) => setTimeout(r, 50));
  }
  throw new Error(`timeout: ${label}`);
}

test.describe("Channel realtime WS delivery", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("posting a message broadcasts channel:message to other members", async ({ api }) => {
    const membersRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/members`);
    const { members } = await membersRes.json();
    const dev2 = members?.find((m: { user?: { email: string } }) =>
      m.user?.email === SECOND_USER.email,
    );
    if (!dev2?.user_id) { test.skip(); return; }

    const createRes = await api.post(CHANNELS, {
      name: "E2E Realtime " + Date.now(),
      visibility: "private",
      member_ids: [dev2.user_id],
    });
    expect(createRes.status).toBe(201);
    const { channel } = await createRes.json();

    const dev2Token = await api.loginAs(SECOND_USER.email, SECOND_USER.password);
    const { socket, events, ready } = await openSocket(dev2Token);
    try {
      await Promise.race([
        ready,
        new Promise((_, rej) => setTimeout(() => rej(new Error("ws connect timeout")), 5_000)),
      ]);

      // Switch back to dev's session to send. EventChannelMessage is
      // delivered with TargetUserIDs = channel members; dev is also a
      // member but receives nothing on dev2's socket — only dev2's
      // listener should record the event.
      await api.login();
      const marker = `realtime-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const sendRes = await api.post(`${CHANNELS}/${channel.id}/messages`, {
        content: textContent(marker),
      });
      expect(sendRes.status).toBe(201);

      const match = await waitFor(
        () => events.find((e) => {
          if (e.type !== "channel:message") return undefined;
          const raw = typeof e.data === "string" ? JSON.parse(e.data) : e.data;
          if (raw && typeof raw === "object" && "body" in raw && (raw as { body: string }).body === marker) {
            return e;
          }
          return undefined;
        }),
        5_000,
        `expected channel:message with body=${marker}`,
      );
      expect(match.type).toBe("channel:message");
    } finally {
      socket.close();
      await api.login();
      await api.post(`${CHANNELS}/${channel.id}/archive`, {});
    }
  });
});

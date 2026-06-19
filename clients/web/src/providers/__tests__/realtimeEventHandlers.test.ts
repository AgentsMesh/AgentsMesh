import { describe, it, expect, vi, beforeEach } from "vitest";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ApplyIncomingChannelMessageRequestSchema,
  ApplyChannelMessageEditedEventRequestSchema,
} from "@proto/channel_state/v1/mutations_pb";
import { handleChannelEvent, handleInfraEvent } from "../realtimeEventHandlers";
import { useChannelMessageStore } from "@/stores/channel";
import { useAuthStore } from "@/stores/auth";
import { getAuthManager } from "@/lib/wasm-core";
import { useChannelStore } from "@/stores/channel";

const mockUpdateTicketStatus = vi.fn();
const mockRemoveTicket = vi.fn();
const mockFetchTickets = vi.fn();
const mockFetchTicket = vi.fn();
const mockFetchPod = vi.fn();
const mockTicketSetState = vi.fn();
const mockRunnerSetState = vi.fn();

vi.mock("@/stores/pod", () => ({
  usePodStore: { getState: () => ({ pods: [{ id: 1, pod_key: "pk-1" }], fetchPod: mockFetchPod }) },
}));
vi.mock("@/lib/wasm-core", async () => {
  const actual = await vi.importActual<typeof import("@/lib/wasm-core")>("@/lib/wasm-core");
  type Bucket = Map<number, Record<string, unknown>[]>;
  const buckets = (globalThis as unknown as { __channelBuckets?: Bucket }).__channelBuckets ?? new Map();
  const unread = (globalThis as unknown as { __channelUnread?: Record<number, number> }).__channelUnread ?? {};
  const authBox = (globalThis as unknown as { __authBox?: { user: unknown; current_org: unknown; organizations: unknown[] } }).__authBox
    ?? { user: null, current_org: null, organizations: [] };
  (globalThis as unknown as { __channelBuckets: Bucket }).__channelBuckets = buckets;
  (globalThis as unknown as { __channelUnread: Record<number, number> }).__channelUnread = unread;
  (globalThis as unknown as { __authBox: typeof authBox }).__authBox = authBox;
  return {
    ...actual,
    getAuthManager: () => ({
      get_current_user_json: () => (authBox.user ? JSON.stringify(authBox.user) : null),
      get_current_org_json: () => (authBox.current_org ? JSON.stringify(authBox.current_org) : null),
      get_organizations_json: () => JSON.stringify(authBox.organizations),
      apply_session: (sessionJson: string) => {
        try { authBox.user = (JSON.parse(sessionJson) as { user?: unknown }).user ?? null; } catch { /* noop */ }
      },
      set_organizations: (json: string) => {
        try {
          const orgs = JSON.parse(json);
          authBox.organizations = Array.isArray(orgs) ? orgs : [];
          if (authBox.current_org == null && authBox.organizations.length > 0) {
            authBox.current_org = authBox.organizations[0];
          }
        } catch { /* noop */ }
      },
      set_current_org: (json: string) => {
        if (json === "") { authBox.current_org = null; return; }
        try { authBox.current_org = JSON.parse(json); } catch { /* noop */ }
      },
      clear_session: () => { authBox.user = null; authBox.current_org = null; authBox.organizations = []; },
      is_authenticated: () => authBox.user !== null,
      logout: () => Promise.resolve(),
      switch_org: () => {},
    }),
    getPodService: () => ({
      pods_json: () => JSON.stringify([{ id: 1, pod_key: "pk-1" }]),
    }),
    getPodState: () => ({
      pods_json: () => JSON.stringify([{ id: 1, pod_key: "pk-1" }]),
    }),
    getTicketService: () => ({
      current_ticket_json: () => null,
    }),
    getTicketState: () => ({
      current_ticket_json: () => null,
    }),
    getChannelService: () => ({
      apply_incoming_channel_message: (bytes: Uint8Array) => {
        const req = fromBinary(ApplyIncomingChannelMessageRequestSchema, bytes);
        if (!req.message) return false;
        const channelId = Number(req.channelId);
        const msg = {
          id: Number(req.message.id), channel_id: channelId,
          body: req.message.body,
          sender_pod: req.message.senderPod,
          sender_user_id: req.message.senderUserId !== undefined ? Number(req.message.senderUserId) : undefined,
          message_type: req.message.messageType,
          content_json: req.message.contentJson,
          mentions_json: req.message.mentionsJson,
          reply_to: req.message.replyTo !== undefined ? Number(req.message.replyTo) : undefined,
          created_at: req.message.createdAt,
          sender_user: req.message.senderUser ? {
            id: Number(req.message.senderUser.id),
            username: req.message.senderUser.username,
            name: req.message.senderUser.name,
          } : undefined,
          sender_pod_info: req.message.senderPodInfo ? {
            pod_key: req.message.senderPodInfo.podKey,
            alias: req.message.senderPodInfo.alias,
            ...(req.message.senderPodInfo.agent ? { agent: { name: req.message.senderPodInfo.agent.name } } : {}),
          } : undefined,
        };
        const list = buckets.get(channelId) ?? [];
        if (!list.some((m: Record<string, unknown>) => (m as { id: number }).id === msg.id)) list.push(msg);
        buckets.set(channelId, list);
        return true;
      },
      apply_channel_message_edited_event: (bytes: Uint8Array) => {
        const req = fromBinary(ApplyChannelMessageEditedEventRequestSchema, bytes);
        const cid = Number(req.channelId);
        const list = buckets.get(cid) ?? [];
        const idx = list.findIndex((m: Record<string, unknown>) => (m as { id: number }).id === Number(req.messageId));
        if (idx >= 0) list[idx] = { ...list[idx], body: req.body, edited_at: req.editedAt };
        buckets.set(cid, list);
      },
      remove_message: (channelId: bigint, messageId: bigint) => {
        const cid = Number(channelId);
        const mid = Number(messageId);
        buckets.set(cid, (buckets.get(cid) ?? []).filter((m: Record<string, unknown>) => (m as { id: number }).id !== mid));
      },
      get_messages_json: (channelId: bigint) =>
        JSON.stringify({ messages: buckets.get(Number(channelId)) ?? [], has_more: false }),
      increment_unread: (channelId: bigint) => {
        unread[Number(channelId)] = (unread[Number(channelId)] ?? 0) + 1;
      },
      clear_channel_unread: (channelId: bigint) => {
        delete unread[Number(channelId)];
      },
      unread_counts_json: () => JSON.stringify(unread),
    }),
    parseWasmAny: (val: unknown) => (val ? (typeof val === "string" ? JSON.parse(val) : val) : null),
  };
});
vi.mock("@/stores/runner", () => ({
  useRunnerStore: {
    setState: (updater: unknown) => mockRunnerSetState(updater),
    getState: () => ({}),
  },
}));
vi.mock("@/stores/ticket", () => ({
  useTicketStore: {
    setState: (updater: unknown) => mockTicketSetState(updater),
    getState: () => ({
      updateTicketStatusFromEvent: mockUpdateTicketStatus,
      removeTicketFromEvent: mockRemoveTicket,
      fetchTickets: mockFetchTickets,
      fetchTicket: mockFetchTicket,
      currentTicket: null,
    }),
  },
}));

describe("handleChannelEvent", () => {
  beforeEach(() => {
    getAuthManager().apply_session(JSON.stringify({ token: "t", refresh_token: "r", user: { id: 1, email: "u@e.com", username: "u" } }));
    useChannelStore.setState({ selectedChannelId: null, currentChannel: null } as never);
    useChannelMessageStore.setState({ cache: {}, _messagesTick: 0 } as never);
  });

  // Message persistence + the unread/mention business rules now live in Rust
  // Core (ChannelState::on_new_message, covered by channel_state_tests.rs).
  // The JS handler only (a) triggers a React re-read and (b) refetches the
  // channel list from the backend when the current user is added to a new
  // channel that isn't in the cache yet.
  describe("channel:message", () => {
    it("triggers a message-store re-read tick without throwing", () => {
      const before = (useChannelMessageStore.getState() as unknown as { _messagesTick: number })._messagesTick;
      handleChannelEvent({
        type: "channel:message",
        data: { id: 10, channel_id: 1, sender_user_id: 2, body: "hi", message_type: "text", created_at: "2024-01-01T00:00:00Z" },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      });
      const after = (useChannelMessageStore.getState() as unknown as { _messagesTick: number })._messagesTick;
      expect(after).toBeGreaterThan(before);
    });
  });

  describe("channel:member_added", () => {
    it("refetches channel list when the current user is added (new channel appears)", () => {
      const fetchChannels = vi.fn();
      useChannelStore.setState({ fetchChannels, currentChannel: null } as never);
      handleChannelEvent({
        type: "channel:member_added",
        data: { channel_id: 5, user_id: 1, role: "member" }, // user 1 == current user
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "5", timestamp: Date.now(),
      });
      expect(fetchChannels).toHaveBeenCalledWith({ includeArchived: true });
    });

    it("does NOT refetch when another user is added", () => {
      const fetchChannels = vi.fn();
      useChannelStore.setState({ fetchChannels, currentChannel: null } as never);
      handleChannelEvent({
        type: "channel:member_added",
        data: { channel_id: 5, user_id: 99, role: "member" },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "5", timestamp: Date.now(),
      });
      expect(fetchChannels).not.toHaveBeenCalled();
    });
  });
});

describe("handleInfraEvent", () => {
  const baseEvent = { category: "entity" as const, organization_id: 1, entity_type: "runner", entity_id: "1", timestamp: Date.now() };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Runner status now lives in Rust event_dispatch (update_runner_status in
  // runtime.state); the JS handler only bumps the re-read tick.
  it("runner:online bumps tick", () => {
    handleInfraEvent({ type: "runner:online", data: { runner_id: 1, node_id: "n1", status: "online" }, ...baseEvent });
    expect(mockRunnerSetState).toHaveBeenCalled();
  });

  it("runner:offline bumps tick", () => {
    handleInfraEvent({ type: "runner:offline", data: { runner_id: 2, node_id: "n2", status: "offline" }, ...baseEvent });
    expect(mockRunnerSetState).toHaveBeenCalled();
  });

  // Ticket status patch + deletion now live in Rust event_dispatch
  // (update_ticket_status / remove_ticket). The JS handler only bumps the
  // re-read tick and triggers the server refetch for full data.
  it("ticket:status_changed bumps tick + refetches", () => {
    handleInfraEvent({
      type: "ticket:status_changed",
      data: { slug: "DEV-1", status: "in_progress", previous_status: "backlog" },
      ...baseEvent, entity_type: "ticket",
    });
    expect(mockUpdateTicketStatus).not.toHaveBeenCalled();
    expect(mockTicketSetState).toHaveBeenCalled();
    expect(mockFetchTickets).toHaveBeenCalled();
  });

  it("ticket:deleted bumps tick + refetches", () => {
    handleInfraEvent({
      type: "ticket:deleted",
      data: { slug: "DEV-2", status: "" },
      ...baseEvent, entity_type: "ticket",
    });
    expect(mockRemoveTicket).not.toHaveBeenCalled();
    expect(mockTicketSetState).toHaveBeenCalled();
    expect(mockFetchTickets).toHaveBeenCalled();
  });

  it("ticket:created triggers refetch", () => {
    handleInfraEvent({
      type: "ticket:created",
      data: { slug: "DEV-3", status: "backlog" },
      ...baseEvent, entity_type: "ticket",
    });
    expect(mockFetchTickets).toHaveBeenCalled();
  });

  it("mr:created with ticket_slug fetches ticket", () => {
    handleInfraEvent({
      type: "mr:created",
      data: { mr_id: 1, mr_iid: 1, mr_url: "", source_branch: "feat", state: "open", ticket_slug: "DEV-1", repository_id: 1 },
      ...baseEvent,
    });
    expect(mockFetchTicket).toHaveBeenCalledWith("DEV-1");
  });

  it("mr:created with pod_id fetches pod", () => {
    handleInfraEvent({
      type: "mr:created",
      data: { mr_id: 1, mr_iid: 1, mr_url: "", source_branch: "feat", state: "open", pod_id: 1, repository_id: 1 },
      ...baseEvent,
    });
    expect(mockFetchPod).toHaveBeenCalledWith("pk-1");
  });

  it("pipeline:updated with ticket_slug fetches ticket", () => {
    handleInfraEvent({
      type: "pipeline:updated",
      data: { pipeline_id: 1, pipeline_status: "success", ticket_slug: "DEV-1", repository_id: 1 },
      ...baseEvent,
    });
    expect(mockFetchTicket).toHaveBeenCalledWith("DEV-1");
  });

  it("pipeline:updated with pod_id fetches pod", () => {
    handleInfraEvent({
      type: "pipeline:updated",
      data: { pipeline_id: 1, pipeline_status: "success", pod_id: 1, repository_id: 1 },
      ...baseEvent,
    });
    expect(mockFetchPod).toHaveBeenCalledWith("pk-1");
  });
});

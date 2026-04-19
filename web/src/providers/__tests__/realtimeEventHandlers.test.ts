import { describe, it, expect, vi, beforeEach } from "vitest";
import { handleChannelEvent, handleInfraEvent } from "../realtimeEventHandlers";
import { useChannelMessageStore } from "@/stores/channel";
import { useAuthStore } from "@/stores/auth";
import { useChannelStore } from "@/stores/channel";
import type { RealtimeEvent } from "@/lib/realtime";

const mockUpdateRunnerStatus = vi.fn();
const mockUpdateTicketStatus = vi.fn();
const mockRemoveTicket = vi.fn();
const mockFetchTickets = vi.fn();
const mockFetchTicket = vi.fn();
const mockFetchPod = vi.fn();

vi.mock("@/stores/pod", () => ({
  usePodStore: { getState: () => ({ pods: [{ id: 1, pod_key: "pk-1" }], fetchPod: mockFetchPod }) },
}));
vi.mock("@/stores/runner", () => ({
  useRunnerStore: { getState: () => ({ updateRunnerStatus: mockUpdateRunnerStatus }) },
}));
vi.mock("@/stores/ticket", () => ({
  useTicketStore: {
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
    useChannelMessageStore.setState({ cache: {}, unreadCounts: {} });
    useAuthStore.setState({ user: { id: 1 } as never });
    useChannelStore.setState({ selectedChannelId: null } as never);
  });

  describe("channel:message", () => {
    it("adds message to store with body and content", () => {
      const event: RealtimeEvent = {
        type: "channel:message",
        data: {
          id: 10, channel_id: 1,
          sender_user_id: 2, sender_name: "alice",
          message_type: "text",
          body: "hello",
          content: { kind: "text", blocks: [{ type: "paragraph", elements: [{ type: "text", text: "hello" }] }] },
          mentions: { users: [3] },
          created_at: "2024-01-01T00:00:00Z",
        },
        category: "entity",
        organization_id: 1,
        entity_type: "channel",
        entity_id: "1",
        timestamp: Date.now(),
      };

      handleChannelEvent(event);

      const cache = useChannelMessageStore.getState().cache[1];
      expect(cache).toBeDefined();
      expect(cache.messages).toHaveLength(1);
      expect(cache.messages[0].body).toBe("hello");
      expect(cache.messages[0].content).toEqual((event.data as Record<string, unknown>).content);
      expect(cache.messages[0].mentions).toEqual({ users: [3] });
    });

    it("includes sender_pod_info when present", () => {
      const event: RealtimeEvent = {
        type: "channel:message",
        data: {
          id: 11, channel_id: 1,
          sender_pod: "pk-bot",
          sender_pod_info: { pod_key: "pk-bot", alias: "MyBot", agent: { name: "Claude" } },
          message_type: "text",
          body: "agent message",
          created_at: "2024-01-01T00:00:00Z",
        },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      const msg = useChannelMessageStore.getState().cache[1].messages[0];
      expect(msg.sender_pod_info).toEqual({ pod_key: "pk-bot", alias: "MyBot", agent: { name: "Claude" } });
    });

    it("constructs sender_user from sender_name", () => {
      const event: RealtimeEvent = {
        type: "channel:message",
        data: {
          id: 12, channel_id: 1,
          sender_user_id: 5, sender_name: "bob",
          message_type: "text", body: "hi",
          created_at: "2024-01-01T00:00:00Z",
        },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      const msg = useChannelMessageStore.getState().cache[1].messages[0];
      expect(msg.sender_user).toEqual({ id: 5, username: "bob", name: "bob" });
    });

    it("increments unread when message is from another user and not viewing", () => {
      useAuthStore.setState({ user: { id: 1 } as never });
      useChannelStore.setState({ selectedChannelId: 999 } as never);

      const event: RealtimeEvent = {
        type: "channel:message",
        data: {
          id: 13, channel_id: 1, sender_user_id: 2,
          message_type: "text", body: "hi",
          created_at: "2024-01-01T00:00:00Z",
        },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      expect(useChannelMessageStore.getState().unreadCounts[1]).toBe(1);
    });

    it("does NOT increment unread for own message", () => {
      useAuthStore.setState({ user: { id: 1 } as never });

      const event: RealtimeEvent = {
        type: "channel:message",
        data: {
          id: 14, channel_id: 1, sender_user_id: 1,
          message_type: "text", body: "self",
          created_at: "2024-01-01T00:00:00Z",
        },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      expect(useChannelMessageStore.getState().unreadCounts[1]).toBeUndefined();
    });

    it("does NOT increment unread when viewing that channel", () => {
      useChannelStore.setState({ selectedChannelId: 1 } as never);

      const event: RealtimeEvent = {
        type: "channel:message",
        data: {
          id: 15, channel_id: 1, sender_user_id: 2,
          message_type: "text", body: "hi",
          created_at: "2024-01-01T00:00:00Z",
        },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      expect(useChannelMessageStore.getState().unreadCounts[1]).toBeUndefined();
    });
  });

  describe("channel:message_edited", () => {
    it("updates message body and content in store", () => {
      // Pre-populate a message
      useChannelMessageStore.setState({
        cache: {
          1: {
            messages: [{ id: 20, channel_id: 1, body: "old", message_type: "text", created_at: "2024-01-01T00:00:00Z" } as never],
            hasMore: false, loading: false, loadingMore: false, error: null,
          },
        },
      });

      const event: RealtimeEvent = {
        type: "channel:message_edited",
        data: {
          id: 20, channel_id: 1,
          body: "edited",
          content: { kind: "text", blocks: [{ type: "paragraph", elements: [{ type: "text", text: "edited" }] }] },
          mentions: { users: [3] },
          edited_at: "2024-01-02T00:00:00Z",
        },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      const msg = useChannelMessageStore.getState().cache[1].messages[0];
      expect(msg.body).toBe("edited");
      expect((msg as { edited_at: string }).edited_at).toBe("2024-01-02T00:00:00Z");
    });
  });

  describe("channel:message_deleted", () => {
    it("removes message from store", () => {
      useChannelMessageStore.setState({
        cache: {
          1: {
            messages: [
              { id: 30, channel_id: 1, body: "keep", message_type: "text", created_at: "2024-01-01T00:00:00Z" } as never,
              { id: 31, channel_id: 1, body: "delete", message_type: "text", created_at: "2024-01-01T00:00:00Z" } as never,
            ],
            hasMore: false, loading: false, loadingMore: false, error: null,
          },
        },
      });

      const event: RealtimeEvent = {
        type: "channel:message_deleted",
        data: { id: 31, channel_id: 1 },
        category: "entity", organization_id: 1, entity_type: "channel", entity_id: "1", timestamp: Date.now(),
      };

      handleChannelEvent(event);

      const msgs = useChannelMessageStore.getState().cache[1].messages;
      expect(msgs).toHaveLength(1);
      expect(msgs[0].id).toBe(30);
    });
  });
});

describe("handleInfraEvent", () => {
  const baseEvent = { category: "entity" as const, organization_id: 1, entity_type: "runner", entity_id: "1", timestamp: Date.now() };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("runner:online updates runner status", () => {
    handleInfraEvent({ type: "runner:online", data: { runner_id: 1, node_id: "n1", status: "online" }, ...baseEvent });
    expect(mockUpdateRunnerStatus).toHaveBeenCalledWith(1, "online");
  });

  it("runner:offline updates runner status", () => {
    handleInfraEvent({ type: "runner:offline", data: { runner_id: 2, node_id: "n2", status: "offline" }, ...baseEvent });
    expect(mockUpdateRunnerStatus).toHaveBeenCalledWith(2, "offline");
  });

  it("ticket:status_changed updates ticket status", () => {
    handleInfraEvent({
      type: "ticket:status_changed",
      data: { slug: "DEV-1", status: "in_progress", previous_status: "backlog" },
      ...baseEvent, entity_type: "ticket",
    });
    expect(mockUpdateTicketStatus).toHaveBeenCalledWith("DEV-1", "in_progress", "backlog");
    expect(mockFetchTickets).toHaveBeenCalled();
  });

  it("ticket:deleted removes ticket", () => {
    handleInfraEvent({
      type: "ticket:deleted",
      data: { slug: "DEV-2", status: "" },
      ...baseEvent, entity_type: "ticket",
    });
    expect(mockRemoveTicket).toHaveBeenCalledWith("DEV-2");
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

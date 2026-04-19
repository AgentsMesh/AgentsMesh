import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { useChannelStore, useChannelMessageStore } from "../channel";
import { EMPTY_CACHE } from "../channelMessageStore";
import type { ChannelMessage, MessageContent } from "@/lib/api";

type Message = ChannelMessage;

vi.mock("@/lib/api", () => ({
  channelApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    archive: vi.fn(),
    unarchive: vi.fn(),
    getMessages: vi.fn(),
    sendMessage: vi.fn(),
    editMessage: vi.fn(),
    deleteMessage: vi.fn(),
    joinPod: vi.fn(),
    leavePod: vi.fn(),
    markRead: vi.fn(),
    getUnreadCounts: vi.fn(),
    mute: vi.fn(),
  },
}));

import { channelApi } from "@/lib/api";

function textContent(text: string): MessageContent {
  return { kind: "text", blocks: [{ type: "paragraph", elements: [{ type: "text", text }] }] };
}

const mockMessage: Message = {
  id: 1, channel_id: 1, body: "Hello, world!", message_type: "text", created_at: "2024-01-01T00:00:00Z",
};

describe("Channel Message Store", () => {
  const CH = 1;

  beforeEach(() => {
    vi.clearAllMocks();
    useChannelMessageStore.setState({ cache: {}, unreadCounts: {} });
    useChannelStore.setState({ currentChannel: null });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const s = useChannelMessageStore.getState();
      expect(s.cache).toEqual({});
      expect(s.unreadCounts).toEqual({});
    });

    it("should return EMPTY_CACHE for unvisited channels", () => {
      const cache = useChannelMessageStore.getState().cache[999] ?? EMPTY_CACHE;
      expect(cache.messages).toEqual([]);
      expect(cache.hasMore).toBe(false);
    });
  });

  describe("fetchMessages", () => {
    it("should populate cache", async () => {
      vi.mocked(channelApi.getMessages).mockResolvedValue({ messages: [mockMessage], has_more: true });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH); });
      const c = useChannelMessageStore.getState().cache[CH];
      expect(c.messages).toHaveLength(1);
      expect(c.hasMore).toBe(true);
      expect(c.loading).toBe(false);
    });

    it("should prepend on load-more (beforeId)", async () => {
      const existing = { ...mockMessage, id: 10, body: "Existing" };
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [existing], hasMore: true, loading: false, loadingMore: false, error: null } },
      });
      vi.mocked(channelApi.getMessages).mockResolvedValue({ messages: [{ ...mockMessage, id: 5, body: "Older" }], has_more: false });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH, 50, 10); });
      const c = useChannelMessageStore.getState().cache[CH];
      expect(c.messages).toHaveLength(2);
      expect(c.messages[0].body).toBe("Older");
      expect(c.messages[1].body).toBe("Existing");
      expect(c.hasMore).toBe(false);
    });

    it("should replace on refresh (no beforeId)", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [{ ...mockMessage, body: "Old" }], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      vi.mocked(channelApi.getMessages).mockResolvedValue({ messages: [mockMessage], has_more: false });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH); });
      expect(useChannelMessageStore.getState().cache[CH].messages[0].body).toBe("Hello, world!");
    });

    it("should isolate channels", async () => {
      useChannelMessageStore.setState({
        cache: { 2: { messages: [{ ...mockMessage, id: 99, channel_id: 2, body: "Ch2" }], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      vi.mocked(channelApi.getMessages).mockResolvedValue({ messages: [mockMessage], has_more: false });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(1);
      expect(useChannelMessageStore.getState().cache[2].messages[0].body).toBe("Ch2");
    });

    it("should handle error", async () => {
      vi.mocked(channelApi.getMessages).mockRejectedValue({ message: "Fail" });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH); });
      expect((useChannelMessageStore.getState().cache[CH] ?? EMPTY_CACHE).loading).toBe(false);
    });
  });

  describe("sendMessage", () => {
    it("should send and add to cache", async () => {
      vi.mocked(channelApi.sendMessage).mockResolvedValue({ message: mockMessage });
      let result: Message;
      await act(async () => { result = await useChannelMessageStore.getState().sendMessage(CH, textContent("Hello, world!")); });
      expect(result!).toEqual(mockMessage);
      expect(useChannelMessageStore.getState().cache[CH].messages).toContainEqual(mockMessage);
    });

    it("should pass podKey", async () => {
      vi.mocked(channelApi.sendMessage).mockResolvedValue({ message: mockMessage });
      await act(async () => { await useChannelMessageStore.getState().sendMessage(CH, textContent("Hello"), "pod-123"); });
      expect(channelApi.sendMessage).toHaveBeenCalledWith(CH, textContent("Hello"), "pod-123");
    });

    it("should handle error", async () => {
      vi.mocked(channelApi.sendMessage).mockRejectedValue({ message: "Send failed" });
      await expect(act(async () => { await useChannelMessageStore.getState().sendMessage(CH, textContent("Hello")); })).rejects.toEqual({ message: "Send failed" });
    });
  });

  describe("addMessage", () => {
    it("should add to channel cache", () => {
      act(() => { useChannelMessageStore.getState().addMessage(CH, mockMessage); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(1);
    });

    it("should append to existing", () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [{ ...mockMessage, id: 0, body: "First" }], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      act(() => { useChannelMessageStore.getState().addMessage(CH, mockMessage); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(2);
    });

    it("should deduplicate", () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      act(() => { useChannelMessageStore.getState().addMessage(CH, { ...mockMessage }); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(1);
    });

    it("should isolate channels", () => {
      useChannelMessageStore.setState({
        cache: { 2: { messages: [], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      act(() => { useChannelMessageStore.getState().addMessage(CH, mockMessage); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(1);
      expect(useChannelMessageStore.getState().cache[2].messages).toHaveLength(0);
    });
  });

  describe("updateMessage", () => {
    it("should update content", () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      act(() => { useChannelMessageStore.getState().updateMessage(CH, { id: 1, body: "Updated", edited_at: "2024-01-02T00:00:00Z" }); });
      expect(useChannelMessageStore.getState().cache[CH].messages[0].body).toBe("Updated");
    });
  });

  describe("removeMessage", () => {
    it("should remove from cache", () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      act(() => { useChannelMessageStore.getState().removeMessage(CH, 1); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(0);
    });
  });

  describe("unread counts", () => {
    it("should increment", () => {
      act(() => { useChannelMessageStore.getState().incrementUnread(CH); useChannelMessageStore.getState().incrementUnread(CH); });
      expect(useChannelMessageStore.getState().unreadCounts[CH]).toBe(2);
    });

    it("should clear via clearChannelUnread", () => {
      act(() => {
        useChannelMessageStore.getState().incrementUnread(CH);
        useChannelMessageStore.getState().incrementUnread(CH);
      });
      expect(useChannelMessageStore.getState().unreadCounts[CH]).toBe(2);
      act(() => { useChannelMessageStore.getState().clearChannelUnread(CH); });
      expect(useChannelMessageStore.getState().unreadCounts[CH]).toBeUndefined();
    });

    it("clearChannelUnread should no-op for unknown channel", () => {
      act(() => { useChannelMessageStore.getState().clearChannelUnread(999); });
      expect(useChannelMessageStore.getState().unreadCounts).toEqual({});
    });

    it("totalUnreadCount should sum all channels", () => {
      act(() => {
        useChannelMessageStore.getState().incrementUnread(1);
        useChannelMessageStore.getState().incrementUnread(1);
        useChannelMessageStore.getState().incrementUnread(2);
        useChannelMessageStore.getState().incrementUnread(3);
        useChannelMessageStore.getState().incrementUnread(3);
        useChannelMessageStore.getState().incrementUnread(3);
      });
      expect(useChannelMessageStore.getState().totalUnreadCount()).toBe(6);
    });

    it("totalUnreadCount should return 0 when no unread", () => {
      expect(useChannelMessageStore.getState().totalUnreadCount()).toBe(0);
    });

    it("totalUnreadCount should update when channel is cleared", () => {
      act(() => {
        useChannelMessageStore.getState().incrementUnread(1);
        useChannelMessageStore.getState().incrementUnread(2);
        useChannelMessageStore.getState().incrementUnread(2);
      });
      expect(useChannelMessageStore.getState().totalUnreadCount()).toBe(3);
      act(() => { useChannelMessageStore.getState().clearChannelUnread(2); });
      expect(useChannelMessageStore.getState().totalUnreadCount()).toBe(1);
    });
  });

  describe("editMessage", () => {
    it("should update body and content in cache", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      const editedContent = textContent("Edited");
      vi.mocked(channelApi.editMessage).mockResolvedValue({
        message: { ...mockMessage, body: "Edited", content: editedContent, edited_at: "2024-01-02T00:00:00Z", mentions: { pods: ["pk-1"] } },
      });
      await act(async () => { await useChannelMessageStore.getState().editMessage(CH, 1, editedContent); });
      const msg = useChannelMessageStore.getState().cache[CH].messages[0];
      expect(msg.body).toBe("Edited");
      expect(msg.edited_at).toBe("2024-01-02T00:00:00Z");
      expect(msg.mentions).toEqual({ pods: ["pk-1"] });
    });

    it("should handle error", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      vi.mocked(channelApi.editMessage).mockRejectedValue(new Error("Edit failed"));
      await expect(
        act(async () => { await useChannelMessageStore.getState().editMessage(CH, 1, textContent("x")); })
      ).rejects.toThrow("Edit failed");
    });
  });

  describe("deleteMessage", () => {
    it("should remove message from cache", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      vi.mocked(channelApi.deleteMessage).mockResolvedValue({ status: "deleted" });
      await act(async () => { await useChannelMessageStore.getState().deleteMessage(CH, 1); });
      expect(useChannelMessageStore.getState().cache[CH].messages).toHaveLength(0);
    });

    it("should handle error", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      vi.mocked(channelApi.deleteMessage).mockRejectedValue(new Error("Delete failed"));
      await expect(
        act(async () => { await useChannelMessageStore.getState().deleteMessage(CH, 1); })
      ).rejects.toThrow("Delete failed");
    });
  });

  describe("sendMessage sender_user backfill", () => {
    it("should backfill sender_user from auth store when missing", async () => {
      const { useAuthStore } = await import("../auth");
      useAuthStore.setState({ user: { id: 42, username: "alice", name: "Alice", avatar_url: "https://a.co/pic.jpg" } as never });
      const msgWithoutUser: Message = { ...mockMessage, sender_user_id: 42 };
      vi.mocked(channelApi.sendMessage).mockResolvedValue({ message: msgWithoutUser });
      await act(async () => { await useChannelMessageStore.getState().sendMessage(CH, textContent("Hi")); });
      const stored = useChannelMessageStore.getState().cache[CH].messages[0];
      expect(stored.sender_user?.username).toBe("alice");
    });
  });

  describe("addMessage dedup with sender_user merge", () => {
    it("should update sender_user when merging duplicate", () => {
      const existing: Message = { ...mockMessage };
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [existing], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      const withUser: Message = { ...mockMessage, sender_user: { id: 1, username: "bot", name: "Bot" } };
      act(() => { useChannelMessageStore.getState().addMessage(CH, withUser); });
      expect(useChannelMessageStore.getState().cache[CH].messages[0].sender_user?.username).toBe("bot");
    });
  });

  describe("updateMessage with content and mentions", () => {
    it("should update body, content, and mentions", () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      const newContent = textContent("Updated");
      act(() => {
        useChannelMessageStore.getState().updateMessage(CH, {
          id: 1, body: "Updated", content: newContent, mentions: { pods: ["pk-1"] }, edited_at: "2024-01-02T00:00:00Z",
        });
      });
      const msg = useChannelMessageStore.getState().cache[CH].messages[0];
      expect(msg.body).toBe("Updated");
      expect(msg.content).toEqual(newContent);
      expect(msg.mentions).toEqual({ pods: ["pk-1"] });
    });
  });

  describe("fetchUnreadCounts", () => {
    it("should fetch and set unread counts", async () => {
      vi.mocked(channelApi.getUnreadCounts).mockResolvedValue({ unread: { "1": 3, "2": 5 } });
      await act(async () => { await useChannelMessageStore.getState().fetchUnreadCounts(); });
      expect(useChannelMessageStore.getState().unreadCounts[1]).toBe(3);
      expect(useChannelMessageStore.getState().unreadCounts[2]).toBe(5);
    });
  });

  describe("markRead", () => {
    it("should clear unread for channel", async () => {
      useChannelMessageStore.setState({ unreadCounts: { [CH]: 5 } });
      vi.mocked(channelApi.markRead).mockResolvedValue({ status: "ok" });
      await act(async () => { await useChannelMessageStore.getState().markRead(CH, 10); });
      expect(useChannelMessageStore.getState().unreadCounts[CH]).toBeUndefined();
    });
  });

  describe("muteChannel", () => {
    it("should call mute API", async () => {
      vi.mocked(channelApi.mute).mockResolvedValue({ status: "ok" });
      await act(async () => { await useChannelMessageStore.getState().muteChannel(CH, true); });
      expect(channelApi.mute).toHaveBeenCalledWith(CH, true);
    });
  });

  describe("fetchMessages dedup guard", () => {
    it("should skip if already loading", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [], hasMore: false, loading: true, loadingMore: false, error: null } },
      });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH); });
      expect(channelApi.getMessages).not.toHaveBeenCalled();
    });

    it("should skip load-more if already loadingMore", async () => {
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [mockMessage], hasMore: true, loading: false, loadingMore: true, error: null } },
      });
      await act(async () => { await useChannelMessageStore.getState().fetchMessages(CH, 30, 1); });
      expect(channelApi.getMessages).not.toHaveBeenCalled();
    });
  });

  describe("sendMessage WS dedup", () => {
    it("should replace WS message with POST response", async () => {
      // WS message arrives first
      useChannelMessageStore.setState({
        cache: { [CH]: { messages: [{ ...mockMessage, body: "ws-body" }], hasMore: false, loading: false, loadingMore: false, error: null } },
      });
      // POST response with same ID arrives
      vi.mocked(channelApi.sendMessage).mockResolvedValue({ message: { ...mockMessage, body: "post-body" } });
      await act(async () => { await useChannelMessageStore.getState().sendMessage(CH, textContent("post-body")); });
      expect(useChannelMessageStore.getState().cache[CH].messages[0].body).toBe("post-body");
    });
  });

  describe("error resilience", () => {
    it("fetchUnreadCounts should handle error gracefully", async () => {
      vi.mocked(channelApi.getUnreadCounts).mockRejectedValue(new Error("fail"));
      await act(async () => { await useChannelMessageStore.getState().fetchUnreadCounts(); });
      expect(useChannelMessageStore.getState().unreadCounts).toEqual({});
    });

    it("markRead should handle error gracefully", async () => {
      vi.mocked(channelApi.markRead).mockRejectedValue(new Error("fail"));
      useChannelMessageStore.setState({ unreadCounts: { [CH]: 5 } });
      await act(async () => { await useChannelMessageStore.getState().markRead(CH, 10); });
      // unread count should NOT have been cleared since markRead failed
      expect(useChannelMessageStore.getState().unreadCounts[CH]).toBe(5);
    });

    it("muteChannel should propagate error", async () => {
      vi.mocked(channelApi.mute).mockRejectedValue(new Error("fail"));
      await expect(
        act(async () => { await useChannelMessageStore.getState().muteChannel(CH, true); })
      ).rejects.toThrow("fail");
    });
  });

  describe("LRU eviction", () => {
    it("should evict oldest channels when cache exceeds 20", async () => {
      // Populate 21 channels
      for (let i = 1; i <= 21; i++) {
        vi.mocked(channelApi.getMessages).mockResolvedValue({ messages: [{ ...mockMessage, id: i, channel_id: i }], has_more: false });
        await act(async () => { await useChannelMessageStore.getState().fetchMessages(i); });
      }
      const keys = Object.keys(useChannelMessageStore.getState().cache).map(Number);
      expect(keys.length).toBeLessThanOrEqual(20);
      // Channel 21 (most recent) should be present
      expect(useChannelMessageStore.getState().cache[21]).toBeDefined();
    });
  });
});

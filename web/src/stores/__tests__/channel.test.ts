import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { useChannelStore, Channel, Message } from "../channel";

// Mock the channel API
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
    joinPod: vi.fn(),
    leavePod: vi.fn(),
  },
}));

import { channelApi } from "@/lib/api";

const mockChannel: Channel = {
  id: 1,
  name: "general",
  description: "General discussion channel",
  is_archived: false,
  organization_id: 1,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

const mockChannel2: Channel = {
  id: 2,
  name: "dev-chat",
  description: "Development discussion",
  is_archived: false,
  organization_id: 1,
  created_at: "2024-01-02T00:00:00Z",
  updated_at: "2024-01-02T00:00:00Z",
};

const mockMessage: Message = {
  id: 1,
  channel_id: 1,
  content: "Hello, world!",
  message_type: "text",
  created_at: "2024-01-01T00:00:00Z",
};

describe("Channel Store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store to initial state
    useChannelStore.setState({
      channels: [],
      currentChannel: null,
      messages: [],
      loading: false,
      messagesLoading: false,
      error: null,
    });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const state = useChannelStore.getState();

      expect(state.channels).toEqual([]);
      expect(state.currentChannel).toBeNull();
      expect(state.messages).toEqual([]);
      expect(state.loading).toBe(false);
      expect(state.messagesLoading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  describe("fetchChannels", () => {
    it("should fetch channels successfully", async () => {
      vi.mocked(channelApi.list).mockResolvedValue({
        channels: [mockChannel, mockChannel2],
        total: 2,
      });

      await act(async () => {
        await useChannelStore.getState().fetchChannels();
      });

      const state = useChannelStore.getState();
      expect(state.channels).toHaveLength(2);
      expect(state.channels[0].name).toBe("general");
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("should pass filters to API", async () => {
      vi.mocked(channelApi.list).mockResolvedValue({ channels: [], total: 0 });

      await act(async () => {
        await useChannelStore.getState().fetchChannels({ includeArchived: true });
      });

      expect(channelApi.list).toHaveBeenCalledWith({
        include_archived: true,
      });
    });

    it("should handle empty response", async () => {
      vi.mocked(channelApi.list).mockResolvedValue({ channels: undefined as unknown as typeof mockChannel[], total: 0 });

      await act(async () => {
        await useChannelStore.getState().fetchChannels();
      });

      const state = useChannelStore.getState();
      expect(state.channels).toEqual([]);
    });

    it("should handle fetch error", async () => {
      vi.mocked(channelApi.list).mockRejectedValue(new Error("Network error"));

      await act(async () => {
        await useChannelStore.getState().fetchChannels();
      });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });
  });

  describe("fetchChannel", () => {
    it("should fetch single channel successfully", async () => {
      vi.mocked(channelApi.get).mockResolvedValue({ channel: mockChannel });

      await act(async () => {
        await useChannelStore.getState().fetchChannel(1);
      });

      const state = useChannelStore.getState();
      expect(state.currentChannel).toEqual(mockChannel);
      expect(state.loading).toBe(false);
    });

    it("should handle fetch error", async () => {
      vi.mocked(channelApi.get).mockRejectedValue({ message: "Channel not found" });

      await act(async () => {
        await useChannelStore.getState().fetchChannel(999);
      });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Channel not found");
      expect(state.loading).toBe(false);
    });
  });

  describe("createChannel", () => {
    it("should create channel successfully", async () => {
      vi.mocked(channelApi.create).mockResolvedValue({ channel: mockChannel });

      let result: Channel;
      await act(async () => {
        result = await useChannelStore.getState().createChannel({
          name: "general",
          description: "General discussion channel",
        });
      });

      const state = useChannelStore.getState();
      expect(result!).toEqual(mockChannel);
      expect(state.channels).toContainEqual(mockChannel);
      expect(state.loading).toBe(false);
    });

    it("should convert camelCase to snake_case for API", async () => {
      vi.mocked(channelApi.create).mockResolvedValue({ channel: mockChannel });

      await act(async () => {
        await useChannelStore.getState().createChannel({
          name: "test",
          repositoryId: 1,
          ticketId: 2,
        });
      });

      expect(channelApi.create).toHaveBeenCalledWith({
        name: "test",
        description: undefined,
        document: undefined,
        repository_id: 1,
        ticket_id: 2,
      });
    });

    it("should handle create error", async () => {
      vi.mocked(channelApi.create).mockRejectedValue(new Error("Create failed"));

      await expect(
        act(async () => {
          await useChannelStore.getState().createChannel({ name: "test" });
        })
      ).rejects.toThrow("Create failed");

      const state = useChannelStore.getState();
      expect(state.error).toBe("Create failed");
    });
  });

  describe("updateChannel", () => {
    beforeEach(() => {
      useChannelStore.setState({
        channels: [mockChannel],
        currentChannel: mockChannel,
      });
    });

    it("should update channel successfully", async () => {
      const updatedChannel = { ...mockChannel, name: "updated-general" };
      vi.mocked(channelApi.update).mockResolvedValue({ channel: updatedChannel });

      await act(async () => {
        await useChannelStore.getState().updateChannel(1, { name: "updated-general" });
      });

      const state = useChannelStore.getState();
      expect(state.channels[0].name).toBe("updated-general");
      expect(state.currentChannel?.name).toBe("updated-general");
    });

    it("should not update currentChannel if different id", async () => {
      const updatedChannel2 = { ...mockChannel2, name: "updated-dev" };
      vi.mocked(channelApi.update).mockResolvedValue({ channel: updatedChannel2 });

      useChannelStore.setState({ channels: [mockChannel, mockChannel2] });

      await act(async () => {
        await useChannelStore.getState().updateChannel(2, { name: "updated-dev" });
      });

      const state = useChannelStore.getState();
      expect(state.currentChannel?.name).toBe("general"); // unchanged
    });

    it("should handle update error", async () => {
      vi.mocked(channelApi.update).mockRejectedValue({ message: "Update failed" });

      await expect(
        act(async () => {
          await useChannelStore.getState().updateChannel(1, { name: "test" });
        })
      ).rejects.toEqual({ message: "Update failed" });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Update failed");
    });
  });

  describe("archiveChannel", () => {
    beforeEach(() => {
      useChannelStore.setState({
        channels: [mockChannel],
        currentChannel: mockChannel,
      });
    });

    it("should archive channel successfully", async () => {
      vi.mocked(channelApi.archive).mockResolvedValue({ message: "Channel archived" });

      await act(async () => {
        await useChannelStore.getState().archiveChannel(1);
      });

      const state = useChannelStore.getState();
      expect(state.channels[0].is_archived).toBe(true);
      expect(state.currentChannel?.is_archived).toBe(true);
    });

    it("should handle archive error", async () => {
      vi.mocked(channelApi.archive).mockRejectedValue({ message: "Archive failed" });

      await expect(
        act(async () => {
          await useChannelStore.getState().archiveChannel(1);
        })
      ).rejects.toEqual({ message: "Archive failed" });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Archive failed");
    });
  });

  describe("unarchiveChannel", () => {
    beforeEach(() => {
      const archivedChannel = { ...mockChannel, is_archived: true };
      useChannelStore.setState({
        channels: [archivedChannel],
        currentChannel: archivedChannel,
      });
    });

    it("should unarchive channel successfully", async () => {
      vi.mocked(channelApi.unarchive).mockResolvedValue({ message: "Channel unarchived" });

      await act(async () => {
        await useChannelStore.getState().unarchiveChannel(1);
      });

      const state = useChannelStore.getState();
      expect(state.channels[0].is_archived).toBe(false);
      expect(state.currentChannel?.is_archived).toBe(false);
    });

    it("should handle unarchive error", async () => {
      vi.mocked(channelApi.unarchive).mockRejectedValue({ message: "Unarchive failed" });

      await expect(
        act(async () => {
          await useChannelStore.getState().unarchiveChannel(1);
        })
      ).rejects.toEqual({ message: "Unarchive failed" });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Unarchive failed");
    });
  });

  describe("fetchMessages", () => {
    it("should fetch messages successfully", async () => {
      vi.mocked(channelApi.getMessages).mockResolvedValue({
        messages: [mockMessage],
      });

      await act(async () => {
        await useChannelStore.getState().fetchMessages(1);
      });

      const state = useChannelStore.getState();
      expect(state.messages).toHaveLength(1);
      expect(state.messages[0].content).toBe("Hello, world!");
      expect(state.messagesLoading).toBe(false);
    });

    it("should append messages when offset > 0", async () => {
      const existingMessage = { ...mockMessage, id: 0, content: "Previous message" };
      useChannelStore.setState({ messages: [existingMessage] });

      const newMessage = { ...mockMessage, id: 2, content: "New message" };
      vi.mocked(channelApi.getMessages).mockResolvedValue({
        messages: [newMessage],
      });

      await act(async () => {
        await useChannelStore.getState().fetchMessages(1, 50, 1);
      });

      const state = useChannelStore.getState();
      expect(state.messages).toHaveLength(2);
      expect(state.messages[0].content).toBe("Previous message");
      expect(state.messages[1].content).toBe("New message");
    });

    it("should replace messages when offset is 0", async () => {
      useChannelStore.setState({
        messages: [{ ...mockMessage, content: "Old message" }],
      });

      vi.mocked(channelApi.getMessages).mockResolvedValue({
        messages: [mockMessage],
      });

      await act(async () => {
        await useChannelStore.getState().fetchMessages(1, 50, 0);
      });

      const state = useChannelStore.getState();
      expect(state.messages).toHaveLength(1);
      expect(state.messages[0].content).toBe("Hello, world!");
    });

    it("should handle fetch error", async () => {
      vi.mocked(channelApi.getMessages).mockRejectedValue({ message: "Fetch failed" });

      await act(async () => {
        await useChannelStore.getState().fetchMessages(1);
      });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Fetch failed");
      expect(state.messagesLoading).toBe(false);
    });
  });

  describe("sendMessage", () => {
    it("should send message successfully", async () => {
      vi.mocked(channelApi.sendMessage).mockResolvedValue({ message: mockMessage });

      let result: Message;
      await act(async () => {
        result = await useChannelStore.getState().sendMessage(1, "Hello, world!");
      });

      const state = useChannelStore.getState();
      expect(result!).toEqual(mockMessage);
      expect(state.messages).toContainEqual(mockMessage);
    });

    it("should send message with podKey", async () => {
      vi.mocked(channelApi.sendMessage).mockResolvedValue({ message: mockMessage });

      await act(async () => {
        await useChannelStore.getState().sendMessage(1, "Hello", "pod-123");
      });

      expect(channelApi.sendMessage).toHaveBeenCalledWith(1, "Hello", "pod-123");
    });

    it("should handle send error", async () => {
      vi.mocked(channelApi.sendMessage).mockRejectedValue({ message: "Send failed" });

      await expect(
        act(async () => {
          await useChannelStore.getState().sendMessage(1, "Hello");
        })
      ).rejects.toEqual({ message: "Send failed" });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Send failed");
    });
  });

  describe("joinChannel", () => {
    beforeEach(() => {
      useChannelStore.setState({
        channels: [mockChannel],
        currentChannel: mockChannel,
      });
    });

    it("should join channel and refresh", async () => {
      const updatedChannel = {
        ...mockChannel,
        pods: [{ pod_key: "pod-123", status: "running" }],
      };
      vi.mocked(channelApi.joinPod).mockResolvedValue({ message: "Pod joined" });
      vi.mocked(channelApi.get).mockResolvedValue({ channel: updatedChannel });

      await act(async () => {
        await useChannelStore.getState().joinChannel(1, "pod-123");
      });

      const state = useChannelStore.getState();
      expect(state.channels[0].pods).toHaveLength(1);
      expect(state.currentChannel?.pods).toHaveLength(1);
    });

    it("should handle join error", async () => {
      vi.mocked(channelApi.joinPod).mockRejectedValue({ message: "Join failed" });

      await expect(
        act(async () => {
          await useChannelStore.getState().joinChannel(1, "pod-123");
        })
      ).rejects.toEqual({ message: "Join failed" });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Join failed");
    });
  });

  describe("leaveChannel", () => {
    beforeEach(() => {
      const channelWithPods = {
        ...mockChannel,
        pods: [{ pod_key: "pod-123", status: "running" }],
      };
      useChannelStore.setState({
        channels: [channelWithPods],
        currentChannel: channelWithPods,
      });
    });

    it("should leave channel and refresh", async () => {
      const updatedChannel = { ...mockChannel, pods: [] };
      vi.mocked(channelApi.leavePod).mockResolvedValue({ message: "Pod left" });
      vi.mocked(channelApi.get).mockResolvedValue({ channel: updatedChannel });

      await act(async () => {
        await useChannelStore.getState().leaveChannel(1, "pod-123");
      });

      const state = useChannelStore.getState();
      expect(state.channels[0].pods).toHaveLength(0);
    });

    it("should handle leave error", async () => {
      vi.mocked(channelApi.leavePod).mockRejectedValue({ message: "Leave failed" });

      await expect(
        act(async () => {
          await useChannelStore.getState().leaveChannel(1, "pod-123");
        })
      ).rejects.toEqual({ message: "Leave failed" });

      const state = useChannelStore.getState();
      expect(state.error).toBe("Leave failed");
    });
  });

  describe("setCurrentChannel", () => {
    it("should set current channel", () => {
      act(() => {
        useChannelStore.getState().setCurrentChannel(mockChannel);
      });

      const state = useChannelStore.getState();
      expect(state.currentChannel).toEqual(mockChannel);
    });

    it("should clear messages when setting channel", () => {
      useChannelStore.setState({ messages: [mockMessage] });

      act(() => {
        useChannelStore.getState().setCurrentChannel(mockChannel);
      });

      const state = useChannelStore.getState();
      expect(state.messages).toEqual([]);
    });

    it("should set to null", () => {
      useChannelStore.setState({ currentChannel: mockChannel });

      act(() => {
        useChannelStore.getState().setCurrentChannel(null);
      });

      const state = useChannelStore.getState();
      expect(state.currentChannel).toBeNull();
    });
  });

  describe("addMessage", () => {
    it("should add message to list", () => {
      act(() => {
        useChannelStore.getState().addMessage(mockMessage);
      });

      const state = useChannelStore.getState();
      expect(state.messages).toHaveLength(1);
      expect(state.messages[0]).toEqual(mockMessage);
    });

    it("should append to existing messages", () => {
      const existingMessage = { ...mockMessage, id: 0, content: "First" };
      useChannelStore.setState({ messages: [existingMessage] });

      act(() => {
        useChannelStore.getState().addMessage(mockMessage);
      });

      const state = useChannelStore.getState();
      expect(state.messages).toHaveLength(2);
    });
  });

  describe("clearError", () => {
    it("should clear error", () => {
      useChannelStore.setState({ error: "Some error" });

      act(() => {
        useChannelStore.getState().clearError();
      });

      const state = useChannelStore.getState();
      expect(state.error).toBeNull();
    });
  });
});

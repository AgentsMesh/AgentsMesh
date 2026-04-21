import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { useChannelStore, useChannelMessageStore, Channel } from "../channel";
import type { ChannelMessage } from "@/lib/api";
import { getChannelService } from "@/lib/wasm-core";

const svc = () => getChannelService();

const getChannels = (): Channel[] => JSON.parse(svc().channels_json());
const getCurrentChannel = (): Channel | null => {
  const v = svc().current_channel_json();
  return v ? (typeof v === "string" ? JSON.parse(v) : v) : null;
};

const mockChannel: Channel = {
  id: 1,
  name: "general",
  description: "General discussion channel",
  visibility: "public",
  is_archived: false,
  is_member: true,
  member_count: 1,
  organization_id: 1,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

const mockChannel2: Channel = {
  id: 2,
  name: "dev-chat",
  description: "Development discussion",
  visibility: "public",
  is_archived: false,
  is_member: true,
  member_count: 1,
  organization_id: 1,
  created_at: "2024-01-02T00:00:00Z",
  updated_at: "2024-01-02T00:00:00Z",
};

const mockMessage: ChannelMessage = {
  id: 1,
  channel_id: 1,
  body: "Hello, world!",
  message_type: "text",
  created_at: "2024-01-01T00:00:00Z",
};

describe("Channel Store", () => {
  const seedChannels = (channels: Channel[], currentChannel: Channel | null = null) => {
    svc().set_channels(JSON.stringify(channels));
    if (currentChannel) svc().set_current_channel(BigInt(currentChannel.id));
  };

  beforeEach(() => {
    vi.clearAllMocks();
    useChannelStore.setState({
      _tick: 0,
      loading: false,
      channelLoading: false,
      error: null,
      selectedChannelId: null,
      searchQuery: "",
      showArchived: false,
    });
    useChannelMessageStore.setState({ cache: {}, _unreadTick: 0 });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const state = useChannelStore.getState();
      expect(getChannels()).toEqual([]);
      expect(getCurrentChannel()).toBeNull();
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  describe("fetchChannels", () => {
    it("should fetch channels successfully", async () => {
      vi.mocked(svc().fetch_channels).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([mockChannel, mockChannel2]));
        return JSON.stringify({ channels: [mockChannel, mockChannel2] });
      });
      await act(async () => { await useChannelStore.getState().fetchChannels(); });
      expect(getChannels()).toHaveLength(2);
      expect(getChannels()[0].name).toBe("general");
    });

    it("should pass filters to service", async () => {
      vi.mocked(svc().fetch_channels).mockResolvedValue(JSON.stringify({ channels: [] }));
      await act(async () => { await useChannelStore.getState().fetchChannels({ includeArchived: true }); });
      expect(svc().fetch_channels).toHaveBeenCalledWith(true);
    });

    it("should handle fetch error", async () => {
      vi.mocked(svc().fetch_channels).mockRejectedValue(new Error("Network error"));
      await act(async () => { await useChannelStore.getState().fetchChannels(); });
      expect(useChannelStore.getState().error).toBe("Network error");
    });
  });

  describe("fetchChannel", () => {
    it("should fetch single channel", async () => {
      svc().set_channels(JSON.stringify([mockChannel]));
      vi.mocked(svc().fetch_channel).mockImplementation(async () => {
        svc().set_current_channel(BigInt(1));
        return JSON.stringify(mockChannel);
      });
      await act(async () => { await useChannelStore.getState().fetchChannel(1); });
      expect(getCurrentChannel()).toEqual(mockChannel);
    });

    it("should handle error", async () => {
      vi.mocked(svc().fetch_channel).mockRejectedValue({ message: "Channel not found" });
      await act(async () => { await useChannelStore.getState().fetchChannel(999); });
      expect(useChannelStore.getState().error).toBe("Channel not found");
    });
  });

  describe("createChannel", () => {
    it("should create and add to list", async () => {
      vi.mocked(svc().create_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([mockChannel]));
        return JSON.stringify(mockChannel);
      });
      let result: Channel;
      await act(async () => { result = await useChannelStore.getState().createChannel({ name: "general", description: "General discussion channel" }); });
      expect(result!).toEqual(mockChannel);
      expect(getChannels()).toContainEqual(mockChannel);
    });

    it("should convert camelCase to snake_case", async () => {
      vi.mocked(svc().create_channel).mockResolvedValue(JSON.stringify(mockChannel));
      await act(async () => { await useChannelStore.getState().createChannel({ name: "test", repositoryId: 1, ticketSlug: "PROJ-2" }); });
      expect(svc().create_channel).toHaveBeenCalledWith(JSON.stringify({ name: "test", description: undefined, document: undefined, repository_id: 1, ticket_slug: "PROJ-2" }));
    });

    it("should handle error", async () => {
      vi.mocked(svc().create_channel).mockRejectedValue(new Error("Create failed"));
      await expect(act(async () => { await useChannelStore.getState().createChannel({ name: "test" }); })).rejects.toThrow("Create failed");
    });
  });

  describe("updateChannel", () => {
    beforeEach(() => { seedChannels([mockChannel], mockChannel); });

    it("should update channel and currentChannel", async () => {
      const updated = { ...mockChannel, name: "updated" };
      vi.mocked(svc().update_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([updated]));
        return JSON.stringify(updated);
      });
      await act(async () => { await useChannelStore.getState().updateChannel(1, { name: "updated" }); });
      expect(getChannels()[0].name).toBe("updated");
      expect(getCurrentChannel()?.name).toBe("updated");
    });

    it("should not update currentChannel if different id", async () => {
      const updated = { ...mockChannel2, name: "updated-dev" };
      vi.mocked(svc().update_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([mockChannel, updated]));
        return JSON.stringify(updated);
      });
      seedChannels([mockChannel, mockChannel2], mockChannel);
      await act(async () => { await useChannelStore.getState().updateChannel(2, { name: "updated-dev" }); });
      expect(getCurrentChannel()?.name).toBe("general");
    });
  });

  describe("archive/unarchive", () => {
    it("should archive", async () => {
      seedChannels([mockChannel], mockChannel);
      vi.mocked(svc().archive_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([{ ...mockChannel, is_archived: true }]));
      });
      await act(async () => { await useChannelStore.getState().archiveChannel(1); });
      expect(getChannels()[0].is_archived).toBe(true);
    });

    it("should unarchive", async () => {
      const archived = { ...mockChannel, is_archived: true };
      seedChannels([archived], archived);
      vi.mocked(svc().unarchive_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([{ ...mockChannel, is_archived: false }]));
      });
      await act(async () => { await useChannelStore.getState().unarchiveChannel(1); });
      expect(getChannels()[0].is_archived).toBe(false);
    });
  });

  describe("join/leave channel", () => {
    it("should join and refresh", async () => {
      seedChannels([mockChannel], mockChannel);
      const updated = { ...mockChannel, pods: [{ pod_key: "pod-123", status: "running" }] };
      vi.mocked(svc().join_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([updated]));
        return JSON.stringify(updated);
      });
      await act(async () => { await useChannelStore.getState().joinChannel(1, "pod-123"); });
      expect(getChannels()[0].pods).toHaveLength(1);
    });

    it("should leave and refresh", async () => {
      seedChannels([{ ...mockChannel, pods: [{ pod_key: "pod-123", status: "running" }] }], mockChannel);
      const updated = { ...mockChannel, pods: [] };
      vi.mocked(svc().leave_channel).mockImplementation(async () => {
        svc().set_channels(JSON.stringify([updated]));
        return JSON.stringify(updated);
      });
      await act(async () => { await useChannelStore.getState().leaveChannel(1, "pod-123"); });
      expect(getChannels()[0].pods).toHaveLength(0);
    });
  });

  describe("setCurrentChannel", () => {
    it("should set and clear", () => {
      svc().set_channels(JSON.stringify([mockChannel]));
      act(() => { useChannelStore.getState().setCurrentChannel(mockChannel); });
      expect(getCurrentChannel()).toEqual(mockChannel);
      act(() => { useChannelStore.getState().setCurrentChannel(null); });
      expect(getCurrentChannel()).toBeNull();
    });

    it("should not affect per-channel message cache", () => {
      useChannelMessageStore.setState({
        cache: { 1: { loading: false, loadingMore: false, error: "keep-me" } },
      });
      act(() => { useChannelStore.getState().setCurrentChannel(mockChannel); });
      expect(useChannelMessageStore.getState().cache[1]?.error).toBe("keep-me");
    });
  });

  describe("clearError", () => {
    it("should clear", () => {
      useChannelStore.setState({ error: "err" });
      act(() => { useChannelStore.getState().clearError(); });
      expect(useChannelStore.getState().error).toBeNull();
    });
  });
});

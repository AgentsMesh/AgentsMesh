import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { useChannelMessageStore, EMPTY_CACHE, readMessages } from "../channelMessageStore";
import { getChannelService } from "@/lib/wasm-core";
import type { ChannelMessage, MessageContent } from "@/lib/api/channel";

interface MockChannelService {
  fetch_messages: ReturnType<typeof vi.fn>;
  send_message: ReturnType<typeof vi.fn>;
  edit_message: ReturnType<typeof vi.fn>;
  delete_message: ReturnType<typeof vi.fn>;
  fetch_unread_counts: ReturnType<typeof vi.fn>;
  mark_read: ReturnType<typeof vi.fn>;
  mute_channel: ReturnType<typeof vi.fn>;
  increment_unread: ReturnType<typeof vi.fn>;
  clear_channel_unread: ReturnType<typeof vi.fn>;
  on_new_message: ReturnType<typeof vi.fn>;
  update_message_local: ReturnType<typeof vi.fn>;
  remove_message_local: ReturnType<typeof vi.fn>;
  get_messages_json: ReturnType<typeof vi.fn>;
  unread_counts_json: ReturnType<typeof vi.fn>;
}

function svc(): MockChannelService {
  return getChannelService() as unknown as MockChannelService;
}

function makeMsg(id: number, overrides: Partial<ChannelMessage> = {}): ChannelMessage {
  return {
    id,
    channel_id: 42,
    body: `msg ${id}`,
    content: undefined,
    mentions: undefined,
    reply_to: undefined,
    sender_user_id: 1,
    message_type: "text",
    created_at: "2026-04-20T00:00:00Z",
    ...overrides,
  } as ChannelMessage;
}

function mockSuccess(messages: ChannelMessage[], hasMore = false) {
  svc().fetch_messages.mockResolvedValue(undefined);
  svc().get_messages_json.mockReturnValue(JSON.stringify({ messages, has_more: hasMore }));
}

beforeEach(() => {
  useChannelMessageStore.setState({ cache: {}, _unreadTick: 0 });
  Object.assign(getChannelService(), {
    fetch_messages: vi.fn().mockResolvedValue(undefined),
    send_message: vi.fn().mockResolvedValue(JSON.stringify(makeMsg(100))),
    edit_message: vi.fn().mockResolvedValue(undefined),
    delete_message: vi.fn().mockResolvedValue(undefined),
    fetch_unread_counts: vi.fn().mockResolvedValue(undefined),
    mark_read: vi.fn().mockResolvedValue(undefined),
    mute_channel: vi.fn().mockResolvedValue(undefined),
    increment_unread: vi.fn(),
    clear_channel_unread: vi.fn(),
    on_new_message: vi.fn(),
    update_message_local: vi.fn(),
    remove_message_local: vi.fn(),
    get_messages_json: vi.fn().mockReturnValue(JSON.stringify({ messages: [], has_more: false })),
    unread_counts_json: vi.fn().mockReturnValue("{}"),
    total_unread_count: vi.fn().mockReturnValue(0),
    get_unread_count: vi.fn().mockReturnValue(0),
  });
});

describe("useChannelMessageStore — WASM integration", () => {
  it("fetchMessages: calls ChannelService.fetch_messages and syncs cache", async () => {
    const msgs = [makeMsg(1), makeMsg(2)];
    mockSuccess(msgs, true);

    await act(async () => {
      await useChannelMessageStore.getState().fetchMessages(42, 20);
    });

    expect(svc().fetch_messages).toHaveBeenCalledWith(BigInt(42), 20, undefined);
    const cache = useChannelMessageStore.getState().cache[42] ?? EMPTY_CACHE;
    const view = readMessages(42);
    expect(view.messages).toHaveLength(2);
    expect(view.hasMore).toBe(true);
    expect(cache.loading).toBe(false);
  });

  it("fetchMessages: records error on failure", async () => {
    svc().fetch_messages.mockRejectedValue(new Error("boom"));
    await act(async () => {
      await useChannelMessageStore.getState().fetchMessages(42);
    });
    expect(useChannelMessageStore.getState().cache[42]?.error).toBe("boom");
    expect(useChannelMessageStore.getState().cache[42]?.loading).toBe(false);
  });

  it("sendMessage: forwards content + podKey and refreshes cache", async () => {
    const content: MessageContent = { schema_version: 1, kind: "ast", blocks: [] };
    const returned = makeMsg(100, { body: "hi" });
    svc().send_message.mockResolvedValue(JSON.stringify(returned));
    svc().get_messages_json.mockReturnValue(JSON.stringify({ messages: [returned], has_more: false }));

    let result: ChannelMessage | undefined;
    await act(async () => {
      result = await useChannelMessageStore.getState().sendMessage(42, content, "pod-abc");
    });

    const [chanArg, bodyArg] = svc().send_message.mock.calls[0];
    expect(chanArg).toBe(BigInt(42));
    expect(JSON.parse(bodyArg)).toEqual({ content, pod_key: "pod-abc" });
    expect(result?.id).toBe(100);
    expect(readMessages(42).messages).toHaveLength(1);
  });

  it("editMessage: JSON-encodes MessageContent and calls edit_message", async () => {
    const content: MessageContent = { schema_version: 1, kind: "ast", blocks: [] };
    await act(async () => {
      await useChannelMessageStore.getState().editMessage(42, 7, content);
    });
    expect(svc().edit_message).toHaveBeenCalledWith(BigInt(42), BigInt(7), JSON.stringify(content));
  });

  it("deleteMessage: calls delete_message and re-syncs cache", async () => {
    await act(async () => {
      await useChannelMessageStore.getState().deleteMessage(42, 5);
    });
    expect(svc().delete_message).toHaveBeenCalledWith(BigInt(42), BigInt(5));
  });

  it("addMessage: dispatches to WASM on_new_message", () => {
    const msg = makeMsg(9);
    useChannelMessageStore.getState().addMessage(42, msg);
    expect(svc().on_new_message).toHaveBeenCalledWith(JSON.stringify(msg));
  });

  it("fetchUnreadCounts: bumps tick so selectors re-read unread_counts_json", async () => {
    svc().unread_counts_json.mockReturnValue(JSON.stringify({ 1: 3, 2: 5 }));
    await act(async () => {
      await useChannelMessageStore.getState().fetchUnreadCounts();
    });
    expect(svc().fetch_unread_counts).toHaveBeenCalled();
    expect(useChannelMessageStore.getState()._unreadTick).toBeGreaterThan(0);
  });

  it("markRead: calls mark_read and bumps unread tick", async () => {
    await act(async () => {
      await useChannelMessageStore.getState().markRead(42, 99);
    });
    expect(svc().mark_read).toHaveBeenCalledWith(BigInt(42), BigInt(99));
    expect(useChannelMessageStore.getState()._unreadTick).toBeGreaterThan(0);
  });

  it("muteChannel: forwards muted flag", async () => {
    await act(async () => {
      await useChannelMessageStore.getState().muteChannel(42, true);
    });
    expect(svc().mute_channel).toHaveBeenCalledWith(BigInt(42), true);
  });

  it("incrementUnread: dispatches to WASM and bumps tick (no JS mirror)", () => {
    const before = useChannelMessageStore.getState()._unreadTick;
    useChannelMessageStore.getState().incrementUnread(42);
    expect(svc().increment_unread).toHaveBeenCalledWith(BigInt(42));
    expect(useChannelMessageStore.getState()._unreadTick).toBe(before + 1);
  });

  it("clearChannelUnread: dispatches to WASM and bumps tick", () => {
    const before = useChannelMessageStore.getState()._unreadTick;
    useChannelMessageStore.getState().clearChannelUnread(42);
    expect(svc().clear_channel_unread).toHaveBeenCalledWith(BigInt(42));
    expect(useChannelMessageStore.getState()._unreadTick).toBe(before + 1);
  });
});

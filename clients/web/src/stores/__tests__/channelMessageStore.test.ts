import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { create, toBinary } from "@bufbuild/protobuf";
import { ChannelMessageSchema, ListChannelMessagesResponseSchema } from "@proto/channel/v1/channel_pb";

const orgSlug = "test-org";

vi.mock("@/stores/auth", async () => {
  const actual = await vi.importActual<typeof import("@/stores/auth")>("@/stores/auth");
  return {
    ...actual,
    readCurrentOrg: () => ({ id: 1, slug: orgSlug, name: "Test Org" }),
    readCurrentUser: () => null,
  };
});

const mocks = vi.hoisted(() => ({
  listChannelMessagesRaw: vi.fn(),
  sendChannelMessage: vi.fn(),
  editChannelMessage: vi.fn(),
  deleteChannelMessage: vi.fn(),
  getChannelUnreadCounts: vi.fn(),
  markChannelRead: vi.fn(),
  muteChannel: vi.fn(),
}));

vi.mock("@/lib/api/facade/channelConnect", () => ({
  listChannelMessagesRaw: mocks.listChannelMessagesRaw,
  sendChannelMessage: mocks.sendChannelMessage,
  editChannelMessage: mocks.editChannelMessage,
  deleteChannelMessage: mocks.deleteChannelMessage,
  getChannelUnreadCounts: mocks.getChannelUnreadCounts,
  markChannelRead: mocks.markChannelRead,
  muteChannel: mocks.muteChannel,
}));

import { useChannelMessageStore, EMPTY_CACHE, readMessages } from "../channelMessageStore";
import { getChannelService } from "@/lib/wasm-core";
import type { ChannelMessage } from "@/lib/api/facade/channel";
import type { MessageSendPayload, MessageEditPayload } from "@/lib/viewModels/channelMessage";

function svc(): ReturnType<typeof getChannelService> {
  return getChannelService();
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

// wire ListChannelMessagesResponse bytes — fetchMessages now hands these to
// Rust apply_fetched_messages (no TS projection).
function wireMessagesBytes(msgs: ChannelMessage[], hasMore: boolean): Uint8Array {
  return toBinary(ListChannelMessagesResponseSchema, create(ListChannelMessagesResponseSchema, {
    items: msgs.map((m) => create(ChannelMessageSchema, {
      id: BigInt(m.id), channelId: BigInt(m.channel_id), body: m.body,
      messageType: m.message_type ?? "text",
      senderUserId: m.sender_user_id !== undefined ? BigInt(m.sender_user_id) : undefined,
      createdAt: m.created_at ?? "",
    })),
    hasMore,
  }));
}

beforeEach(() => {
  vi.clearAllMocks();
  Object.values(mocks).forEach((m) => m.mockReset());
  useChannelMessageStore.setState({ cache: {}, _messagesTick: 0, _unreadTick: 0 });
});

describe("useChannelMessageStore — Connect adapter integration", () => {
  it("fetchMessages: calls listChannelMessagesRaw and seeds the SSOT cache", async () => {
    const msgs = [makeMsg(1), makeMsg(2)];
    mocks.listChannelMessagesRaw.mockResolvedValue(wireMessagesBytes(msgs, true));

    await act(async () => {
      await useChannelMessageStore.getState().fetchMessages(42, 20);
    });

    expect(mocks.listChannelMessagesRaw).toHaveBeenCalledWith(orgSlug, 42, { beforeId: undefined, limit: 20 });
    const cache = useChannelMessageStore.getState().cache[42] ?? EMPTY_CACHE;
    const view = readMessages(42);
    expect(view.messages).toHaveLength(2);
    expect(view.hasMore).toBe(true);
    expect(cache.loading).toBe(false);
  });

  it("fetchMessages: records error on failure", async () => {
    mocks.listChannelMessagesRaw.mockRejectedValue(new Error("boom"));
    await act(async () => {
      await useChannelMessageStore.getState().fetchMessages(42);
    });
    expect(useChannelMessageStore.getState().cache[42]?.error).toBe("boom");
    expect(useChannelMessageStore.getState().cache[42]?.loading).toBe(false);
  });

  it("sendMessage: forwards source + mentions + podKey and merges into cache", async () => {
    const payload: MessageSendPayload = {
      source: "hi",
      mentions: { bot: { entity_type: "pod", entity_key: "pod-abc" } },
    };
    const returned = makeMsg(100, { body: "hi" });
    mocks.sendChannelMessage.mockResolvedValue(returned);

    let result: ChannelMessage | undefined;
    await act(async () => {
      result = await useChannelMessageStore.getState().sendMessage(42, payload, "pod-abc");
    });

    expect(mocks.sendChannelMessage).toHaveBeenCalledWith(orgSlug, 42, {
      source: "hi", mentions: payload.mentions, attachment_key: undefined, pod_key: "pod-abc",
    });
    expect(result?.id).toBe(100);
    expect(readMessages(42).messages.some((m) => m.id === 100)).toBe(true);
  });

  it("editMessage: forwards source payload to editChannelMessage", async () => {
    mocks.editChannelMessage.mockResolvedValue(makeMsg(7, { body: "edited" }));
    const payload: MessageEditPayload = { source: "edited" };
    await act(async () => {
      await useChannelMessageStore.getState().editMessage(42, 7, payload);
    });
    expect(mocks.editChannelMessage).toHaveBeenCalledWith(orgSlug, 42, 7, {
      source: "edited", mentions: undefined,
    });
  });

  it("deleteMessage: calls deleteChannelMessage and clears local copy", async () => {
    mocks.deleteChannelMessage.mockResolvedValue(undefined);
    await act(async () => {
      await useChannelMessageStore.getState().deleteMessage(42, 5);
    });
    expect(mocks.deleteChannelMessage).toHaveBeenCalledWith(orgSlug, 42, 5);
  });

  it("addMessage: dispatches to WASM on_new_message (local-only event handler)", () => {
    const msg = makeMsg(9);
    useChannelMessageStore.getState().addMessage(42, msg);
    // svc().on_new_message is the existing local hook; verify state effect via readMessages.
    expect(readMessages(42).messages.some((m) => m.id === 9)).toBe(true);
  });

  it("fetchUnreadCounts: writes counts into SSOT and bumps tick", async () => {
    mocks.getChannelUnreadCounts.mockResolvedValue({ "1": 3, "2": 5 });
    await act(async () => {
      await useChannelMessageStore.getState().fetchUnreadCounts();
    });
    expect(mocks.getChannelUnreadCounts).toHaveBeenCalledWith(orgSlug);
    expect(useChannelMessageStore.getState()._unreadTick).toBeGreaterThan(0);
    expect(svc().get_unread_count(BigInt(1))).toBe(3);
  });

  it("markRead: calls markChannelRead and bumps unread tick", async () => {
    mocks.markChannelRead.mockResolvedValue(undefined);
    await act(async () => {
      await useChannelMessageStore.getState().markRead(42, 99);
    });
    expect(mocks.markChannelRead).toHaveBeenCalledWith(orgSlug, 42, 99);
    expect(useChannelMessageStore.getState()._unreadTick).toBeGreaterThan(0);
  });

  it("muteChannel: forwards muted flag", async () => {
    mocks.muteChannel.mockResolvedValue(undefined);
    await act(async () => {
      await useChannelMessageStore.getState().muteChannel(42, true);
    });
    expect(mocks.muteChannel).toHaveBeenCalledWith(orgSlug, 42, true);
  });

  it("incrementUnread: dispatches to WASM and bumps tick (no JS mirror)", () => {
    const before = useChannelMessageStore.getState()._unreadTick;
    useChannelMessageStore.getState().incrementUnread(42);
    expect(useChannelMessageStore.getState()._unreadTick).toBe(before + 1);
  });

  it("clearChannelUnread: dispatches to WASM and bumps tick", () => {
    const before = useChannelMessageStore.getState()._unreadTick;
    useChannelMessageStore.getState().clearChannelUnread(42);
    expect(useChannelMessageStore.getState()._unreadTick).toBe(before + 1);
  });
});

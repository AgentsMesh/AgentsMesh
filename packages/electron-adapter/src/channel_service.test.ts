import { describe, it, expect, beforeEach } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  ChannelSchema, ChannelMessageSchema, ChannelMessageSenderUserSchema,
  ListChannelsResponseSchema, ListChannelMessagesResponseSchema,
} from "@agentsmesh/proto/channel/v1/channel_pb";
import {
  ReplaceCachedChannelsRequestSchema, ReplaceCachedChannelMessagesRequestSchema,
} from "@agentsmesh/proto/channel_state/v1/mutations_pb";
import { ElectronChannelService } from "./channel";

// Desktop fetch→state runtime contract: the shared store calls
// apply_fetched_* with wire bytes, and reads back via *_bytes. This verifies
// (1) the renderer cache updates synchronously, (2) the SAME wire bytes are
// fanned out to main (runtime.state SSOT), (3) the read path re-encodes the
// cache into state proto the web selectors decode — all WITHOUT a backend.
describe("ElectronChannelService fetch→state", () => {
  let invokes: Array<{ channel: string; args: unknown[] }>;

  beforeEach(() => {
    invokes = [];
    (globalThis as { window?: unknown }).window = {
      electronAPI: {
        invoke: async (channel: string, ...args: unknown[]) => {
          invokes.push({ channel, args });
          return undefined;
        },
      },
    };
  });

  const wireChannelsBytes = () =>
    toBinary(ListChannelsResponseSchema, create(ListChannelsResponseSchema, {
      items: [create(ChannelSchema, {
        id: 1n, organizationId: 7n, name: "general", memberCount: 3n, agentCount: 0n,
        isMember: true, createdAt: "2026-01-01", updatedAt: "2026-01-02",
      })],
    }));

  it("apply_fetched_channels updates cache + fans wire bytes to main", () => {
    const svc = new ElectronChannelService();
    const bytes = wireChannelsBytes();
    svc.apply_fetched_channels(bytes);

    // (1) renderer cache reflects the fetch
    const cached = JSON.parse(svc.channels_json()) as Array<{ id: number; name: string }>;
    expect(cached).toHaveLength(1);
    expect(cached[0].name).toBe("general");

    // (2) main SSOT fan-out with the SAME wire bytes
    const fan = invokes.find((i) => i.channel === "appChannelApplyFetchedChannels");
    expect(fan).toBeDefined();
    expect(Array.from(fan!.args[0] as number[])).toEqual(Array.from(bytes));

    // (3) read path re-encodes into state proto the web selector decodes
    const decoded = fromBinary(ReplaceCachedChannelsRequestSchema, svc.channels_bytes());
    expect(decoded.channels[0].name).toBe("general");
    expect(decoded.channels[0].organizationId).toBe(7n);
    expect(decoded.channels[0].memberCount).toBe(3n);
  });

  it("apply_fetched_messages caches + read returns nested sender via bytes", () => {
    const svc = new ElectronChannelService();
    const bytes = toBinary(ListChannelMessagesResponseSchema, create(ListChannelMessagesResponseSchema, {
      items: [create(ChannelMessageSchema, {
        id: 10n, channelId: 1n, body: "hi", messageType: "text", createdAt: "2026-01-01",
        senderUser: create(ChannelMessageSenderUserSchema, { id: 2n, username: "alice", name: "Alice" }),
      })],
      hasMore: true,
    }));
    svc.apply_fetched_messages(1n, bytes);

    expect(invokes.some((i) => i.channel === "appChannelApplyFetchedMessages")).toBe(true);
    const decoded = fromBinary(ReplaceCachedChannelMessagesRequestSchema, svc.get_messages_bytes(1n));
    expect(decoded.hasMore).toBe(true);
    expect(decoded.messages[0].body).toBe("hi");
    expect(decoded.messages[0].senderUser?.name).toBe("Alice");
  });

  it("apply_fetched_messages_prepend dedups against existing cache", () => {
    const svc = new ElectronChannelService();
    const mk = (id: bigint, body: string) => create(ChannelMessageSchema, {
      id, channelId: 1n, body, messageType: "text", createdAt: "2026-01-01",
    });
    svc.apply_fetched_messages(1n, toBinary(ListChannelMessagesResponseSchema,
      create(ListChannelMessagesResponseSchema, { items: [mk(2n, "b"), mk(3n, "c")], hasMore: false })));
    // Prepend older page that re-includes id=2 — must not duplicate.
    svc.apply_fetched_messages_prepend(1n, toBinary(ListChannelMessagesResponseSchema,
      create(ListChannelMessagesResponseSchema, { items: [mk(1n, "a"), mk(2n, "b")], hasMore: true })));

    const decoded = fromBinary(ReplaceCachedChannelMessagesRequestSchema, svc.get_messages_bytes(1n));
    expect(decoded.messages.map((m) => Number(m.id))).toEqual([1, 2, 3]);
    expect(decoded.hasMore).toBe(true);
  });
});

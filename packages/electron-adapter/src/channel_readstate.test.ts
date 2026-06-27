import { describe, it, expect, beforeEach } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { ChannelSchema, ListChannelsResponseSchema } from "@agentsmesh/proto/channel/v1/channel_pb";
import {
  ReplaceChannelUnreadCountsRequestSchema,
  ReplaceCachedChannelsRequestSchema,
} from "@agentsmesh/proto/channel_state/v1/mutations_pb";
import { ElectronChannelService } from "./channel";

// Desktop read-state parity (the #1 fix): the shared web store now calls
// get_last_read_id / advance_last_read and fans mentions/last_read/manually_unread
// through replace_channel_unread_counts; is_muted/is_pinned must round-trip the
// renderer cache. Without these the desktop crashes / shows dead pin+mute+@me.
describe("ElectronChannelService read-state", () => {
  beforeEach(() => {
    (globalThis as { window?: unknown }).window = {
      electronAPI: { invoke: async () => undefined },
    };
  });

  it("replace_channel_unread_counts fans unread/mention/last_read/manually_unread into the cache", async () => {
    const svc = new ElectronChannelService();
    const req = create(ReplaceChannelUnreadCountsRequestSchema, {
      counts: { "1": 3 },
      mentions: { "1": 2 },
      lastRead: { "1": 99n },
      manuallyUnread: { "2": true },
    });
    await svc.replace_channel_unread_counts(toBinary(ReplaceChannelUnreadCountsRequestSchema, req));

    // unread + manuallyUnread ride unread_counts_bytes (the sidebar selectors decode it)
    const decoded = fromBinary(ReplaceChannelUnreadCountsRequestSchema, svc.unread_counts_bytes());
    expect(decoded.counts["1"]).toBe(3);
    expect(decoded.manuallyUnread["2"]).toBe(true);
    // mentions land in mention_counts_json (the @me selector reads it)
    expect((JSON.parse(svc.mention_counts_json()) as Record<string, number>)["1"]).toBe(2);
    // last_read drives the divider cursor; manuallyUnread reads back per-channel
    expect(svc.get_last_read_id(1n)).toBe(99);
    expect(svc.get_manually_unread(2n)).toBe(true);
  });

  it("get_last_read_id is -1 until known, then advances monotonically", () => {
    const svc = new ElectronChannelService();
    expect(svc.get_last_read_id(1n)).toBe(-1); // sentinel: no cursor (≠ genuine 0)
    svc.advance_last_read(1n, 50n);
    expect(svc.get_last_read_id(1n)).toBe(50);
    svc.advance_last_read(1n, 40n); // older → ignored
    expect(svc.get_last_read_id(1n)).toBe(50);
    svc.advance_last_read(1n, 60n);
    expect(svc.get_last_read_id(1n)).toBe(60);
  });

  it("clear_channel_unread dismisses manually_unread", async () => {
    const svc = new ElectronChannelService();
    await svc.replace_channel_unread_counts(
      toBinary(
        ReplaceChannelUnreadCountsRequestSchema,
        create(ReplaceChannelUnreadCountsRequestSchema, { manuallyUnread: { "1": true } }),
      ),
    );
    expect(svc.get_manually_unread(1n)).toBe(true);
    svc.clear_channel_unread(1n);
    expect(svc.get_manually_unread(1n)).toBe(false);
  });

  it("is_muted/is_pinned round-trip through the channel cache", () => {
    const svc = new ElectronChannelService();
    const bytes = toBinary(
      ListChannelsResponseSchema,
      create(ListChannelsResponseSchema, {
        items: [
          create(ChannelSchema, {
            id: 1n, organizationId: 7n, name: "general", memberCount: 1n, agentCount: 0n,
            isMember: true, isMuted: true, isPinned: true, createdAt: "x", updatedAt: "y",
          }),
        ],
      }),
    );
    svc.apply_fetched_channels(bytes);

    const cached = (JSON.parse(svc.channels_json()) as Array<{ is_muted: boolean; is_pinned: boolean }>)[0];
    expect(cached.is_muted).toBe(true);
    expect(cached.is_pinned).toBe(true);
    // channels_bytes re-encodes them so the shared web selector decodes them too
    const decoded = fromBinary(ReplaceCachedChannelsRequestSchema, svc.channels_bytes());
    expect(decoded.channels[0].isMuted).toBe(true);
    expect(decoded.channels[0].isPinned).toBe(true);
  });
});

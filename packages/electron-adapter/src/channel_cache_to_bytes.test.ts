import { describe, it, expect } from "vitest";
import { create, fromBinary } from "@bufbuild/protobuf";
import {
  ChannelSchema, ChannelMessageSchema, ChannelMemberSchema, ChannelPodSchema,
  ChannelMessageSenderUserSchema, ChannelMessageSenderPodSchema,
} from "@agentsmesh/proto/channel/v1/channel_pb";
import {
  ReplaceCachedChannelsRequestSchema,
  ReplaceCachedChannelMessagesRequestSchema,
  ReplaceChannelMembersRequestSchema,
  ReplaceChannelPodsRequestSchema,
  InsertChannelRequestSchema,
} from "@agentsmesh/proto/channel_state/v1/mutations_pb";
import { MessagePreviewSchema } from "@agentsmesh/proto/channel_state/v1/channel_state_pb";
import { channelToCache, messageToCache, memberToCache, podToCache } from "./channel_proto_to_cache";
import {
  channelsBytes, channelBytes, currentChannelBytes, messagesBytes,
  lastMessageBytes, membersBytes, podsBytes,
} from "./channel_cache_to_bytes";

// The desktop renderer round-trips fetched wire protos through a snake_case
// cache and back into state proto bytes. The shared web selectors then decode
// those bytes — so this round-trip MUST preserve every field the selectors
// read, or desktop silently diverges from web. wire → cache → bytes → state.
describe("channel cache→bytes round-trip", () => {
  it("preserves channel fields incl. Option-wrapped scalars", () => {
    const wire = create(ChannelSchema, {
      id: 1n, organizationId: 7n, name: "general", description: "d", document: "doc",
      repositoryId: 3n, ticketId: 9n, ticketSlug: "tk", visibility: "public",
      isArchived: false, isMember: true, memberCount: 5n, agentCount: 2n,
      createdAt: "2026-01-01", updatedAt: "2026-01-02",
    });
    const decoded = fromBinary(ReplaceCachedChannelsRequestSchema, channelsBytes(JSON.stringify([channelToCache(wire)])));
    const c = decoded.channels[0];
    expect(c.id).toBe(1n);
    expect(c.organizationId).toBe(7n);
    expect(c.name).toBe("general");
    expect(c.repositoryId).toBe(3n);
    expect(c.ticketSlug).toBe("tk");
    expect(c.visibility).toBe("public");
    expect(c.isMember).toBe(true);
    expect(c.memberCount).toBe(5n);
    expect(c.agentCount).toBe(2n);
  });

  it("preserves single channel via InsertChannelRequest (get/current)", () => {
    const wire = create(ChannelSchema, { id: 42n, organizationId: 1n, name: "ops", memberCount: 0n, agentCount: 0n });
    const cacheJson = JSON.stringify([channelToCache(wire)]);
    expect(fromBinary(InsertChannelRequestSchema, channelBytes(cacheJson, 42)).channel?.name).toBe("ops");
    expect(fromBinary(InsertChannelRequestSchema, currentChannelBytes(cacheJson, 42)).channel?.id).toBe(42n);
    expect(channelBytes(cacheJson, 999).length).toBe(0);
    expect(currentChannelBytes(cacheJson, null).length).toBe(0);
  });

  it("preserves message nested sender_user + content_json", () => {
    const wire = create(ChannelMessageSchema, {
      id: 10n, channelId: 1n, body: "hi", messageType: "text", createdAt: "2026-01-01",
      contentJson: '{"ops":[]}', isDeleted: false,
      senderUser: create(ChannelMessageSenderUserSchema, { id: 2n, username: "alice", name: "Alice", avatarUrl: "a.png" }),
    });
    const decoded = fromBinary(ReplaceCachedChannelMessagesRequestSchema, messagesBytes(1, [messageToCache(wire)], true));
    expect(decoded.hasMore).toBe(true);
    const m = decoded.messages[0];
    expect(m.body).toBe("hi");
    expect(m.contentJson).toBe('{"ops":[]}');
    expect(m.senderUser?.name).toBe("Alice");
    expect(m.senderUser?.username).toBe("alice");
    expect(m.senderUser?.avatarUrl).toBe("a.png");
  });

  it("preserves message nested sender_pod_info.agent", () => {
    const wire = create(ChannelMessageSchema, {
      id: 11n, channelId: 1n, body: "bot", messageType: "text", createdAt: "2026-01-01",
      senderPodInfo: create(ChannelMessageSenderPodSchema, { podKey: "pk-1", alias: "Bot" }),
    });
    const m = fromBinary(ReplaceCachedChannelMessagesRequestSchema, messagesBytes(1, [messageToCache(wire)], false)).messages[0];
    expect(m.senderPodInfo?.podKey).toBe("pk-1");
    expect(m.senderPodInfo?.alias).toBe("Bot");
  });

  it("derives last-message preview from the newest cache message", () => {
    const a = messageToCache(create(ChannelMessageSchema, { id: 1n, channelId: 1n, body: "old", createdAt: "2026-01-01" }));
    const b = messageToCache(create(ChannelMessageSchema, {
      id: 2n, channelId: 1n, body: "newest", createdAt: "2026-01-02",
      senderUser: create(ChannelMessageSenderUserSchema, { id: 5n, username: "u", name: "Newbie" }),
    }));
    const p = fromBinary(MessagePreviewSchema, lastMessageBytes([a, b]));
    expect(p.messageId).toBe(2n);
    expect(p.senderName).toBe("Newbie");
    expect(p.contentPreview).toBe("newest");
    expect(lastMessageBytes([]).length).toBe(0);
  });

  it("preserves members + pods", () => {
    const member = memberToCache(create(ChannelMemberSchema, { channelId: 1n, userId: 4n, role: "admin", isMuted: true, joinedAt: "2026-01-01" }));
    const dm = fromBinary(ReplaceChannelMembersRequestSchema, membersBytes(1, JSON.stringify([member]))).members[0];
    expect(dm.userId).toBe(4n);
    expect(dm.role).toBe("admin");
    expect(dm.isMuted).toBe(true);

    const pod = podToCache(create(ChannelPodSchema, { id: 8n, podKey: "pk-x", alias: "X", status: "running", agentStatus: "idle" }));
    const dp = fromBinary(ReplaceChannelPodsRequestSchema, podsBytes(1, JSON.stringify([pod]))).pods[0];
    expect(dp.podKey).toBe("pk-x");
    expect(dp.status).toBe("running");
    expect(dp.agentStatus).toBe("idle");
  });
});

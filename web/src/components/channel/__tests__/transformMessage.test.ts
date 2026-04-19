import { describe, it, expect } from "vitest";
import { transformMessage } from "../transformMessage";
import type { ChannelMessage } from "@/lib/api";

const baseMessage: ChannelMessage = {
  id: 1,
  channel_id: 10,
  message_type: "text",
  body: "hello",
  content: { kind: "text", blocks: [{ type: "paragraph", elements: [{ type: "text", text: "hello" }] }] },
  created_at: "2024-01-01T00:00:00Z",
};

describe("transformMessage", () => {
  it("maps body and content to TransformedMessage", () => {
    const result = transformMessage(baseMessage);
    expect(result.body).toBe("hello");
    expect(result.content).toEqual(baseMessage.content);
    expect(result.messageType).toBe("text");
  });

  it("maps sender_pod_info to pod field", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      sender_pod_info: { pod_key: "pk-123", alias: "MyBot", agent: { name: "Claude" } },
    };
    const result = transformMessage(msg);
    expect(result.pod).toEqual({ podKey: "pk-123", alias: "MyBot", agent: { name: "Claude" } });
    expect(result.user).toBeUndefined();
  });

  it("falls back to sender_pod when sender_pod_info is missing", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      sender_pod: "pk-fallback-key",
    };
    const result = transformMessage(msg);
    expect(result.pod).toEqual({ podKey: "pk-fallback-key" });
  });

  it("returns undefined pod when neither sender_pod_info nor sender_pod present", () => {
    const result = transformMessage(baseMessage);
    expect(result.pod).toBeUndefined();
  });

  it("maps sender_user to user field", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      sender_user: { id: 42, username: "alice", name: "Alice", avatar_url: "https://a.co/pic.jpg" },
    };
    const result = transformMessage(msg);
    expect(result.user).toEqual({ id: 42, username: "alice", name: "Alice", avatarUrl: "https://a.co/pic.jpg" });
  });

  it("maps mentions through", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      mentions: { pods: ["pk-1"], users: [42] },
    };
    const result = transformMessage(msg);
    expect(result.mentions).toEqual({ pods: ["pk-1"], users: [42] });
  });

  it("handles system message with no content", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      message_type: "system",
      body: "User joined",
      content: undefined,
    };
    const result = transformMessage(msg);
    expect(result.messageType).toBe("system");
    expect(result.body).toBe("User joined");
    expect(result.content).toBeUndefined();
  });

  it("maps editedAt and createdAt", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      edited_at: "2024-01-02T00:00:00Z",
    };
    const result = transformMessage(msg);
    expect(result.editedAt).toBe("2024-01-02T00:00:00Z");
    expect(result.createdAt).toBe("2024-01-01T00:00:00Z");
  });

  it("prefers sender_pod_info over sender_pod", () => {
    const msg: ChannelMessage = {
      ...baseMessage,
      sender_pod: "pk-fallback",
      sender_pod_info: { pod_key: "pk-primary", alias: "Bot" },
    };
    const result = transformMessage(msg);
    expect(result.pod?.podKey).toBe("pk-primary");
    expect(result.pod?.alias).toBe("Bot");
  });
});

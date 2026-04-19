import { describe, it, expect } from "vitest";
import { buildMessageContent, extractMentionMap } from "../message-content-builder";
import type { MessageContent } from "@/lib/api/channel-message-types";

describe("buildMessageContent", () => {
  it("creates a single paragraph for plain text", () => {
    const result = buildMessageContent("hello world", new Map());
    expect(result.kind).toBe("text");
    expect(result.blocks).toHaveLength(1);
    expect(result.blocks![0].type).toBe("paragraph");
    expect(result.blocks![0].elements).toEqual([
      { type: "text", text: "hello world" },
    ]);
  });

  it("splits newlines into separate paragraph blocks", () => {
    const result = buildMessageContent("line one\nline two\nline three", new Map());
    expect(result.blocks).toHaveLength(3);
    expect(result.blocks![0].elements).toEqual([{ type: "text", text: "line one" }]);
    expect(result.blocks![1].elements).toEqual([{ type: "text", text: "line two" }]);
    expect(result.blocks![2].elements).toEqual([{ type: "text", text: "line three" }]);
  });

  it("resolves known @mentions into mention elements", () => {
    const mentions = new Map([
      ["alice", { entityType: "user", entityKey: "42" }],
    ]);
    const result = buildMessageContent("hey @alice check this", mentions);
    expect(result.blocks![0].elements).toEqual([
      { type: "text", text: "hey " },
      { type: "mention", entity_type: "user", entity_key: "42", display: "alice" },
      { type: "text", text: " check this" },
    ]);
  });

  it("resolves pod mentions", () => {
    const mentions = new Map([
      ["My_Bot", { entityType: "pod", entityKey: "pod-key-123" }],
    ]);
    const result = buildMessageContent("@My_Bot do something", mentions);
    expect(result.blocks![0].elements).toEqual([
      { type: "mention", entity_type: "pod", entity_key: "pod-key-123", display: "My_Bot" },
      { type: "text", text: " do something" },
    ]);
  });

  it("leaves unknown @tokens as plain text", () => {
    const result = buildMessageContent("hey @unknown person", new Map());
    expect(result.blocks![0].elements).toEqual([
      { type: "text", text: "hey " },
      { type: "text", text: "@unknown" },
      { type: "text", text: " person" },
    ]);
  });

  it("handles multiple mentions in one line", () => {
    const mentions = new Map([
      ["alice", { entityType: "user", entityKey: "1" }],
      ["bob", { entityType: "user", entityKey: "2" }],
    ]);
    const result = buildMessageContent("@alice and @bob", mentions);
    expect(result.blocks![0].elements).toEqual([
      { type: "mention", entity_type: "user", entity_key: "1", display: "alice" },
      { type: "text", text: " and " },
      { type: "mention", entity_type: "user", entity_key: "2", display: "bob" },
    ]);
  });

  it("handles mention with dots and dashes in name", () => {
    const mentions = new Map([
      ["my-bot.v2", { entityType: "pod", entityKey: "pk-123" }],
    ]);
    const result = buildMessageContent("ask @my-bot.v2", mentions);
    expect(result.blocks![0].elements).toEqual([
      { type: "text", text: "ask " },
      { type: "mention", entity_type: "pod", entity_key: "pk-123", display: "my-bot.v2" },
    ]);
  });

  it("handles empty string", () => {
    const result = buildMessageContent("", new Map());
    expect(result.kind).toBe("text");
    expect(result.blocks).toHaveLength(1);
    expect(result.blocks![0].elements).toEqual([]);
  });
});

describe("extractMentionMap", () => {
  it("returns empty map for undefined content", () => {
    expect(extractMentionMap(undefined).size).toBe(0);
  });

  it("returns empty map for content without blocks", () => {
    expect(extractMentionMap({ kind: "text" }).size).toBe(0);
  });

  it("returns empty map for content without mentions", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{ type: "paragraph", elements: [{ type: "text", text: "hello" }] }],
    };
    expect(extractMentionMap(content).size).toBe(0);
  });

  it("extracts pod mention from content", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "mention", entity_type: "pod", entity_key: "pk-123", display: "MyBot" },
        ],
      }],
    };
    const map = extractMentionMap(content);
    expect(map.size).toBe(1);
    expect(map.get("MyBot")).toEqual({ entityType: "pod", entityKey: "pk-123" });
  });

  it("extracts user mention from content", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "mention", entity_type: "user", entity_key: "42", display: "alice" },
        ],
      }],
    };
    const map = extractMentionMap(content);
    expect(map.get("alice")).toEqual({ entityType: "user", entityKey: "42" });
  });

  it("extracts multiple mentions across blocks", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [
        { type: "paragraph", elements: [{ type: "mention", entity_type: "pod", entity_key: "pk-1", display: "BotA" }] },
        { type: "paragraph", elements: [{ type: "mention", entity_type: "user", entity_key: "99", display: "bob" }] },
      ],
    };
    const map = extractMentionMap(content);
    expect(map.size).toBe(2);
    expect(map.has("BotA")).toBe(true);
    expect(map.has("bob")).toBe(true);
  });

  it("skips mention elements without display or entity_key", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "mention", entity_type: "pod" },
          { type: "mention", entity_key: "pk-1" },
        ],
      }],
    };
    expect(extractMentionMap(content).size).toBe(0);
  });

  it("roundtrip: build → extract → rebuild preserves mentions", () => {
    const mentions = new Map([
      ["alice", { entityType: "user", entityKey: "42" }],
      ["MyBot", { entityType: "pod", entityKey: "pk-bot" }],
    ]);
    const original = buildMessageContent("Hey @alice ask @MyBot", mentions);
    const extracted = extractMentionMap(original);
    const rebuilt = buildMessageContent("Hey @alice ask @MyBot please", extracted);

    const mentionEls = rebuilt.blocks![0].elements!.filter(e => e.type === "mention");
    expect(mentionEls).toHaveLength(2);
    expect(mentionEls[0].entity_key).toBe("42");
    expect(mentionEls[1].entity_key).toBe("pk-bot");
  });
});

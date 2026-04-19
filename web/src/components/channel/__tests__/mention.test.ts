import { describe, it, expect } from "vitest";
import { getMentionQuery } from "../mention";

describe("getMentionQuery", () => {
  it("returns null when no @ present", () => {
    expect(getMentionQuery("hello world", 5)).toBeNull();
  });

  it("detects @ at start of text", () => {
    const result = getMentionQuery("@alice", 6);
    expect(result).toEqual({ query: "alice", startIndex: 0 });
  });

  it("detects @ after whitespace", () => {
    const result = getMentionQuery("hey @bob", 8);
    expect(result).toEqual({ query: "bob", startIndex: 4 });
  });

  it("returns partial query at cursor position", () => {
    const result = getMentionQuery("@ali", 4);
    expect(result).toEqual({ query: "ali", startIndex: 0 });
  });

  it("returns empty query right after @", () => {
    const result = getMentionQuery("@", 1);
    expect(result).toEqual({ query: "", startIndex: 0 });
  });

  it("returns null when @ is mid-word (not preceded by whitespace)", () => {
    expect(getMentionQuery("email@example.com", 17)).toBeNull();
  });

  it("returns null when query contains whitespace", () => {
    // "@ alice" — space after @ makes it invalid
    expect(getMentionQuery("hey @ alice", 11)).toBeNull();
  });

  it("uses the last @ before cursor when multiple exist", () => {
    const result = getMentionQuery("@alice hey @bo", 14);
    expect(result).toEqual({ query: "bo", startIndex: 11 });
  });

  it("detects @ after newline", () => {
    const result = getMentionQuery("line1\n@user", 11);
    expect(result).toEqual({ query: "user", startIndex: 6 });
  });

  it("detects @ after tab", () => {
    const result = getMentionQuery("text\t@name", 10);
    expect(result).toEqual({ query: "name", startIndex: 5 });
  });

  it("returns null when cursor is before @", () => {
    expect(getMentionQuery("before @after", 3)).toBeNull();
  });

  it("handles @ at start with cursor right after @", () => {
    const result = getMentionQuery("hey @", 5);
    expect(result).toEqual({ query: "", startIndex: 4 });
  });
});

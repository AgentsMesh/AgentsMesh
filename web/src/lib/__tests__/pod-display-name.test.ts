import { describe, it, expect } from "vitest";
import { getPodDisplayName, getShortPodKey, getMentionSafeName } from "../pod-display-name";

describe("getShortPodKey", () => {
  it("returns first 8 characters of pod key", () => {
    expect(getShortPodKey("abcdefgh12345678")).toBe("abcdefgh");
  });

  it("returns full key when shorter than 8 characters", () => {
    expect(getShortPodKey("short")).toBe("short");
  });
});

describe("getPodDisplayName", () => {
  it("returns alias when set (highest priority)", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        alias: "My Alias",
        title: "OSC Title",
        ticket: { title: "Ticket Title", slug: "T-1" },
      })
    ).toBe("My Alias");
  });

  it("returns ticket title when no alias", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        title: "OSC Title",
        ticket: { title: "Ticket Title", slug: "T-1" },
      })
    ).toBe("Ticket Title");
  });

  it("returns loop name when no alias or ticket title", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        title: "OSC Title",
        loop: { name: "My Loop" },
      })
    ).toBe("My Loop");
  });

  it("returns OSC title when no alias, ticket, or loop", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        title: "OSC Title",
      })
    ).toBe("OSC Title");
  });

  it("returns ticket slug when no alias, title, or loop", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        ticket: { slug: "T-42" },
      })
    ).toBe("T-42");
  });

  it("returns agent name + short key when no other info", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        agent: { name: "Claude" },
      })
    ).toBe("Claude (pod-key-)");
  });

  it("returns short key with ellipsis as last fallback", () => {
    expect(
      getPodDisplayName({ pod_key: "pod-key-12345678" })
    ).toBe("pod-key-...");
  });

  it("truncates long alias to maxLength", () => {
    const longAlias = "A".repeat(30);
    expect(
      getPodDisplayName({ pod_key: "k", alias: longAlias }, 20)
    ).toBe("A".repeat(17) + "...");
  });

  it("truncates long ticket title to maxLength", () => {
    const longTitle = "T".repeat(30);
    expect(
      getPodDisplayName({ pod_key: "k", ticket: { title: longTitle } }, 20)
    ).toBe("T".repeat(17) + "...");
  });

  it("truncates long loop name to maxLength", () => {
    const longName = "L".repeat(30);
    expect(
      getPodDisplayName({ pod_key: "k", loop: { name: longName } }, 20)
    ).toBe("L".repeat(17) + "...");
  });

  it("truncates long OSC title to maxLength", () => {
    const longTitle = "O".repeat(30);
    expect(
      getPodDisplayName({ pod_key: "k", title: longTitle }, 20)
    ).toBe("O".repeat(17) + "...");
  });

  it("skips null alias", () => {
    expect(
      getPodDisplayName({
        pod_key: "pod-key-12345678",
        alias: null,
        title: "Fallback Title",
      })
    ).toBe("Fallback Title");
  });
});

describe("getMentionSafeName", () => {
  it("returns alias as-is when no spaces", () => {
    expect(getMentionSafeName({ pod_key: "k", alias: "MyBot" })).toBe("MyBot");
  });

  it("replaces spaces with underscores in alias", () => {
    expect(getMentionSafeName({ pod_key: "k", alias: "My Bot Name" })).toBe("My_Bot_Name");
  });

  it("falls back to ticket slug when no alias", () => {
    expect(getMentionSafeName({ pod_key: "k", ticket: { slug: "AM-42" } })).toBe("AM-42");
  });

  it("falls back to short pod key when no alias or ticket slug", () => {
    expect(getMentionSafeName({ pod_key: "abcdefgh12345678" })).toBe("abcdefgh");
  });

  it("prefers alias over ticket slug", () => {
    expect(
      getMentionSafeName({ pod_key: "k", alias: "MyAlias", ticket: { slug: "AM-1" } })
    ).toBe("MyAlias");
  });
});

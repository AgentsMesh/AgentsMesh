import { describe, it, expect } from "vitest";
import { getPodDisplayName, getShortPodKey } from "../pod-utils";

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

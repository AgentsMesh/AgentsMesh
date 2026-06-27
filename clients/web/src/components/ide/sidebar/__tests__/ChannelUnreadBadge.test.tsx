import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { ChannelUnreadBadge } from "../ChannelUnreadBadge";

vi.mock("next-intl", () => ({ useTranslations: () => (k: string) => k }));

type Props = Parameters<typeof ChannelUnreadBadge>[0];

function indicators(c: HTMLElement) {
  const spans = Array.from(c.querySelectorAll("span"));
  const destructive = spans.filter((s) => s.className.includes("bg-destructive"));
  return {
    redCount: destructive.find((s) => (s.textContent ?? "").trim() !== "")?.textContent?.trim() ?? null,
    redDot: destructive.some((s) => (s.textContent ?? "").trim() === ""),
    grayDot: spans.some((s) => s.className.includes("bg-muted-foreground/40")),
    muted: !!c.querySelector("svg"),
  };
}
const view = (p: Props) => indicators(render(<ChannelUnreadBadge {...p} />).container);

describe("ChannelUnreadBadge priority", () => {
  it("plain unread → red count", () => {
    const i = view({ unreadCount: 3 });
    expect(i.redCount).toBe("3");
    expect(i.grayDot).toBe(false);
  });

  it("muted unread → gray dot + bell, no count", () => {
    const i = view({ unreadCount: 3, isMuted: true });
    expect(i.redCount).toBeNull();
    expect(i.grayDot).toBe(true);
    expect(i.muted).toBe(true);
  });

  it("mention pierces mute → red count even when muted", () => {
    expect(view({ unreadCount: 3, mentionCount: 2, isMuted: true }).redCount).toBe("3");
  });

  it("stranded mention (mention>0, unread=0) → never a red \"0\"", () => {
    const i = view({ unreadCount: 0, mentionCount: 2 });
    expect(i.redCount).toBeNull(); // the #2 fix: gate the red badge on unreadCount > 0
    expect(i.redDot).toBe(false);
  });

  it("manually-unread (unread=0) → red dot, no count", () => {
    const i = view({ unreadCount: 0, manuallyUnread: true });
    expect(i.redDot).toBe(true);
    expect(i.redCount).toBeNull();
  });

  it("selected channel → no indicator", () => {
    const i = view({ unreadCount: 3, isSelected: true });
    expect(i.redCount).toBeNull();
    expect(i.redDot).toBe(false);
    expect(i.grayDot).toBe(false);
  });

  it("caps the count at 99+", () => {
    expect(view({ unreadCount: 150 }).redCount).toBe("99+");
  });
});

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ChannelRailPodItem } from "../ChannelRailPodItem";
import type { ChannelPodSummary } from "@/hooks/useChannelPods";

const ALL_COLORS = ["bg-emerald-500", "bg-amber-500", "bg-red-500", "bg-muted-foreground/50"];

function renderRow(pod: ChannelPodSummary, dimmed?: boolean) {
  render(
    <ul>
      <ChannelRailPodItem pod={pod} dimmed={dimmed} />
    </ul>,
  );
  return screen.getByTestId("channel-rail-pod");
}

describe("ChannelRailPodItem status colors", () => {
  const cases: Array<[string, string]> = [
    ["running", "bg-emerald-500"],
    ["initializing", "bg-amber-500"],
    ["paused", "bg-muted-foreground/50"],
    ["disconnected", "bg-muted-foreground/50"],
    ["orphaned", "bg-muted-foreground/50"],
    ["completed", "bg-muted-foreground/50"],
    ["terminated", "bg-muted-foreground/50"],
    ["error", "bg-red-500"],
    ["failed", "bg-red-500"],
  ];

  it.each(cases)("colors status '%s' as %s on both avatar and dot", (status, expected) => {
    const row = renderRow({ pod_key: "pk", alias: "Pod", status });
    const avatar = row.querySelector(".rounded-md");
    const dot = row.querySelector(".rounded-full");

    for (const el of [avatar, dot]) {
      expect(el).not.toBeNull();
      expect(el!.className).toContain(expected);
      for (const other of ALL_COLORS) {
        if (other !== expected) expect(el!.className).not.toContain(other);
      }
    }
  });
});

describe("ChannelRailPodItem rendering", () => {
  it("dims the row and strikes the label only when dimmed", () => {
    const row = renderRow({ pod_key: "pk", alias: "Pod", status: "terminated" }, true);
    expect(row.className).toContain("opacity-60");
    expect(screen.getByText("Pod").className).toContain("line-through");
  });

  it("falls back to pod_key for the label and avatar letter when alias is missing", () => {
    renderRow({ pod_key: "zeta-pod", status: "running" });
    expect(screen.getByText("zeta-pod")).toBeDefined();
    expect(screen.getByText("Z")).toBeDefined();
  });
});

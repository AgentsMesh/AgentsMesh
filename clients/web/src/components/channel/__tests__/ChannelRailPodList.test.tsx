import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ChannelRailPodList } from "../ChannelRailPodList";
import type { ChannelPodSummary } from "@/hooks/useChannelPods";
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) =>
    key === "channels.rightRail.destroyed" ? "Destroyed" : key,
}));

const pods: ChannelPodSummary[] = [
  { pod_key: "pk-run", alias: "Runner", status: "running" },
  { pod_key: "pk-pause", alias: "Paused", status: "paused" },
  { pod_key: "pk-term", alias: "Terminated", status: "terminated" },
  { pod_key: "pk-fail", alias: "Failed", status: "failed" },
];

describe("ChannelRailPodList", () => {
  it("renders live pods (running, paused) and hides destroyed pods by default", () => {
    render(<ChannelRailPodList pods={pods} />);

    expect(screen.getByText("Runner")).toBeDefined();
    expect(screen.getByText("Paused")).toBeDefined();
    expect(screen.queryByText("Terminated")).toBeNull();
    expect(screen.queryByText("Failed")).toBeNull();
  });

  it("shows the destroyed toggle with the terminal-state count", () => {
    render(<ChannelRailPodList pods={pods} />);

    const toggle = screen.getByTestId("channel-rail-destroyed-toggle");
    expect(toggle.textContent).toContain("Destroyed · 2");
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
  });

  it("reveals dimmed destroyed pods when the toggle is clicked", () => {
    render(<ChannelRailPodList pods={pods} />);

    fireEvent.click(screen.getByTestId("channel-rail-destroyed-toggle"));

    const terminated = screen.getByText("Terminated").closest("li");
    expect(terminated).not.toBeNull();
    expect(terminated?.className).toContain("opacity-60");
    expect(screen.getByText("Terminated").className).toContain("line-through");
    expect(screen.getByTestId("channel-rail-destroyed-toggle").getAttribute("aria-expanded")).toBe("true");
  });

  it("omits the destroyed section entirely when no pods are destroyed", () => {
    render(
      <ChannelRailPodList
        pods={[{ pod_key: "pk-run", alias: "Runner", status: "running" }]}
      />,
    );

    expect(screen.queryByTestId("channel-rail-destroyed-toggle")).toBeNull();
  });

  it("renders no list and no toggle when given an empty pods array", () => {
    const { container } = render(<ChannelRailPodList pods={[]} />);

    expect(container.querySelector("ul")).toBeNull();
    expect(screen.queryByTestId("channel-rail-destroyed-toggle")).toBeNull();
    expect(screen.queryByTestId("channel-rail-pod")).toBeNull();
  });

  it("does not dim live pods", () => {
    render(
      <ChannelRailPodList
        pods={[{ pod_key: "pk-run", alias: "Runner", status: "running" }]}
      />,
    );

    expect(screen.getByText("Runner").closest("li")?.className).not.toContain("opacity-60");
    expect(screen.getByText("Runner").className).not.toContain("line-through");
  });

  it("auto-expands the destroyed section when there are no live pods, then collapses on toggle", () => {
    render(
      <ChannelRailPodList
        pods={[
          { pod_key: "pk-done", alias: "Done", status: "completed" },
          { pod_key: "pk-dead", alias: "Dead", status: "terminated" },
        ]}
      />,
    );

    const toggle = screen.getByTestId("channel-rail-destroyed-toggle");
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText("Done")).toBeDefined();
    expect(screen.getByText("Dead")).toBeDefined();

    fireEvent.click(toggle);

    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByText("Done")).toBeNull();
    expect(screen.queryByText("Dead")).toBeNull();
  });
});

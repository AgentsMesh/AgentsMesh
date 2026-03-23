import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { useAcpSessionStore } from "@/stores/acpSession";
import { AcpPlanTracker } from "@/components/workspace/acp/AcpPlanTracker";

const POD = "pod-plan";

describe("AcpPlanTracker", () => {
  beforeEach(() => {
    useAcpSessionStore.setState({ sessions: {} });
  });

  it("renders nothing when no plan exists", () => {
    const { container } = render(<AcpPlanTracker podKey="nonexistent" />);
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing for empty plan", () => {
    useAcpSessionStore.getState().updatePlan(POD, "s1", []);
    const { container } = render(<AcpPlanTracker podKey={POD} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders plan steps with correct labels", () => {
    useAcpSessionStore.getState().updatePlan(POD, "s1", [
      { title: "Read files", status: "completed" },
      { title: "Write code", status: "in_progress" },
      { title: "Run tests", status: "pending" },
    ]);

    render(<AcpPlanTracker podKey={POD} />);
    expect(screen.getByText("Plan")).toBeInTheDocument();
    expect(screen.getByText("Read files")).toBeInTheDocument();
    expect(screen.getByText("Write code")).toBeInTheDocument();
    expect(screen.getByText("Run tests")).toBeInTheDocument();
  });

  it("applies correct styling per status", () => {
    useAcpSessionStore.getState().updatePlan(POD, "s1", [
      { title: "Done step", status: "completed" },
      { title: "Active step", status: "in_progress" },
      { title: "Todo step", status: "pending" },
    ]);

    render(<AcpPlanTracker podKey={POD} />);

    const done = screen.getByText("Done step").closest("span");
    expect(done?.className).toContain("bg-green");

    const active = screen.getByText("Active step").closest("span");
    expect(active?.className).toContain("bg-blue");

    const todo = screen.getByText("Todo step").closest("span");
    expect(todo?.className).toContain("bg-muted");
  });
});

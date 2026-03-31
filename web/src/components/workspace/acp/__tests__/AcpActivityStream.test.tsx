import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { useAcpSessionStore } from "@/stores/acpSession";
import { AcpActivityStream } from "@/components/workspace/acp/AcpActivityStream";

const POD = "pod-stream";

function seedSession(setup: () => void) {
  useAcpSessionStore.setState({ sessions: {} });
  setup();
}

describe("AcpActivityStream", () => {
  beforeEach(() => {
    useAcpSessionStore.setState({ sessions: {} });
  });

  it("shows waiting message when no session exists", () => {
    render(<AcpActivityStream podKey="nonexistent" />);
    expect(screen.getByText("Waiting for ACP session...")).toBeInTheDocument();
  });

  it("renders user instruction with > prefix", () => {
    seedSession(() => {
      const s = useAcpSessionStore.getState();
      s.addContentChunk(POD, "s1", "create a hello world app", "user");
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText(">")).toBeInTheDocument();
    expect(screen.getByText("create a hello world app")).toBeInTheDocument();
  });

  it("renders slash commands with distinct styling", () => {
    seedSession(() => {
      const s = useAcpSessionStore.getState();
      s.addContentChunk(POD, "s1", "/compact", "user");
    });

    render(<AcpActivityStream podKey={POD} />);
    const cmd = screen.getByText("/compact");
    expect(cmd.className).toContain("font-mono");
    expect(cmd.className).toContain("text-blue");
  });

  it("renders assistant output as markdown", () => {
    seedSession(() => {
      const s = useAcpSessionStore.getState();
      s.addContentChunk(POD, "s1", "I'll help you with that.", "assistant");
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText(/I'll help you with that/)).toBeInTheDocument();
  });

  it("renders tool calls in timeline", () => {
    seedSession(() => {
      const s = useAcpSessionStore.getState();
      s.updateToolCall(POD, "s1", {
        toolCallId: "tc1", toolName: "write_file", status: "completed",
        argumentsJson: '{"path":"main.ts"}', timestamp: 0,
      });
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText("write_file")).toBeInTheDocument();
  });

  it("renders thinking indicator", () => {
    seedSession(() => {
      const s = useAcpSessionStore.getState();
      s.addThinking(POD, "s1", "Let me analyze this problem...");
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText("Thinking...")).toBeInTheDocument();
  });

  it("renders complete timeline in correct order", () => {
    seedSession(() => {
      const s = useAcpSessionStore.getState();
      // Seed with timestamps to ensure order
      s.addContentChunk(POD, "s1", "fix the bug", "user");
      s.addThinking(POD, "s1", "Analyzing...");
      s.addContentChunk(POD, "s1", "I see the issue.", "assistant");
      s.updateToolCall(POD, "s1", {
        toolCallId: "tc1", toolName: "edit_file", status: "running",
        argumentsJson: "{}", timestamp: 0,
      });
    });

    render(<AcpActivityStream podKey={POD} />);

    // All elements should be present
    expect(screen.getByText("fix the bug")).toBeInTheDocument();
    expect(screen.getByText("Thinking...")).toBeInTheDocument();
    expect(screen.getByText(/I see the issue/)).toBeInTheDocument();
    expect(screen.getByText("edit_file")).toBeInTheDocument();
  });
});

import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  __seedAcpSessionForTests,
  __resetAcpSessionsForTests,
} from "@/stores/acpSession";
import { EMPTY_SESSION } from "@/stores/acpSessionTypes";
import type { AcpSessionState } from "@/stores/acpSessionTypes";
import { AcpActivityStream } from "@/components/workspace/acp/AcpActivityStream";

const POD = "pod-stream";

function seedSession(partial: Partial<AcpSessionState>) {
  __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, ...partial });
}

describe("AcpActivityStream", () => {
  beforeEach(() => {
    __resetAcpSessionsForTests();
  });

  it("shows waiting message when no session exists", () => {
    render(<AcpActivityStream podKey="nonexistent" />);
    expect(screen.getByText("Waiting for ACP session...")).toBeInTheDocument();
  });

  it("renders user instruction with > prefix", () => {
    seedSession({
      messages: [{ text: "create a hello world app", role: "user", timestamp: 1 }],
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText(">")).toBeInTheDocument();
    expect(screen.getByText("create a hello world app")).toBeInTheDocument();
  });

  it("renders slash commands with distinct styling", () => {
    seedSession({
      messages: [{ text: "/compact", role: "user", timestamp: 1 }],
    });

    render(<AcpActivityStream podKey={POD} />);
    const cmd = screen.getByText("/compact");
    expect(cmd.className).toContain("font-mono");
    expect(cmd.className).toContain("text-blue");
  });

  it("renders assistant output as markdown", () => {
    seedSession({
      messages: [{ text: "I'll help you with that.", role: "assistant", timestamp: 1 }],
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText(/I'll help you with that/)).toBeInTheDocument();
  });

  it("renders tool calls in timeline", () => {
    seedSession({
      toolCalls: {
        tc1: {
          toolCallId: "tc1", toolName: "write_file", status: "completed",
          argumentsJson: '{"path":"main.ts"}', timestamp: 1,
        },
      },
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText("write_file")).toBeInTheDocument();
  });

  it("renders thinking indicator", () => {
    seedSession({
      thinkings: [{ text: "Let me analyze this problem...", timestamp: 1 }],
    });

    render(<AcpActivityStream podKey={POD} />);
    expect(screen.getByText("Thinking...")).toBeInTheDocument();
  });

  it("renders complete timeline in correct order", () => {
    seedSession({
      messages: [
        { text: "fix the bug", role: "user", timestamp: 1 },
        { text: "I see the issue.", role: "assistant", timestamp: 3 },
      ],
      thinkings: [{ text: "Analyzing...", timestamp: 2 }],
      toolCalls: {
        tc1: {
          toolCallId: "tc1", toolName: "edit_file", status: "running",
          argumentsJson: "{}", timestamp: 4,
        },
      },
    });

    render(<AcpActivityStream podKey={POD} />);

    expect(screen.getByText("fix the bug")).toBeInTheDocument();
    expect(screen.getByText("Thinking...")).toBeInTheDocument();
    expect(screen.getByText(/I see the issue/)).toBeInTheDocument();
    expect(screen.getByText("edit_file")).toBeInTheDocument();
  });
});

import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AcpToolCallCard } from "@/components/workspace/acp/AcpToolCallCard";
import type { AcpToolCall } from "@/stores/acpSession";

function makeTool(overrides: Partial<AcpToolCall> = {}): AcpToolCall {
  return {
    toolCallId: "tc-1",
    toolName: "read_file",
    status: "running",
    argumentsJson: '{"path":"src/main.ts"}',
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("AcpToolCallCard", () => {
  it("shows spinner for running tool call", () => {
    render(<AcpToolCallCard toolCall={makeTool({ status: "running" })} />);
    expect(screen.getByText("read_file")).toBeInTheDocument();
    // Loader2 has animate-spin class
    const svg = document.querySelector(".animate-spin");
    expect(svg).toBeTruthy();
  });

  it("shows neutral circle for completed but no result yet", () => {
    render(<AcpToolCallCard toolCall={makeTool({ status: "completed" })} />);
    // success is undefined → Circle icon (no animate-spin, no text-green, no text-red)
    expect(document.querySelector(".animate-spin")).toBeNull();
    expect(document.querySelector(".text-green-500")).toBeNull();
    expect(document.querySelector(".text-red-500")).toBeNull();
  });

  it("shows green check for success", () => {
    render(<AcpToolCallCard toolCall={makeTool({ status: "completed", success: true })} />);
    expect(document.querySelector(".text-green-500")).toBeTruthy();
  });

  it("shows red X for failure", () => {
    render(<AcpToolCallCard toolCall={makeTool({ status: "completed", success: false })} />);
    expect(document.querySelector(".text-red-500")).toBeTruthy();
  });

  it("expands to show arguments on click", () => {
    const tool = makeTool({ status: "completed", success: true, argumentsJson: '{"path":"test.ts"}' });
    render(<AcpToolCallCard toolCall={tool} />);

    // Arguments not visible initially
    expect(screen.queryByText('{"path":"test.ts"}')).toBeNull();

    // Click to expand
    fireEvent.click(screen.getByText("read_file"));
    expect(screen.getByText('{"path":"test.ts"}')).toBeInTheDocument();
  });

  it("shows resultText and errorMessage when expanded", () => {
    const tool = makeTool({
      status: "completed",
      success: false,
      resultText: "",
      errorMessage: "File not found",
    });
    render(<AcpToolCallCard toolCall={tool} />);

    fireEvent.click(screen.getByText("read_file"));
    expect(screen.getByText("File not found")).toBeInTheDocument();
  });
});

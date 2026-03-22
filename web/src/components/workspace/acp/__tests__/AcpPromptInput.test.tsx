import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AcpPromptInput } from "@/components/workspace/acp/AcpPromptInput";
import { relayPool } from "@/stores/relayConnection";

vi.mock("@/stores/relayConnection", () => ({
  relayPool: {
    sendAcpCommand: vi.fn(),
    isConnected: vi.fn().mockReturnValue(true),
  },
}));

describe("AcpPromptInput", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders input with correct placeholder", () => {
    render(<AcpPromptInput podKey="pod-1" />);
    expect(screen.getByPlaceholderText("Send instruction...")).toBeInTheDocument();
  });

  it("sends prompt on Enter", () => {
    render(<AcpPromptInput podKey="pod-1" />);
    const textarea = screen.getByPlaceholderText("Send instruction...");

    fireEvent.change(textarea, { target: { value: "create hello world" } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith("pod-1", {
      type: "prompt",
      prompt: "create hello world",
    });
  });

  it("does not send on Shift+Enter", () => {
    render(<AcpPromptInput podKey="pod-1" />);
    const textarea = screen.getByPlaceholderText("Send instruction...");

    fireEvent.change(textarea, { target: { value: "multiline" } });
    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: true });

    expect(relayPool.sendAcpCommand).not.toHaveBeenCalled();
  });

  it("clears input after sending", () => {
    render(<AcpPromptInput podKey="pod-1" />);
    const textarea = screen.getByPlaceholderText("Send instruction...") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "test" } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(textarea.value).toBe("");
  });

  it("does not send empty prompt", () => {
    render(<AcpPromptInput podKey="pod-1" />);
    const textarea = screen.getByPlaceholderText("Send instruction...");

    fireEvent.keyDown(textarea, { key: "Enter" });
    expect(relayPool.sendAcpCommand).not.toHaveBeenCalled();
  });

  it("shows error when not connected", () => {
    vi.mocked(relayPool.isConnected).mockReturnValue(false);

    render(<AcpPromptInput podKey="pod-1" />);
    const textarea = screen.getByPlaceholderText("Send instruction...");

    fireEvent.change(textarea, { target: { value: "test" } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(screen.getByText("Not connected")).toBeInTheDocument();
    expect(relayPool.sendAcpCommand).not.toHaveBeenCalled();
  });
});

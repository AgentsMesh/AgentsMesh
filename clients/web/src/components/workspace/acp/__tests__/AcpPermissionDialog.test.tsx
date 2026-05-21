import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@/test/test-utils";
import {
  __seedAcpSessionForTests,
  __resetAcpSessionsForTests,
  readAcpSession,
} from "@/stores/acpSession";
import { EMPTY_SESSION } from "@/stores/acpSessionTypes";
import { AcpPermissionDialog } from "@/components/workspace/acp/AcpPermissionDialog";
import { relayPool } from "@/stores/relayConnection";

vi.mock("@/stores/relayConnection", () => ({
  relayPool: {
    sendAcpCommand: vi.fn(),
    isConnected: vi.fn().mockReturnValue(true),
  },
}));

const POD = "pod-perm";

describe("AcpPermissionDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(relayPool.isConnected).mockReturnValue(true);
    __resetAcpSessionsForTests();
  });

  const perms = [
    {
      requestId: "perm-1",
      toolName: "bash",
      argumentsJson: '{"cmd":"rm -rf /tmp/test"}',
      description: "Execute: rm -rf /tmp/test",
    },
  ];

  it("renders permission request details", () => {
    vi.useFakeTimers();
    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    expect(screen.getByText("Permission Required")).toBeInTheDocument();
    expect(screen.getByText("Execute: rm -rf /tmp/test")).toBeInTheDocument();
    expect(screen.getByText("bash")).toBeInTheDocument();
    vi.useRealTimers();
  });

  it("sends approve command and removes permission", () => {
    vi.useFakeTimers();
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [...perms] });

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Approve"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, {
      type: "permission_response",
      requestId: "perm-1",
      approved: true,
    });

    const s = readAcpSession(POD) ?? EMPTY_SESSION;
    expect(s.pendingPermissions).toHaveLength(0);
    vi.useRealTimers();
  });

  it("sends deny command and removes permission", () => {
    vi.useFakeTimers();
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [...perms] });

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Deny"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, {
      type: "permission_response",
      requestId: "perm-1",
      approved: false,
    });
    vi.useRealTimers();
  });

  it("shows error when relay is not connected", () => {
    vi.useFakeTimers();
    vi.mocked(relayPool.isConnected).mockReturnValue(false);

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Approve"));

    expect(screen.getByText(/not connected/i)).toBeInTheDocument();
    expect(relayPool.sendAcpCommand).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it("renders multiple permission requests", () => {
    vi.useFakeTimers();
    const multiPerms = [
      { requestId: "p1", toolName: "bash", argumentsJson: "{}", description: "Run bash" },
      { requestId: "p2", toolName: "write_file", argumentsJson: "{}", description: "Write file" },
    ];

    render(<AcpPermissionDialog podKey={POD} permissions={multiPerms} />);
    expect(screen.getAllByText("Permission Required")).toHaveLength(2);
    expect(screen.getByText("Run bash")).toBeInTheDocument();
    expect(screen.getByText("Write file")).toBeInTheDocument();
    vi.useRealTimers();
  });

  it("renders AskUserQuestion UI for AskUserQuestion tool", () => {
    const askPerms = [
      {
        requestId: "ask-1",
        toolName: "AskUserQuestion",
        argumentsJson: JSON.stringify({
          questions: [{
            question: "Which framework?",
            header: "Framework",
            options: [
              { label: "React", description: "Frontend library" },
              { label: "Vue", description: "Progressive framework" },
            ],
            multiSelect: false,
          }],
        }),
        description: "Claude has a question",
      },
    ];

    render(<AcpPermissionDialog podKey={POD} permissions={askPerms} />);
    expect(screen.queryByText("Permission Required")).not.toBeInTheDocument();
    expect(screen.getByText("Which framework?")).toBeInTheDocument();
    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("Vue")).toBeInTheDocument();
  });

  it("sends updatedInput with AskUserQuestion answers", () => {
    const askPerm = {
      requestId: "ask-2",
      toolName: "AskUserQuestion",
      argumentsJson: JSON.stringify({
        questions: [{
          question: "Pick one?",
          header: "Choice",
          options: [
            { label: "OptionA", description: "First option" },
            { label: "OptionB", description: "Second option" },
          ],
          multiSelect: false,
        }],
      }),
      description: "Question",
    };
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [askPerm] });
    const permsFromStore = (readAcpSession(POD) ?? EMPTY_SESSION).pendingPermissions;

    render(<AcpPermissionDialog podKey={POD} permissions={permsFromStore} />);
    fireEvent.click(screen.getByText("OptionA"));
    fireEvent.click(screen.getByText("Submit"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, expect.objectContaining({
      type: "permission_response",
      requestId: "ask-2",
      approved: true,
      updatedInput: expect.objectContaining({
        answers: expect.objectContaining({ "Pick one?": "OptionA" }),
      }),
    }));
  });

  it("auto-denies permission after timeout", () => {
    vi.useFakeTimers();
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [...perms] });

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);

    act(() => { vi.advanceTimersByTime(61000); });

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, {
      type: "permission_response",
      requestId: "perm-1",
      approved: false,
    });
    vi.useRealTimers();
  });

  it("Always Allow sends updatedInput with _alwaysAllow flag", () => {
    vi.useFakeTimers();
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [...perms] });

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Always Allow"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, {
      type: "permission_response",
      requestId: "perm-1",
      approved: true,
      updatedInput: expect.objectContaining({
        updatedPermissions: expect.arrayContaining([
          expect.objectContaining({ type: "addRules", destination: "session" }),
        ]),
      }),
    });
    vi.useRealTimers();
  });

  it("handles AskUserQuestion multiSelect with multiple choices", () => {
    const askPerm = {
      requestId: "multi-1",
      toolName: "AskUserQuestion",
      argumentsJson: JSON.stringify({
        questions: [{
          question: "Which features?",
          header: "Features",
          options: [
            { label: "Auth", description: "Authentication" },
            { label: "DB", description: "Database" },
            { label: "Cache", description: "Caching" },
          ],
          multiSelect: true,
        }],
      }),
      description: "Pick features",
    };
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [askPerm] });
    const permsFromStore = (readAcpSession(POD) ?? EMPTY_SESSION).pendingPermissions;

    render(<AcpPermissionDialog podKey={POD} permissions={permsFromStore} />);
    fireEvent.click(screen.getByText("Auth"));
    fireEvent.click(screen.getByText("Cache"));
    fireEvent.click(screen.getByText("Submit"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, expect.objectContaining({
      approved: true,
      updatedInput: expect.objectContaining({
        answers: expect.objectContaining({ "Which features?": "Auth, Cache" }),
      }),
    }));
  });

  it("handles AskUserQuestion custom Other text input", () => {
    const askPerm = {
      requestId: "custom-1",
      toolName: "AskUserQuestion",
      argumentsJson: JSON.stringify({
        questions: [{
          question: "Which framework?",
          header: "Framework",
          options: [
            { label: "React", description: "Frontend lib" },
            { label: "Vue", description: "Progressive" },
          ],
          multiSelect: false,
        }],
      }),
      description: "Pick framework",
    };
    __seedAcpSessionForTests(POD, { ...EMPTY_SESSION, pendingPermissions: [askPerm] });
    const permsFromStore = (readAcpSession(POD) ?? EMPTY_SESSION).pendingPermissions;

    render(<AcpPermissionDialog podKey={POD} permissions={permsFromStore} />);
    const customInput = screen.getByPlaceholderText("Other...");
    fireEvent.change(customInput, { target: { value: "Svelte" } });
    fireEvent.click(screen.getByText("Submit"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, expect.objectContaining({
      approved: true,
      updatedInput: expect.objectContaining({
        answers: expect.objectContaining({ "Which framework?": "Svelte" }),
      }),
    }));
  });
});

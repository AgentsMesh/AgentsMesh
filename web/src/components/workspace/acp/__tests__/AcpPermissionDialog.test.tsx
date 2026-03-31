import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { useAcpSessionStore } from "@/stores/acpSession";
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
    useAcpSessionStore.setState({ sessions: {} });
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
    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    expect(screen.getByText("Permission Required")).toBeInTheDocument();
    expect(screen.getByText("Execute: rm -rf /tmp/test")).toBeInTheDocument();
    expect(screen.getByText("bash")).toBeInTheDocument();
  });

  it("sends approve command and removes permission", () => {
    // Pre-populate store
    useAcpSessionStore.getState().addPermissionRequest(POD, perms[0]);

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Approve"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, {
      type: "permission_response",
      requestId: "perm-1",
      approved: true,
    });

    // Permission removed from store
    const s = useAcpSessionStore.getState().sessions[POD];
    expect(s.pendingPermissions).toHaveLength(0);
  });

  it("sends deny command and removes permission", () => {
    useAcpSessionStore.getState().addPermissionRequest(POD, perms[0]);

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Deny"));

    expect(relayPool.sendAcpCommand).toHaveBeenCalledWith(POD, {
      type: "permission_response",
      requestId: "perm-1",
      approved: false,
    });
  });

  it("shows error when relay is not connected", () => {
    vi.mocked(relayPool.isConnected).mockReturnValue(false);

    render(<AcpPermissionDialog podKey={POD} permissions={perms} />);
    fireEvent.click(screen.getByText("Approve"));

    expect(screen.getByText(/not connected/i)).toBeInTheDocument();
    expect(relayPool.sendAcpCommand).not.toHaveBeenCalled();
  });

  it("renders multiple permission requests", () => {
    const multiPerms = [
      { requestId: "p1", toolName: "bash", argumentsJson: "{}", description: "Run bash" },
      { requestId: "p2", toolName: "write_file", argumentsJson: "{}", description: "Write file" },
    ];

    render(<AcpPermissionDialog podKey={POD} permissions={multiPerms} />);
    expect(screen.getAllByText("Permission Required")).toHaveLength(2);
    expect(screen.getByText("Run bash")).toBeInTheDocument();
    expect(screen.getByText("Write file")).toBeInTheDocument();
  });
});

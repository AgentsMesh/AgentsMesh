import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@/test/test-utils";
import { AcpPermissionModeSelector } from "@/components/workspace/acp/AcpPermissionModeSelector";
import {
  __seedAcpSessionForTests,
  __resetAcpSessionsForTests,
} from "@/stores/acpSession";
import { EMPTY_SESSION } from "@/stores/acpSessionTypes";

vi.mock("@/stores/relayConnection", () => ({
  relayPool: {
    sendAcpCommand: vi.fn(),
    isConnected: vi.fn().mockReturnValue(true),
  },
}));

function seed(podKey: string, permissionMode: string, supportedPermissionModes: string[]) {
  __seedAcpSessionForTests(podKey, {
    ...EMPTY_SESSION,
    configuration: { permissionMode, model: "", supportedPermissionModes },
  });
}

// The trigger label is t(`${currentKey}.label`) where currentKey falls back to
// "unknown" when the active mode isn't in the rendered set — so the label alone
// proves which set (advertised vs Claude fallback) is in effect, no dropdown
// expansion (and no jsdom Radix pointer plumbing) required.
describe("AcpPermissionModeSelector", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    __resetAcpSessionsForTests();
  });

  it("renders the agent's advertised mode label when capability is present", () => {
    seed("pod-1", "ask_dangerous", ["bypass", "ask_dangerous", "ask_any_write"]);
    render(<AcpPermissionModeSelector podKey="pod-1" />);
    expect(screen.getByText("Ask Risky")).toBeInTheDocument();
  });

  it("falls back to Claude modes when the agent advertises none", () => {
    seed("pod-1", "bypassPermissions", []);
    render(<AcpPermissionModeSelector podKey="pod-1" />);
    expect(screen.getByText("Bypass")).toBeInTheDocument();
  });

  it("shows the unknown placeholder when the active mode is outside the rendered set", () => {
    // A loopal mode with no advertisement → fallback set lacks it → "unknown".
    seed("pod-1", "ask_dangerous", []);
    render(<AcpPermissionModeSelector podKey="pod-1" />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

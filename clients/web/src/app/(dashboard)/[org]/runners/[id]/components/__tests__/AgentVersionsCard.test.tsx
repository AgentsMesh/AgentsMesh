import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { NextIntlClientProvider } from "next-intl";
import type { RunnerData } from "@/lib/viewModels/runner";
import { AgentVersionsCard } from "../AgentVersionsCard";
import runnersMessages from "@/messages/en/runners.json";
import commonMessages from "@/messages/en/common.json";

// Render with REAL en messages (not the usual `t => key` mock) so a missing
// i18n key surfaces as the raw key path in the DOM — that's what catches the
// agentUpgradeUnsupported regression. The button/gate assertions cover the
// managers-null fallback + per-manager visibility.
const messages = { ...runnersMessages, ...commonMessages };

vi.mock("next/navigation", () => ({ useParams: () => ({ org: "test-org" }) }));

const mockUpgradeAgent = vi.fn();
vi.mock("@/lib/api/facade/runnerConnect", () => ({
  upgradeAgent: (...a: unknown[]) => mockUpgradeAgent(...a),
}));

const mockListAgents = vi.fn();
vi.mock("@/lib/api/facade/agentConnect", () => ({
  listAgents: (...a: unknown[]) => mockListAgents(...a),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/components/ui/confirm-dialog", () => ({
  useConfirmDialog: () => ({ dialogProps: {}, confirm: () => Promise.resolve(true) }),
  ConfirmDialog: () => null,
}));

function renderCard(runner: Partial<RunnerData>) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages} timeZone="UTC">
      <AgentVersionsCard
        runner={{ id: 1, status: "online", agent_versions: [], available_agents: [], ...runner } as RunnerData}
      />
    </NextIntlClientProvider>,
  );
}

describe("AgentVersionsCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListAgents.mockResolvedValue({ builtin_agents: [] });
  });

  it("renders nothing when the runner has no agents", () => {
    const { container } = renderCard({ agent_versions: [], available_agents: [] });
    expect(container).toBeEmptyDOMElement();
  });

  it("shows version, manager badge and Upgrade button for an agent with a manager", async () => {
    mockListAgents.mockResolvedValue({ builtin_agents: [{ slug: "claude-code", upgrade_manager: "npm" }] });
    renderCard({ agent_versions: [{ slug: "claude-code", version: "1.2.3" }] });
    await waitFor(() => expect(screen.getByText("npm")).toBeInTheDocument());
    expect(screen.getByText("claude-code")).toBeInTheDocument();
    expect(screen.getByText("v1.2.3")).toBeInTheDocument();
    expect(screen.getByText("Upgrade")).toBeInTheDocument();
  });

  it("renders the TRANSLATED unsupported label (not the raw key) for agents without a manager", async () => {
    renderCard({ available_agents: ["cursor-cli"] });
    // Once listAgents resolves with no manager for cursor-cli, it gates to unsupported.
    await waitFor(() =>
      expect(screen.getByText("This agent does not support remote upgrade.")).toBeInTheDocument(),
    );
    // Regression guard for the missing-key bug: the raw path must never render.
    expect(screen.queryByText(/agentUpgradeUnsupported/)).not.toBeInTheDocument();
    expect(screen.queryByText("Upgrade")).not.toBeInTheDocument();
  });

  it("falls back to showing the button when listAgents fails (managers stays null)", async () => {
    mockListAgents.mockRejectedValue(new Error("network"));
    renderCard({ available_agents: ["cursor-cli"] });
    await waitFor(() => expect(screen.getByText("Upgrade")).toBeInTheDocument());
    expect(screen.queryByText("This agent does not support remote upgrade.")).not.toBeInTheDocument();
  });

  it("calls upgradeAgent with org/runner/slug on click", async () => {
    mockListAgents.mockResolvedValue({ builtin_agents: [{ slug: "claude-code", upgrade_manager: "npm" }] });
    mockUpgradeAgent.mockResolvedValue({ request_id: "r1", message: "ok" });
    renderCard({ agent_versions: [{ slug: "claude-code", version: "1.2.3" }] });
    await waitFor(() => screen.getByText("Upgrade"));
    fireEvent.click(screen.getByText("Upgrade"));
    await waitFor(() => expect(mockUpgradeAgent).toHaveBeenCalledWith("test-org", 1, "claude-code"));
  });
});

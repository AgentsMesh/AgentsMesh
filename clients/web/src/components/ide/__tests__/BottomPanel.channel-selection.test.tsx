import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { BottomPanel } from "../BottomPanel";
import { useIDEStore } from "@/stores/ide";
import { useChannelStore } from "@/stores/channel";

vi.mock("next-intl", () => ({ useTranslations: () => (key: string) => key }));
vi.mock("@/stores/auth", () => ({
  useCurrentOrg: () => ({ slug: "test-org" }),
  readCurrentOrg: () => ({ slug: "test-org" }),
  readCurrentUser: () => ({ id: 1 }),
}));
vi.mock("@/components/autopilot", () => ({ AutopilotPanelContent: () => null }));
vi.mock("../useBottomPanelData", () => ({
  useBottomPanelData: () => ({
    selectedPodKey: "pod-1",
    currentPod: null,
    activeAutopilot: null,
    topology: { nodes: [], edges: [], channels: [] },
    fetchTopology: vi.fn(),
    podChannels: [{ id: 42, name: "general", pod_keys: ["pod-1"] }],
    incomingBindings: [],
    outgoingBindings: [],
    getPodInfo: vi.fn(),
  }),
}));
vi.mock("../BottomPanel/index", () => ({
  ChannelsTabContent: ({ selectedChannelId, onChannelClick, onBackToList }: {
    selectedChannelId: number | null;
    onChannelClick: (id: number) => void;
    onBackToList: () => void;
  }) => selectedChannelId ? (
    <button onClick={onBackToList}>back-to-list</button>
  ) : (
    <button onClick={() => onChannelClick(42)}>general</button>
  ),
  ActivityTabContent: () => null,
  DeliveryTabContent: () => null,
  InfoTabContent: () => null,
}));

describe("BottomPanel channel selection (panel-local)", () => {
  beforeEach(() => {
    useIDEStore.setState({ bottomPanelOpen: true, bottomPanelTab: "channels", bottomPanelHeight: 200 });
    useChannelStore.setState({ selectedChannelId: null });
  });

  it("opens the channel detail view locally", () => {
    render(<BottomPanel />);

    fireEvent.click(screen.getByText("general"));

    expect(screen.getByText("back-to-list")).toBeInTheDocument();
  });

  it("returns to the channel list on back", () => {
    render(<BottomPanel />);
    fireEvent.click(screen.getByText("general"));

    fireEvent.click(screen.getByText("back-to-list"));

    expect(screen.getByText("general")).toBeInTheDocument();
  });

  it("returns to the channel list when switching tabs", () => {
    render(<BottomPanel />);
    fireEvent.click(screen.getByText("general"));

    fireEvent.click(screen.getByText("ide.bottomPanel.activity"));
    fireEvent.click(screen.getByText("ide.bottomPanel.channels"));

    expect(screen.getByText("general")).toBeInTheDocument();
  });

  it("never touches the app-global selectedChannelId", () => {
    render(<BottomPanel />);

    fireEvent.click(screen.getByText("general"));
    fireEvent.click(screen.getByText("back-to-list"));

    // The docked panel is a local peek surface; the /channels page and IDE
    // sidebar selection must survive interacting with it.
    expect(useChannelStore.getState().selectedChannelId).toBeNull();
  });
});

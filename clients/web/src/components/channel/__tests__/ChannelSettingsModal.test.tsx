import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@/test/test-utils";
import { ChannelSettingsModal } from "@/components/channel/ChannelSettingsModal";
import { useChannelStore } from "@/stores/channelStore";

vi.mock("@/stores/auth", async () => {
  const actual = await vi.importActual<typeof import("@/stores/auth")>("@/stores/auth");
  return {
    ...actual,
    useCurrentUser: () => ({ id: 42, email: "test@example.com", username: "tester" }),
    useCurrentOrg: () => ({ id: 7, name: "Acme", slug: "acme", role: "member" }),
  };
});

describe("ChannelSettingsModal", () => {
  beforeEach(() => {
    useChannelStore.setState({
      deleteChannel: vi.fn().mockResolvedValue(undefined),
      fetchChannels: vi.fn().mockResolvedValue(undefined),
    });
  });

  it("lets a channel creator delete their channel", async () => {
    const remove = useChannelStore.getState().deleteChannel;
    render(
      <ChannelSettingsModal
        open
        onOpenChange={vi.fn()}
        channel={{ id: 3, name: "created-by-me", is_archived: false, created_by_user_id: 42 }}
      />,
    );

    fireEvent.click(screen.getByText("Danger"));
    fireEvent.click(screen.getByTestId("channel-settings-delete"));

    await waitFor(() => expect(remove).toHaveBeenCalledWith(3));
  });

  it("hides delete from non-creator members", () => {
    render(
      <ChannelSettingsModal
        open
        onOpenChange={vi.fn()}
        channel={{ id: 4, name: "someone-elses", is_archived: false, created_by_user_id: 99 }}
      />,
    );

    expect(screen.queryByText("Danger")).not.toBeInTheDocument();
  });
});

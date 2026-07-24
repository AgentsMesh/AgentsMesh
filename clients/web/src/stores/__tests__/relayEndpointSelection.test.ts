import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getPodConnection: vi.fn(),
  readCurrentOrg: vi.fn(),
  getLocalRunnerService: vi.fn(),
  probeRelayOpen: vi.fn(),
}));

vi.mock("@/lib/api/facade/podConnect", () => ({
  getPodConnection: mocks.getPodConnection,
}));
vi.mock("@/stores/auth", () => ({ readCurrentOrg: mocks.readCurrentOrg }));
vi.mock("@agentsmesh/service-runtime", () => ({
  getLocalRunnerService: mocks.getLocalRunnerService,
}));
vi.mock("../relayProbe", () => ({ probeRelayOpen: mocks.probeRelayOpen }));

let selectRelayEndpoint: typeof import("../relayEndpointSelection").selectRelayEndpoint;

const remote = {
  relay_url: "wss://remote.example/relay",
  token: "remote-token",
  pod_key: "pod-1",
};

describe("selectRelayEndpoint", () => {
  beforeEach(async () => {
    vi.resetModules();
    vi.clearAllMocks();
    mocks.readCurrentOrg.mockReturnValue({ slug: "acme" });
    mocks.probeRelayOpen.mockResolvedValue(true);
    mocks.getLocalRunnerService.mockReturnValue(null);
    mocks.getPodConnection.mockResolvedValue(remote);
    ({ selectRelayEndpoint } = await import("../relayEndpointSelection"));
  });

  it("uses the current org and falls back when no complete local endpoint exists", async () => {
    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: remote.relay_url,
      token: remote.token,
    });
    expect(mocks.getPodConnection).toHaveBeenCalledWith("acme", "pod-1");
    expect(mocks.getLocalRunnerService).not.toHaveBeenCalled();
    expect(mocks.probeRelayOpen).not.toHaveBeenCalled();

    mocks.readCurrentOrg.mockReturnValue(null);
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://local.example/relay",
    });
    await selectRelayEndpoint("pod-2");
    expect(mocks.getPodConnection).toHaveBeenLastCalledWith("", "pod-2");
    expect(mocks.probeRelayOpen).not.toHaveBeenCalled();
  });

  it("accepts an unscoped local endpoint after a successful probe", async () => {
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local token",
    });

    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: "ws://127.0.0.1:8081",
      token: "local token",
    });
    expect(mocks.probeRelayOpen).toHaveBeenCalledWith(
      "ws://127.0.0.1:8081",
      "local token",
      1000,
    );
    expect(mocks.getLocalRunnerService).not.toHaveBeenCalled();
  });

  it("rejects a node-scoped local endpoint when no local runner service exists", async () => {
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-a",
    });

    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: remote.relay_url,
      token: remote.token,
    });
    expect(mocks.getLocalRunnerService).toHaveBeenCalledTimes(1);
    expect(mocks.probeRelayOpen).not.toHaveBeenCalled();
  });

  it("falls back if the local runner disappears between host checks", async () => {
    mocks.getLocalRunnerService
      .mockReturnValueOnce({ local_node_id: vi.fn() })
      .mockReturnValueOnce(null);
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-a",
    });

    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: remote.relay_url,
      token: remote.token,
    });
    expect(mocks.probeRelayOpen).not.toHaveBeenCalled();
  });

  it("does not cache an empty pre-onboarding node ID", async () => {
    const localNodeId = vi.fn().mockResolvedValue(null);
    mocks.getLocalRunnerService.mockReturnValue({ local_node_id: localNodeId });
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-a",
    });

    await selectRelayEndpoint("pod-1");
    await selectRelayEndpoint("pod-1");

    expect(localNodeId).toHaveBeenCalledTimes(2);
    expect(mocks.probeRelayOpen).not.toHaveBeenCalled();
  });

  it("clears a rejected node lookup so a later attempt can recover", async () => {
    const localNodeId = vi.fn()
      .mockRejectedValueOnce(new Error("runner starting"))
      .mockResolvedValueOnce("node-a");
    mocks.getLocalRunnerService.mockReturnValue({ local_node_id: localNodeId });
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-a",
    });

    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: remote.relay_url,
      token: remote.token,
    });
    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: "ws://127.0.0.1:8081",
      token: "local-token",
    });
    expect(localNodeId).toHaveBeenCalledTimes(2);
  });

  it("caches a resolved node ID and selects a same-host local relay", async () => {
    const localNodeId = vi.fn().mockResolvedValue("node-a");
    mocks.getLocalRunnerService.mockReturnValue({ local_node_id: localNodeId });
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-a",
    });

    await selectRelayEndpoint("pod-1");
    await selectRelayEndpoint("pod-2");

    expect(localNodeId).toHaveBeenCalledTimes(1);
    expect(mocks.probeRelayOpen).toHaveBeenCalledTimes(2);
  });

  it("falls back for a different host or a failed local probe", async () => {
    const localNodeId = vi.fn().mockResolvedValue("node-a");
    mocks.getLocalRunnerService.mockReturnValue({ local_node_id: localNodeId });
    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-b",
    });

    await expect(selectRelayEndpoint("pod-1")).resolves.toEqual({
      url: remote.relay_url,
      token: remote.token,
    });
    expect(mocks.probeRelayOpen).not.toHaveBeenCalled();

    mocks.getPodConnection.mockResolvedValue({
      ...remote,
      local_relay_url: "ws://127.0.0.1:8081",
      local_token: "local-token",
      local_relay_node_id: "node-a",
    });
    mocks.probeRelayOpen.mockResolvedValue(false);
    await expect(selectRelayEndpoint("pod-2")).resolves.toEqual({
      url: remote.relay_url,
      token: remote.token,
    });
    expect(mocks.probeRelayOpen).toHaveBeenCalledOnce();
  });
});

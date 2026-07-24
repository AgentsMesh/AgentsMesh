import { getPodConnection } from "@/lib/api/facade/podConnect";
import { readCurrentOrg } from "@/stores/auth";
import { getLocalRunnerService } from "@agentsmesh/service-runtime";
import { probeRelayOpen } from "./relayProbe";

export type RelayEndpoint = Readonly<{ url: string; token: string }>;

export async function selectRelayEndpoint(podKey: string): Promise<RelayEndpoint> {
  const info = await getPodConnection(readCurrentOrg()?.slug ?? "", podKey);
  if (
    info.local_relay_url &&
    info.local_token &&
    (await isSameHostRunner(info.local_relay_node_id)) &&
    (await probeRelayOpen(info.local_relay_url, info.local_token, 1000))
  ) {
    return { url: info.local_relay_url, token: info.local_token };
  }
  return { url: info.relay_url, token: info.token };
}

// Cache only resolved non-empty IDs — pre-onboarding null must not pin the
// renderer to "different host" after the local runner finishes onboarding.
let cachedNodeIdPromise: Promise<string | null> | null = null;

async function resolveLocalNodeId(): Promise<string | null> {
  const service = getLocalRunnerService();
  if (!service) return null;
  if (!cachedNodeIdPromise) {
    cachedNodeIdPromise = service.local_node_id().then(
      (id: string | null) => {
        if (!id) cachedNodeIdPromise = null;
        return id;
      },
      () => {
        cachedNodeIdPromise = null;
        return null;
      },
    );
  }
  return cachedNodeIdPromise;
}

async function isSameHostRunner(advertisedNodeId: string | undefined): Promise<boolean> {
  if (!advertisedNodeId) return true;
  if (!getLocalRunnerService()) return false;
  const localNodeId = await resolveLocalNodeId();
  return localNodeId !== null && localNodeId === advertisedNodeId;
}

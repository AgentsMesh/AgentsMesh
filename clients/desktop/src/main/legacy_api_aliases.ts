import { ipcMain } from "electron";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import { ReplaceChannelPodsRequestSchema } from "@proto/channel_state/v1/mutations_pb";
import { PodSchema } from "@proto/pod/v1/pod_pb";
import { type AppState } from "@agentsmesh/node-bridge";
import { makeConnectCaller } from "./connect_call";
import { snakeCaseDeep, coerceInt64, ensureChannelDefaults, toIdNumber } from "./connect-json";

const LEGACY_ALIAS_HANDLERS = [
  "userGetMe",
  "autopilotFetchControllers",
  "channelCreateChannel",
  "channelJoinChannel",
  "channelLeaveChannel",
  "channelGetChannelPods",
  "orgCreatePersonal",
  "agentListAgents",
  "repositoryList",
  "runnerFetchRunners",
  "podCreatePod",
  "podTerminatePod",
] as const;

interface LegacyAliasDeps {
  // Getters (not values) because AppState + the resolved API URL are rebound on
  // server switch — handlers must read the current ones each call.
  getAppState: () => AppState;
  getApiUrl: () => string;
}

// Legacy IPC aliases for method names that predate the R6 Connect-RPC refactor.
// Desktop e2e specs still invoke `userGetMe` / `autopilotFetchControllers` /
// `runnerFetchRunners` / `channelCreateChannel` by name. The Rust napi handlers
// were renamed (and switched to proto binary payloads) without an alias hop, so
// the invokes hit `No handler registered`. We forward through the Connect JSON
// wire here — it preserves the failure-surface details the orbstack-port-conflict
// spec depends on (status + URL in the error message) and avoids dragging
// proto-js into the main bundle.
export function registerLegacyApiAliases(
  { getAppState, getApiUrl }: LegacyAliasDeps,
  tracked: Set<string>,
): void {
  for (const ch of LEGACY_ALIAS_HANDLERS) {
    ipcMain.removeHandler(ch);
    tracked.add(ch);
  }
  const { callConnectJson, orgSlug } = makeConnectCaller({ getAppState, getApiUrl });

  ipcMain.handle("userGetMe", () =>
    callConnectJson("proto.user.v1.UserService", "GetMe"),
  );
  ipcMain.handle("autopilotFetchControllers", () =>
    callConnectJson(
      "proto.autopilot.v1.AutopilotControllerService",
      "ListAutopilotControllers",
      { orgSlug: orgSlug() },
    ),
  );

  // R6 dropped the Rust channel_join_channel napi (replaced by direct
  // ChannelService.JoinChannelPod). Renderer paths use wasm Connect; main
  // exposes these aliases for desktop e2e specs that still invoke through IPC by
  // name. The cache update is necessary because main's AppState holds a separate
  // Rust state from the renderer wasm — without the app_channel_replace_pods
  // fan-out into runtime.state, appChannelPodsJson returns empty.
  ipcMain.handle("channelCreateChannel", async (_e, requestJson: string) => {
    const req = JSON.parse(requestJson) as Record<string, unknown>;
    const raw = await callConnectJson(
      "proto.channel.v1.ChannelService",
      "CreateChannel",
      { orgSlug: orgSlug(), ...req },
    );
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return JSON.stringify(ensureChannelDefaults(coerceInt64(snakeCaseDeep(parsed)) as Record<string, unknown>));
  });

  const refreshChannelPodsCache = async (channelId: number): Promise<void> => {
    const raw = await callConnectJson(
      "proto.channel.v1.ChannelService",
      "ListChannelPods",
      { orgSlug: orgSlug(), id: channelId },
    );
    const parsed = JSON.parse(raw) as { items?: Array<Record<string, unknown>> };
    // Wrap the 5 ChannelPod summary fields into proto.pod.v1.Pod (other fields
    // default to proto3 zeros) and dispatch via the cross-domain SSOT mutator.
    const protoPods = (parsed.items ?? []).map((p) => protoCreate(PodSchema, {
      id: BigInt(typeof p.id === "number" ? p.id : Number(p.id ?? 0)),
      podKey: String(p.podKey ?? p.pod_key ?? ""),
      alias: p.alias != null ? String(p.alias) : undefined,
      status: String(p.status ?? ""),
      agentStatus: String(p.agentStatus ?? p.agent_status ?? ""),
    }));
    const req = protoCreate(ReplaceChannelPodsRequestSchema, {
      channelId: BigInt(channelId),
      pods: protoPods,
    });
    const bytes = toBinary(ReplaceChannelPodsRequestSchema, req);
    // napi-rs Vec<u8> wants Array<number> over the JS boundary (Uint8Array is
    // seen as an opaque object without a length accessor through serialised IPC).
    await (getAppState() as { appChannelReplacePods: (b: number[]) => Promise<void> })
      .appChannelReplacePods(Array.from(bytes));
  };

  const fetchChannelEnvelope = async (channelId: number): Promise<string> => {
    const channelRaw = await callConnectJson(
      "proto.channel.v1.ChannelService",
      "GetChannel",
      { orgSlug: orgSlug(), id: channelId },
    );
    const parsed = JSON.parse(channelRaw) as Record<string, unknown>;
    const coerced = coerceInt64(snakeCaseDeep(parsed)) as Record<string, unknown>;
    return JSON.stringify(ensureChannelDefaults(coerced));
  };

  ipcMain.handle("channelJoinChannel", async (_e, rawId: number | string, podKey: string) => {
    const channelId = toIdNumber(rawId);
    await callConnectJson(
      "proto.channel.v1.ChannelService",
      "JoinChannelPod",
      { orgSlug: orgSlug(), id: channelId, podKey },
    );
    await refreshChannelPodsCache(channelId);
    return fetchChannelEnvelope(channelId);
  });

  ipcMain.handle("channelLeaveChannel", async (_e, rawId: number | string, podKey: string) => {
    const channelId = toIdNumber(rawId);
    await callConnectJson(
      "proto.channel.v1.ChannelService",
      "LeaveChannelPod",
      { orgSlug: orgSlug(), id: channelId, podKey },
    );
    await refreshChannelPodsCache(channelId);
    return fetchChannelEnvelope(channelId);
  });

  ipcMain.handle("channelGetChannelPods", async (_e, rawId: number | string) => {
    const channelId = toIdNumber(rawId);
    await refreshChannelPodsCache(channelId);
    // ElectronChannelService.get_channel_pods expects `{pods, total}`; the Rust
    // cache JSON is a bare array — wrap it back into the legacy shape.
    const cacheJson = await (getAppState() as { appChannelPodsJson: (id: number) => Promise<string> })
      .appChannelPodsJson(channelId);
    const pods = JSON.parse(cacheJson) as unknown[];
    return JSON.stringify({ pods, total: pods.length });
  });

  ipcMain.handle("orgCreatePersonal", () =>
    callConnectJson("proto.org.v1.OrgService", "CreatePersonalOrg", {}),
  );

  // Renderer parses `{builtin_agents, custom_agents}` (snake_case); Connect emits
  // proto camelCase — remap + rename nested keys recursively.
  ipcMain.handle("agentListAgents", async () => {
    const raw = await callConnectJson("proto.agent.v1.AgentService", "ListAgents", { orgSlug: orgSlug() });
    const parsed = JSON.parse(raw) as {
      builtinAgents?: unknown[]; customAgents?: unknown[];
      builtin_agents?: unknown[]; custom_agents?: unknown[];
    };
    return JSON.stringify({
      builtin_agents: (parsed.builtin_agents ?? parsed.builtinAgents ?? []).map(snakeCaseDeep),
      custom_agents: (parsed.custom_agents ?? parsed.customAgents ?? []).map(snakeCaseDeep),
    });
  });

  // Renderer cache expects `{repositories: [...]}`; Connect returns `{items}`.
  ipcMain.handle("repositoryList", async () => {
    const raw = await callConnectJson("proto.repository.v1.RepositoryService", "ListRepositories", { orgSlug: orgSlug() });
    const parsed = JSON.parse(raw) as { items?: unknown[] };
    return JSON.stringify({ repositories: (parsed.items ?? []).map(snakeCaseDeep) });
  });

  // e2e seeds pods via `runnerFetchRunners`; remap `{items}`→`{runners}`,
  // snake_case nested fields, coerce int64 `id` (reused as runner_id).
  ipcMain.handle("runnerFetchRunners", async () => {
    const raw = await callConnectJson("proto.runner_api.v1.RunnerService", "ListRunners", { orgSlug: orgSlug() });
    const parsed = JSON.parse(raw) as { items?: unknown[] };
    return JSON.stringify({
      runners: (parsed.items ?? []).map((r) => coerceInt64(snakeCaseDeep(r))),
    });
  });

  // Renderer hands snake_case (legacy REST); Connect JSON accepts both shapes.
  ipcMain.handle("podCreatePod", async (_e, requestJson: string) => {
    const req = JSON.parse(requestJson) as Record<string, unknown>;
    const raw = await callConnectJson("proto.pod.v1.PodService", "CreatePod", { orgSlug: orgSlug(), ...req });
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return JSON.stringify(snakeCaseDeep(parsed));
  });

  ipcMain.handle("podTerminatePod", async (_e, podKey: string) => {
    await callConnectJson("proto.pod.v1.PodService", "TerminatePod", { orgSlug: orgSlug(), podKey });
  });
}

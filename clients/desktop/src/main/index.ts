import { app, BrowserWindow, dialog, ipcMain, Menu, shell } from "electron";
import path from "path";
import { AppState, initLogger, logEvent } from "@agentsmesh/node-bridge";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import { ReplaceChannelPodsRequestSchema } from "@proto/channel_state/v1/mutations_pb";
import { PodSchema } from "@proto/pod/v1/pod_pb";
import { createLocalRunnerStubs, type LocalRunnerStubMap } from "./local_runner_stubs";
import { acquireSingleInstance } from "./single_instance";
import { IPC_ALLOWLIST, IPC_ALLOWLIST_SET } from "./ipc-allowlist.generated";
import { setupRealtimeBridge, type RealtimeBridge } from "./realtime";
import { setupRelayBridge, type RelayBridge } from "./relay";
import { connectFetch } from "./connect-fetch";
import {
  registerProtocol,
  attachSecondInstanceUrlHandler,
  installOpenUrlHandler,
  captureColdLaunchUrl,
  flushPendingUrl,
} from "./oauth_deep_link";
import * as serverConfig from "./server_config";
import { isValidServerUrl } from "../shared/server-config-types";

// Cold-start resolution: (1) AGENTSMESH_API_URL env override (dev/e2e, process-only,
// does NOT pollute SSOT) → (2) activeUrl(currentCfg). After save, serverConfig:set
// rebinds directly, shadowing env. Env-override lives here (not server_config.ts) to keep
// that module pure.
// pendingStartupErrorMsg is module-load stashed; flushed in whenReady because
// dialog.showErrorBox is unreliable before app.ready fires.
let pendingStartupErrorMsg: string | null = null;

function resolveColdStartApiUrl(cfg: serverConfig.ServerConfig): string {
  const envOverride = process.env.AGENTSMESH_API_URL;
  if (envOverride && isValidServerUrl(envOverride)) return envOverride;
  try {
    return serverConfig.activeUrl(cfg);
  } catch (e) {
    pendingStartupErrorMsg = (e as Error).message;
    return serverConfig.activeUrl(serverConfig.DEFAULT);
  }
}

let currentCfg = serverConfig.load();
let currentApiUrl = resolveColdStartApiUrl(currentCfg);

const storageDir = path.join(app.getPath("userData"), "agentsmesh");
const logsDir = app.getPath("logs");

const isHeadlessTest = process.env.NODE_ENV === "test";

let appState: AppState;
let stubs: LocalRunnerStubMap | null = null;
let mainWindow: BrowserWindow | null = null;
let realtimeBridge: RealtimeBridge | null = null;
let relayBridge: RelayBridge | null = null;
const appStateHandlers = new Set<string>();

const getMainWindow = () => mainWindow;

if (acquireSingleInstance()) {
  registerProtocol();
  attachSecondInstanceUrlHandler(getMainWindow);
  captureColdLaunchUrl();
}

function createWindow() {
  const win = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    title: "AgentsMesh",
    show: !isHeadlessTest,
    paintWhenInitiallyHidden: true,
    skipTaskbar: isHeadlessTest,
    webPreferences: {
      preload: path.join(__dirname, "../preload/index.js"),
      contextIsolation: true,
      sandbox: false,
      nodeIntegration: false,
    },
  });

  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:\/\//i.test(url) || url.startsWith("mailto:") || url.startsWith("agentsmesh://")) {
      shell.openExternal(url);
    }
    return { action: "deny" };
  });

  if (process.env.ELECTRON_RENDERER_URL) {
    win.loadURL(process.env.ELECTRON_RENDERER_URL);
    win.webContents.openDevTools({ mode: "detach" });
  } else {
    win.loadFile(path.join(__dirname, "../renderer/index.html"));
  }

  mainWindow = win;
  win.on("closed", () => {
    if (mainWindow === win) mainWindow = null;
  });
  flushPendingUrl(getMainWindow);
}

// Called at boot and after server switch (new AppState). MUST removeHandler first
// because ipcMain.handle throws on duplicate registration.
//
// Allowlist-driven (vs reflect-everything): the set of methods exposed as IPC
// channels comes from `ipc-allowlist.generated.ts` — auto-generated from the
// same NAPI binary symbol enumeration that drives e2e contract specs, so the
// two stay in lock-step. Any NAPI method not in the allowlist is unreachable
// from the renderer even if it exists on AppState.prototype. Drift (allowlist
// references a method that's missing from AppState, or vice-versa) is logged
// at boot.
function bindAppStateHandlers() {
  for (const ch of appStateHandlers) {
    ipcMain.removeHandler(ch);
  }
  appStateHandlers.clear();

  const proto = Object.getPrototypeOf(appState);
  const protoMethods = new Set(
    Object.getOwnPropertyNames(proto).filter(
      (k) => k !== "constructor" && typeof (appState as any)[k] === "function",
    ),
  );

  // Drift detection: warn (don't crash) on either side mismatching the
  // allowlist. Crash would block legitimate dev workflows after a Bazel
  // rebuild lag; a warn surfaces the issue immediately in the dev log
  // while keeping renderer paths working.
  const missingFromBinary: string[] = [];
  for (const name of IPC_ALLOWLIST) {
    if (!protoMethods.has(name)) missingFromBinary.push(name);
  }
  const missingFromAllowlist: string[] = [];
  for (const name of protoMethods) {
    if (!IPC_ALLOWLIST_SET.has(name)) missingFromAllowlist.push(name);
  }
  if (missingFromBinary.length > 0) {
    console.warn(
      `[electron] IPC allowlist drift: ${missingFromBinary.length} methods listed but not in AppState.prototype — regenerate with \`pnpm --filter desktop e2e:gen\``,
    );
  }
  if (missingFromAllowlist.length > 0) {
    console.warn(
      `[electron] IPC allowlist drift: ${missingFromAllowlist.length} AppState methods not in allowlist (denied to renderer) — regenerate with \`pnpm --filter desktop e2e:gen\``,
    );
  }

  let registered = 0;
  for (const m of IPC_ALLOWLIST) {
    if (!protoMethods.has(m)) continue; // skip missing methods (logged above)
    ipcMain.handle(m, async (_e, ...args: unknown[]) => {
      try {
        if (stubs && m in stubs) {
          return await stubs[m](...args);
        }
        return await (appState as any)[m](...args);
      } catch (err) {
        throw typeof err === "string" ? new Error(err) : err;
      }
    });
    appStateHandlers.add(m);
    registered++;
  }
  console.log(`[electron] Registered ${registered} IPC handlers (allowlist: ${IPC_ALLOWLIST.length}, AppState methods: ${protoMethods.size})`);
}

// Known leak: LocalRunnerManager / RelayManager have no shutdown hook, so their tokio
// tasks may outlive the rebind. Rare in practice (server switches are uncommon).
function rebindAppState(newApiUrl: string) {
  console.log(`[electron] Rebinding AppState: ${currentApiUrl} → ${newApiUrl}`);
  // Dispose old realtime bridge before swapping AppState — its NAPI
  // callbacks would otherwise keep firing into a stale Rust handle.
  if (realtimeBridge) {
    void realtimeBridge.dispose();
    realtimeBridge = null;
  }
  if (relayBridge) {
    relayBridge.dispose();
    relayBridge = null;
  }
  appState = new AppState(newApiUrl, storageDir);
  if (isHeadlessTest) {
    stubs = createLocalRunnerStubs();
  }
  bindAppStateHandlers();
  void setupRealtimeBridge(appState, getMainWindow).then((bridge) => {
    realtimeBridge = bridge;
  });
  relayBridge = setupRelayBridge(appState, getMainWindow);
  currentApiUrl = newApiUrl;
}

function registerStaticHandlers() {
  ipcMain.handle("shellOpen", async (_e, url: string) => {
    if (/^https?:\/\//i.test(url) || url.startsWith("mailto:") || url.startsWith("agentsmesh://")) {
      await shell.openExternal(url);
    }
  });

  // Renderer log forwarding into rolling file (renderer + main + Rust in timestamp order).
  ipcMain.handle("core:log", (_e, level: string, target: string, msg: string) => {
    logEvent(level, target, msg);
  });

  ipcMain.handle("logs:openFolder", async () => {
    await shell.openPath(logsDir);
  });

  // Sync IPC by design: preload populates window.electronAPI.apiUrl synchronously at boot
  // (before any renderer code runs — no UI thread to block). mainWindow.reload() re-runs
  // preload to propagate serverConfig:set updates.
  // Invariant: registerStaticHandlers() MUST run before createWindow() (enforced by ordering below).
  ipcMain.on("serverConfig:getActiveUrlSync", (e) => {
    e.returnValue = currentApiUrl;
  });
  ipcMain.on("serverConfig:getSync", (e) => {
    e.returnValue = currentCfg;
  });
  ipcMain.handle("serverConfig:get", () => serverConfig.load());
  ipcMain.handle("serverConfig:set", async (_e, raw: serverConfig.ServerConfig) => {
    // raw is type-erased at IPC boundary. activeUrl throws on invalid custom URL (propagates
    // back to dialog). MUST use save()'s return value as currentCfg (round-3 bug: assigning
    // raw left untrimmed fields in memory).
    serverConfig.activeUrl(raw); // validate, throw on invalid
    const next = serverConfig.save(raw);
    const newUrl = serverConfig.activeUrl(next);
    currentCfg = next;
    if (newUrl !== currentApiUrl) {
      rebindAppState(newUrl);
    }
    // Always reload: even when resolved URL is unchanged (env override masks both), the cfg
    // fields the dialog reads can have changed (kind/custom*). Reload re-snapshots preload.
    mainWindow?.reload();
  });

  registerLegacyApiAliases();
}

// Recursively rename object keys from camelCase to snake_case. Used by
// the legacy IPC aliases so renderer code written against the REST shape
// can keep reading snake_case fields off the Connect-JSON envelope.
function snakeCaseDeep(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(snakeCaseDeep);
  if (value !== null && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      const sk = k.replace(/([A-Z])/g, "_$1").toLowerCase();
      out[sk] = snakeCaseDeep(v);
    }
    return out;
  }
  return value;
}

// Legacy IPC aliases for method names that predate the R6 Connect-RPC
// refactor. Desktop e2e specs still invoke `userGetMe` /
// `autopilotFetchControllers` / `runnerFetchRunners` /
// `channelCreateChannel` by name. The Rust napi handlers were renamed
// (and switched to proto binary payloads) without an alias hop, so the
// invokes hit `No handler registered`. We forward through the Connect
// JSON wire here — it preserves the failure-surface details the
// orbstack-port-conflict spec depends on (status + URL in the error
// message) and avoids dragging proto-js into the main bundle.
function registerLegacyApiAliases() {
  const callConnectJson = async (
    service: string,
    method: string,
    payload: unknown = {},
  ): Promise<string> => {
    const url = `${currentApiUrl}/${service}/${method}`;
    const token = (appState as { authGetToken?: () => string | null }).authGetToken?.();
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "Connect-Protocol-Version": "1",
    };
    if (token) headers.Authorization = `Bearer ${token}`;
    const res = await connectFetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const body = await res.text().catch(() => "");
      // Surface the standard `auth_expired` token the desktop renderer +
      // e2e specs key off when the backend returned an Unauthorized.
      // The Connect-JSON error envelope is `{"code":"unauthenticated", ...}`
      // — rewrite the code so callers don't need to know two vocabularies.
      const message = body.includes("unauthenticated")
        ? `auth_expired ${res.status} ${url} ${body}`
        : `${res.status} ${res.statusText} ${url} ${body}`;
      throw new Error(message.trim());
    }
    return await res.text();
  };

  const orgSlug = () => {
    const raw = (appState as { authGetCurrentOrgJson?: () => string | null })
      .authGetCurrentOrgJson?.();
    if (!raw) return "";
    try { return (JSON.parse(raw) as { slug?: string }).slug ?? ""; }
    catch { return ""; }
  };

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
  // The Rust runner cache parses `{runners: [...]}` shape — Connect's
  // ListRunnersResponse uses `items`. Also the renderer reads
  // `r.available_agents` (snake_case) but Connect emits camelCase, so
  // recursively rename keys before returning.
  ipcMain.handle("runnerFetchRunners", async () => {
    const raw = await callConnectJson(
      "proto.runner_api.v1.RunnerService",
      "ListRunners",
      { orgSlug: orgSlug() },
    );
    const parsed = JSON.parse(raw) as { items?: unknown[] };
    return JSON.stringify({ runners: (parsed.items ?? []).map(snakeCaseDeep) });
  });

  // R6 dropped the Rust channel_join_channel napi (replaced by direct
  // ChannelService.JoinChannelPod). Renderer paths use wasm Connect; main
  // exposes these aliases for desktop e2e specs that still invoke through
  // IPC by name. The cache update is necessary because main's AppState
  // holds a separate Rust state from the renderer wasm — without the
  // app_channel_replace_pods fan-out into runtime.state, appChannelPodsJson
  // returns empty.
  //
  // Connect-JSON serializes int64 as a JSON string to preserve precision —
  // IPC callers parse the channel response and forward `id` to us as a
  // string. Coerce to a JS number before handing it to napi (i64 binding).
  const toIdNumber = (v: number | string): number =>
    typeof v === "number" ? v : Number(v);

  // Recursively coerce numeric-looking string values to numbers for keys
  // known to be int64 fields. The Connect-JSON wire serializes int64 as a
  // string, but desktop e2e callers (and the legacy Rust napi shape) expect
  // numbers — `expect(joined.agent_count).toBe(1)` would fail on "1".
  const INT64_KEYS = new Set([
    "id", "organization_id", "repository_id", "ticket_id",
    "created_by_user_id", "member_count", "agent_count",
    "message_count", "sender_user_id", "channel_id", "reply_to",
  ]);
  const coerceInt64 = (value: unknown): unknown => {
    if (Array.isArray(value)) return value.map(coerceInt64);
    if (value !== null && typeof value === "object") {
      const out: Record<string, unknown> = {};
      for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
        if (INT64_KEYS.has(k) && typeof v === "string" && /^-?\d+$/.test(v)) {
          out[k] = Number(v);
        } else {
          out[k] = coerceInt64(v);
        }
      }
      return out;
    }
    return value;
  };

  // Connect-JSON omits proto3 default values (0, "", false). The legacy
  // Rust napi shape always emitted them explicitly. Channel-shaped responses
  // need agent_count / member_count present as numbers so e2e specs that
  // do `expect(channel.agent_count).toBe(0)` get a number, not undefined.
  const ensureChannelDefaults = (value: Record<string, unknown>): Record<string, unknown> => {
    const defaults: Record<string, unknown> = {
      agent_count: 0,
      member_count: 0,
      is_member: false,
      is_archived: false,
    };
    return { ...defaults, ...value };
  };

  ipcMain.handle("channelCreateChannel", async (_e, requestJson: string) => {
    const req = JSON.parse(requestJson) as Record<string, unknown>;
    const raw = await callConnectJson(
      "proto.channel.v1.ChannelService",
      "CreateChannel",
      { orgSlug: orgSlug(), ...req },
    );
    // Coerce int64-string fields and emit proto3 defaults (Connect-JSON
    // omits zero values; legacy Rust napi shape always included them).
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return JSON.stringify(ensureChannelDefaults(coerceInt64(parsed) as Record<string, unknown>));
  });

  const refreshChannelPodsCache = async (channelId: number): Promise<void> => {
    const raw = await callConnectJson(
      "proto.channel.v1.ChannelService",
      "ListChannelPods",
      { orgSlug: orgSlug(), id: channelId },
    );
    const parsed = JSON.parse(raw) as { items?: Array<Record<string, unknown>> };
    // Wrap the 5 ChannelPod summary fields into proto.pod.v1.Pod (other
    // fields default to proto3 zeros) and dispatch via the cross-domain
    // SSOT mutator on the NAPI bridge.
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
    // napi-rs Vec<u8> binding expects `Array<number>` over the JS boundary;
    // Uint8Array works in some paths but the channelJoinChannel ipcMain
    // handler (called from renderer through serialised IPC) sees the value
    // as an opaque object without a `length` accessor. Materialise as a
    // plain array to match the rest of the proto-bytes NAPI surface.
    await (appState as { appChannelReplacePods: (b: number[]) => Promise<void> })
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
    // ElectronChannelService.get_channel_pods (electron-adapter) expects the
    // legacy envelope `{pods: [...], total: N}` so it can mirror the array
    // into its renderer-side cache. The Rust cache JSON is a bare array —
    // wrap it back into the legacy shape.
    const cacheJson = await (appState as { appChannelPodsJson: (id: number) => Promise<string> })
      .appChannelPodsJson(channelId);
    const pods = JSON.parse(cacheJson) as unknown[];
    return JSON.stringify({ pods, total: pods.length });
  });

  // ElectronOrgService.create_personal() invokes this — main → backend
  // proto.org.v1.OrgService/CreatePersonalOrg. Server derives slug from
  // users.username; we return the raw Organization JSON unchanged.
  ipcMain.handle("orgCreatePersonal", () =>
    callConnectJson("proto.org.v1.OrgService", "CreatePersonalOrg", {}),
  );

  // ElectronAgentService.list_agents() invokes this. Renderer hooks parse
  // the response as `{builtin_agents, custom_agents}` (snake_case); the
  // Connect endpoint emits proto camelCase, so we remap and also rename
  // nested keys recursively.
  ipcMain.handle("agentListAgents", async () => {
    const raw = await callConnectJson(
      "proto.agent.v1.AgentService",
      "ListAgents",
      { orgSlug: orgSlug() },
    );
    const parsed = JSON.parse(raw) as {
      builtinAgents?: unknown[];
      customAgents?: unknown[];
      builtin_agents?: unknown[];
      custom_agents?: unknown[];
    };
    return JSON.stringify({
      builtin_agents: (parsed.builtin_agents ?? parsed.builtinAgents ?? []).map(snakeCaseDeep),
      custom_agents: (parsed.custom_agents ?? parsed.customAgents ?? []).map(snakeCaseDeep),
    });
  });

  // ElectronRepositoryService.list() invokes this. The renderer cache
  // expects `{repositories: [...]}` (legacy REST shape) while the
  // Connect endpoint returns `{items, total, ...}` — remap. Also
  // convert proto camelCase fields to snake_case for legacy callers.
  ipcMain.handle("repositoryList", async () => {
    const raw = await callConnectJson(
      "proto.repository.v1.RepositoryService",
      "ListRepositories",
      { orgSlug: orgSlug() },
    );
    const parsed = JSON.parse(raw) as { items?: unknown[] };
    return JSON.stringify({ repositories: (parsed.items ?? []).map(snakeCaseDeep) });
  });

  // ElectronPodService.create_pod() invokes this. The renderer hands us
  // a JSON request payload using snake_case (legacy REST shape); we
  // upgrade it to the proto camelCase shape Connect expects and remap
  // the response back to snake_case for the renderer's pod cache.
  ipcMain.handle("podCreatePod", async (_e, requestJson: string) => {
    const req = JSON.parse(requestJson) as Record<string, unknown>;
    // Pass through every field unchanged; the Connect JSON layer accepts
    // both proto camelCase AND grpc-gateway snake_case via @bufbuild/protobuf.
    const raw = await callConnectJson(
      "proto.pod.v1.PodService",
      "CreatePod",
      { orgSlug: orgSlug(), ...req },
    );
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return JSON.stringify(snakeCaseDeep(parsed));
  });

  // Counterpart to podCreatePod. The realtime bridge e2e specs depend on
  // it for cleanup. Renderer hands us a pod_key string; we route to
  // Connect proto.pod.v1.PodService/TerminatePod.
  ipcMain.handle("podTerminatePod", async (_e, podKey: string) => {
    await callConnectJson(
      "proto.pod.v1.PodService",
      "TerminatePod",
      { orgSlug: orgSlug(), podKey },
    );
  });

  // Generic binary Connect-RPC proxy. Web's wasm-side services expose
  // `<method>Connect(Uint8Array) -> Uint8Array`; ElectronXxxService
  // adapters that don't yet have hand-written `_connect` IPC handlers
  // route through this instead. The protobuf encode/decode stays on the
  // renderer; main only ferries bytes (as number[]) over IPC and forwards
  // to the backend Connect endpoint.
  ipcMain.handle("connectCall", async (
    _e, service: string, method: string, bodyArr: number[],
  ) => {
    const url = `${currentApiUrl}/${service}/${method}`;
    const token = (appState as { authGetToken?: () => string | null }).authGetToken?.();
    const headers: Record<string, string> = {
      "Content-Type": "application/proto",
      "Connect-Protocol-Version": "1",
    };
    if (token) headers.Authorization = `Bearer ${token}`;
    const res = await connectFetch(url, {
      method: "POST",
      headers,
      body: Uint8Array.from(bodyArr),
    });
    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(`${res.status} ${res.statusText} ${url} ${text}`.trim());
    }
    const bytes = new Uint8Array(await res.arrayBuffer());
    return Array.from(bytes);
  });
}

function buildMenu() {
  const isMac = process.platform === "darwin";
  const template: Electron.MenuItemConstructorOptions[] = [
    ...(isMac ? ([{ role: "appMenu" }] as Electron.MenuItemConstructorOptions[]) : []),
    { role: "fileMenu" },
    { role: "editMenu" },
    { role: "viewMenu" },
    { role: "windowMenu" },
    {
      role: "help",
      submenu: [
        {
          label: "Open Logs",
          click: async () => {
            await shell.openPath(logsDir);
          },
        },
      ],
    },
  ];
  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}

app.whenReady().then(() => {
  try {
    initLogger(logsDir, process.env.AGENTSMESH_LOG_LEVEL ?? "info");
    logEvent("info", "electron-main", `Starting, API: ${currentApiUrl}`);
  } catch (e) {
    console.warn("[electron] initLogger failed:", e);
  }
  console.log(`[electron] Starting, API: ${currentApiUrl}, storage: ${storageDir}, logs: ${logsDir}`);
  if (isHeadlessTest && process.platform === "darwin") {
    app.dock?.hide();
  }
  if (pendingStartupErrorMsg) {
    dialog.showErrorBox(
      "Invalid server configuration",
      `${pendingStartupErrorMsg}\n\nFalling back to AgentsMesh Global. Open Server Settings to pick a different server.`,
    );
    pendingStartupErrorMsg = null;
  }
  appState = new AppState(currentApiUrl, storageDir);
  if (isHeadlessTest) {
    stubs = createLocalRunnerStubs();
  }
  registerStaticHandlers();
  bindAppStateHandlers();
  // Wire the realtime EventBus bridge BEFORE createWindow() so the renderer's
  // EventSubscriptionManager finds `realtime:connect` etc. registered as soon
  // as preload runs. The stream itself doesn't start until renderer calls
  // electronAPI.invoke("realtime:connect").
  void setupRealtimeBridge(appState, getMainWindow).then((bridge) => {
    realtimeBridge = bridge;
  });
  relayBridge = setupRelayBridge(appState, getMainWindow);
  buildMenu();
  installOpenUrlHandler(getMainWindow);
  createWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

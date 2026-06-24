import { app, dialog, ipcMain, shell } from "electron";
import path from "path";
import { AppState, initLogger, logEvent } from "@agentsmesh/node-bridge";
import { createLocalRunnerStubs, type LocalRunnerStubMap } from "./local_runner_stubs";
import { acquireSingleInstance } from "./single_instance";
import { bindAppStateHandlers } from "./app_state_handlers";
import { resolveColdStartApiUrl } from "./cold_start";
import { setupRealtimeBridge, type RealtimeBridge } from "./realtime";
import { setupRelayBridge, type RelayBridge } from "./relay";
import { setupAutoUpdater } from "./auto_updater";
import { WindowRegistry } from "./window_registry";
import { createAppWindow } from "./create_window";
import { registerWindowIpc } from "./window_ipc";
import { buildMenu } from "./menu";
import { registerConnectCall } from "./connect_call";
import { registerLegacyApiAliases } from "./legacy_api_aliases";
import {
  registerProtocol,
  attachSecondInstanceUrlHandler,
  installOpenUrlHandler,
  captureColdLaunchUrl,
  flushPendingUrl,
} from "./oauth_deep_link";
import * as serverConfig from "./server_config";

// pendingStartupErrorMsg is module-load stashed; flushed in whenReady because
// dialog.showErrorBox is unreliable before app.ready fires.
let currentCfg = serverConfig.load();
const coldStart = resolveColdStartApiUrl(currentCfg);
let currentApiUrl = coldStart.apiUrl;
let pendingStartupErrorMsg: string | null = coldStart.error;

const storageDir = path.join(app.getPath("userData"), "agentsmesh");
const logsDir = app.getPath("logs");

const isHeadlessTest = process.env.NODE_ENV === "test";

let appState: AppState;
let stubs: LocalRunnerStubMap | null = null;
let realtimeBridge: RealtimeBridge | null = null;
let relayBridge: RelayBridge | null = null;
const appStateHandlers = new Set<string>();
const registry = new WindowRegistry();

const getPrimaryWindow = () => registry.getPrimary();

// oauth deep-links need a real window; getPrimary() is null when only popouts
// survive, so spawn one on demand.
function ensurePrimaryWindow() {
  return registry.getPrimary() ?? (createPrimaryWindow(), registry.getPrimary());
}

// A closing window never runs the renderer's unsubscribe — release its relay/
// realtime ref-counts here so shared links tear down once the last window leaves.
function onWindowClosed(wcId: number): void {
  relayBridge?.releaseWebContents(wcId);
  realtimeBridge?.releaseWebContents(wcId);
}

if (acquireSingleInstance()) {
  registerProtocol();
  attachSecondInstanceUrlHandler(ensurePrimaryWindow);
  captureColdLaunchUrl();
}

function createPrimaryWindow(): void {
  createAppWindow(registry, { kind: "primary", onClosed: onWindowClosed });
  flushPendingUrl(getPrimaryWindow);
}

// Known leak: LocalRunnerManager / RelayManager have no shutdown hook, so their tokio
// tasks may outlive the rebind. Rare in practice (server switches are uncommon).
async function rebindAppState(newApiUrl: string) {
  logEvent("info", "electron-main", `rebind AppState → ${newApiUrl}`);
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
  bindAppStateHandlers({ appState, stubs, tracked: appStateHandlers });
  registerConnectCall({ getAppState: () => appState, getApiUrl: () => currentApiUrl }, appStateHandlers);
  registerLegacyApiAliases({ getAppState: () => appState, getApiUrl: () => currentApiUrl }, appStateHandlers);
  // Await so realtime:connect/getState are registered before serverConfig:set
  // reloads windows — else a reloaded renderer's connect rejects and stays dead.
  realtimeBridge = await setupRealtimeBridge(appState, registry);
  relayBridge = setupRelayBridge(appState, registry);
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
  // (before any renderer code runs — no UI thread to block). Reloading a window re-runs
  // preload to propagate serverConfig:set updates.
  // Invariant: registerStaticHandlers() MUST run before createPrimaryWindow() (enforced by ordering below).
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
    // Capture before rebindAppState mutates currentApiUrl.
    const urlChanged = newUrl !== currentApiUrl;
    if (urlChanged) {
      await rebindAppState(newUrl);
    }
    // Re-snapshots preload even on a same-URL edit (dialog cfg fields can change).
    registry.applyConfigChange(urlChanged);
  });
}

app.whenReady().then(() => {
  try {
    // Default debug; the logging crate scopes third-party crates to info
    // (clients/core/crates/logging/src/init.rs) so business logs stay readable.
    initLogger(logsDir, process.env.AGENTSMESH_LOG_LEVEL ?? "debug");
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
  logEvent("info", "electron-main", "AppState ready");
  if (isHeadlessTest) {
    stubs = createLocalRunnerStubs();
  }
  registerStaticHandlers();
  bindAppStateHandlers({ appState, stubs, tracked: appStateHandlers });
  // MUST follow bindAppStateHandlers (which clears appStateHandlers): connectCall +
  // aliases aren't in the allowlist, so registering earlier gets them wiped.
  registerConnectCall({ getAppState: () => appState, getApiUrl: () => currentApiUrl }, appStateHandlers);
  registerLegacyApiAliases({ getAppState: () => appState, getApiUrl: () => currentApiUrl }, appStateHandlers);
  // Wire the realtime EventBus bridge BEFORE createPrimaryWindow() so the renderer's
  // EventSubscriptionManager finds `realtime:connect` etc. registered as soon
  // as preload runs. The stream itself doesn't start until renderer calls
  // electronAPI.invoke("realtime:connect").
  void setupRealtimeBridge(appState, registry).then((bridge) => {
    realtimeBridge = bridge;
  });
  relayBridge = setupRelayBridge(appState, registry);
  setupAutoUpdater(getPrimaryWindow);
  registerWindowIpc(registry, onWindowClosed);
  registry.setPrimaryChangeListener(() => registry.broadcast("window:primary-changed", null));
  buildMenu({
    logsDir,
    onNewWindow: () => void createAppWindow(registry, { kind: "secondary", onClosed: onWindowClosed }),
  });
  installOpenUrlHandler(ensurePrimaryWindow);
  createPrimaryWindow();

  app.on("activate", () => {
    if (!registry.hasWindows()) createPrimaryWindow();
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

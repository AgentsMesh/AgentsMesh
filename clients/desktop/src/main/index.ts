import { app, BrowserWindow, dialog, ipcMain, Menu, shell } from "electron";
import path from "path";
import { AppState, initLogger, logEvent } from "@agentsmesh/node-bridge";
import { createLocalRunnerStubs, type LocalRunnerStubMap } from "./local_runner_stubs";
import { acquireSingleInstance } from "./single_instance";
import {
  registerProtocol,
  attachSecondInstanceUrlHandler,
  installOpenUrlHandler,
  captureColdLaunchUrl,
  flushPendingUrl,
} from "./oauth_deep_link";
import * as serverConfig from "./server_config";
import { isValidServerUrl } from "../shared/server-config-types";

// SSOT: server_config owns the persisted choice (server.json), main owns
// the cold-start env override policy. Keeping the env-override resolution
// out of server_config means that module stays a pure function library
// (easier to test, harder to misuse from `serverConfig:set` handler).
//
// Cold-start resolution order:
//   1. AGENTSMESH_API_URL env (dev / e2e / ad-hoc) — does NOT pollute the
//      dialog SSOT, just redirects requests for this process lifetime.
//   2. activeUrl(currentCfg) — server.json or DEFAULT.
//
// Once the user saves through the dialog, `serverConfig:set` rebinds to
// `activeUrl(next)` directly, shadowing the env override.

// Stashed at module load (before app.ready); flushed in whenReady so users
// actually see the dialog. Calling dialog.showErrorBox at module load is
// unreliable — most Electron dialog APIs assume app.ready has fired.
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
// Electron resolves "logs" to ~/Library/Logs/<AppName>/ on macOS,
// %APPDATA%\<AppName>\logs\ on Windows, ~/.config/<AppName>/logs/ on Linux.
// The Rust logger creates the dir if missing and rolls files daily inside it.
const logsDir = app.getPath("logs");

// Headless flag for e2e: keeps the window invisible + drops the macOS
// dock icon so the test process doesn't steal focus from the user's IDE.
const isHeadlessTest = process.env.NODE_ENV === "test";

let appState: AppState;
let stubs: LocalRunnerStubMap | null = null;
let mainWindow: BrowserWindow | null = null;
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

// Re-bind every IPC channel that fronts an `AppState` method. Called once
// at boot and again when the user switches server (which constructs a new
// AppState bound to the new base_url). Must `removeHandler` first because
// `ipcMain.handle` throws on duplicate registration.
function bindAppStateHandlers() {
  for (const ch of appStateHandlers) {
    ipcMain.removeHandler(ch);
  }
  appStateHandlers.clear();

  const proto = Object.getPrototypeOf(appState);
  const methodNames = Object.getOwnPropertyNames(proto).filter(
    (k) => k !== "constructor" && typeof (appState as any)[k] === "function",
  );
  for (const m of methodNames) {
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
  }
  console.log(`[electron] Registered ${methodNames.length} IPC handlers`);
}

// Replace the running AppState with one bound to a new base_url. Used when
// the user picks a different server. Old service instances inside the prior
// AppState are dropped naturally once their last `Arc` ref expires
// (in-flight requests fade out without aborting).
//
// Known limitation: Rust services without explicit shutdown logic
// (LocalRunnerManager, RelayManager) may keep tokio tasks alive past
// the rebind, leaking until process exit. Fine in practice because
// server switches are rare; would need graceful shutdown plumbing in
// Rust core to fix properly.
function rebindAppState(newApiUrl: string) {
  console.log(`[electron] Rebinding AppState: ${currentApiUrl} → ${newApiUrl}`);
  appState = new AppState(newApiUrl, storageDir);
  // Stubs are scoped to the AppState that owns them — re-create alongside.
  // (isHeadlessTest itself is fixed at startup; the conditional here is
  // not a runtime check, just a "if we needed stubs before, we need them now".)
  if (isHeadlessTest) {
    stubs = createLocalRunnerStubs();
  }
  bindAppStateHandlers();
  currentApiUrl = newApiUrl;
}

function registerStaticHandlers() {
  ipcMain.handle("shellOpen", async (_e, url: string) => {
    if (/^https?:\/\//i.test(url) || url.startsWith("mailto:") || url.startsWith("agentsmesh://")) {
      await shell.openExternal(url);
    }
  });

  // Renderer-side log forwarding. Renderer calls `electronAPI.log(...)` →
  // we hand it to the Rust subscriber so renderer + main + Rust all land
  // in one rolling file in timestamp order.
  ipcMain.handle("core:log", (_e, level: string, target: string, msg: string) => {
    logEvent(level, target, msg);
  });

  // Help → Open Logs reveals the rolling log directory in Finder/Explorer.
  ipcMain.handle("logs:openFolder", async () => {
    await shell.openPath(logsDir);
  });

  // server-config IPC. The sync variants are deliberate: preload reads
  // them at boot to populate `window.electronAPI.apiUrl` / serverConfig
  // .snapshot synchronously, so renderer's getApiBaseUrl() stays
  // synchronous (no async ceremony in OAuth/WS hot paths).
  //
  // The Electron sync-IPC anti-pattern warning is about runtime calls
  // that block the renderer's UI thread; here the call happens during
  // preload BEFORE any renderer code runs, so there's no UI to block.
  // On `mainWindow.reload()`, preload re-executes and re-reads — that's
  // also how we propagate `serverConfig:set` updates without a separate
  // mutable IPC channel.
  //
  // Invariant: registerStaticHandlers() MUST run before createWindow()
  // so the sync handlers exist when preload sends. Enforced by ordering
  // in app.whenReady() below.
  ipcMain.on("serverConfig:getActiveUrlSync", (e) => {
    e.returnValue = currentApiUrl;
  });
  ipcMain.on("serverConfig:getSync", (e) => {
    e.returnValue = currentCfg;
  });
  ipcMain.handle("serverConfig:get", () => serverConfig.load());
  ipcMain.handle("serverConfig:set", async (_e, raw: serverConfig.ServerConfig) => {
    // `raw` is whatever the renderer (or a misbehaving stub) sent over
    // IPC — type-erased at the boundary. Validate via activeUrl first
    // (throws on malformed custom URL → propagates back to dialog), then
    // let `save` normalise + persist. Use `save`'s return value as the new
    // currentCfg so memory and disk stay byte-identical; assigning `raw`
    // directly was the third-round bug.
    serverConfig.activeUrl(raw); // validate, throw on invalid
    const next = serverConfig.save(raw);
    const newUrl = serverConfig.activeUrl(next);
    currentCfg = next;
    if (newUrl !== currentApiUrl) {
      rebindAppState(newUrl);
    }
    // Always reload renderer after a save — preload's sync snapshot of
    // serverConfig is captured at boot, so even when the resolved URL
    // doesn't change (e.g. env override masks both old and new), the
    // *cfg* fields the dialog reads can have changed (kind/custom* etc).
    // Reloading is the cheap way to re-snapshot without a separate
    // mutable IPC channel.
    mainWindow?.reload();
  });
}

function buildMenu() {
  // Default-role menus give us the standard macOS/Win/Linux items (Quit,
  // Cut/Copy/Paste, View, Window) without re-implementing them. The Help
  // submenu is the only item we customise — it exposes the rolling log
  // directory so users can ship logs back when reporting bugs.
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
  // Install the global tracing subscriber FIRST — every Rust call after
  // this (including AppState construction) emits through it. Failure here
  // is logged but not fatal; renderer still works, just without persistent
  // logs. Idempotent on the Rust side.
  try {
    initLogger(logsDir, process.env.AGENTSMESH_LOG_LEVEL ?? "info");
    // First post-init record proves the file sink is live. The Rust
    // non-blocking writer flushes per-line, so this lands on disk
    // immediately — useful as a "logger is wired up" smoke marker for
    // bug reports.
    logEvent("info", "electron-main", `Starting, API: ${currentApiUrl}`);
  } catch (e) {
    console.warn("[electron] initLogger failed:", e);
  }
  console.log(`[electron] Starting, API: ${currentApiUrl}, storage: ${storageDir}, logs: ${logsDir}`);
  if (isHeadlessTest && process.platform === "darwin") {
    app.dock?.hide();
  }
  // Surface any startup error stashed during module load (server.json
  // pointed at a malformed custom URL etc.). Dialog APIs need app.ready,
  // which is why we deferred this from `resolveColdStartApiUrl`.
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

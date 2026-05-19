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

// Known leak: LocalRunnerManager / RelayManager have no shutdown hook, so their tokio
// tasks may outlive the rebind. Rare in practice (server switches are uncommon).
function rebindAppState(newApiUrl: string) {
  console.log(`[electron] Rebinding AppState: ${currentApiUrl} → ${newApiUrl}`);
  appState = new AppState(newApiUrl, storageDir);
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

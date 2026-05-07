import { app, BrowserWindow, ipcMain, shell } from "electron";
import path from "path";
import { AppState } from "@agentsmesh/node-bridge";
import { createLocalRunnerStubs, type LocalRunnerStubMap } from "./local_runner_stubs";
import { acquireSingleInstance } from "./single_instance";
import {
  registerProtocol,
  attachSecondInstanceUrlHandler,
  installOpenUrlHandler,
  captureColdLaunchUrl,
  flushPendingUrl,
} from "./oauth_deep_link";

const apiUrl = process.env.AGENTSMESH_API_URL ?? "http://localhost:25350";
const storageDir = path.join(app.getPath("userData"), "agentsmesh");

// Headless flag for e2e: keeps the window invisible + drops the macOS
// dock icon so the test process doesn't steal focus from the user's IDE.
// `global.setup.ts` sets `NODE_ENV=test` on every Electron launch.
const isHeadlessTest = process.env.NODE_ENV === "test";

let appState: AppState;
let stubs: LocalRunnerStubMap | null = null;
let mainWindow: BrowserWindow | null = null;

const getMainWindow = () => mainWindow;

// MUST run before whenReady — single-instance lock is acquired sync,
// and "second-instance" can fire before whenReady on the duplicate
// launch path.
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
    // Required when `show: false` — keeps the renderer painting so
    // Playwright assertions (`expect(locator).toBeVisible`) still pass.
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
    shell.openExternal(url);
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

function registerIpcHandlers() {
  ipcMain.handle("shellOpen", async (_e, url: string) => {
    await shell.openExternal(url);
  });

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
  }
  console.log(`[electron] Registered ${methodNames.length} IPC handlers`);
}

app.whenReady().then(() => {
  console.log(`[electron] Starting, API: ${apiUrl}, storage: ${storageDir}`);
  // Drop the dock icon in headless e2e on macOS so the runner doesn't
  // pull focus away from the user's editor mid-run.
  if (isHeadlessTest && process.platform === "darwin") {
    app.dock?.hide();
  }
  appState = new AppState(apiUrl, storageDir);
  if (isHeadlessTest) {
    stubs = createLocalRunnerStubs();
  }
  registerIpcHandlers();
  installOpenUrlHandler(getMainWindow);
  createWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

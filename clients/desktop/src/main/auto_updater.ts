import { app, ipcMain, powerMonitor, type BrowserWindow } from "electron";
// electron-updater is CJS-only — a named import resolves to undefined under
// electron-vite interop. Default-import; destructure inside the function since
// the `autoUpdater` getter instantiates the platform updater (touches app).
import electronUpdater, { type UpdateInfo, type ProgressInfo } from "electron-updater";
import { logEvent } from "@agentsmesh/node-bridge";
import { reduceUpdater, INITIAL_SNAPSHOT, type UpdaterEvent, type UpdaterSnapshot } from "../shared/updater-reducer";

const CHECK_INTERVAL_MS = 4 * 60 * 60 * 1000 + 7 * 60 * 1000; // ~4h, off the hour
const FOCUS_THROTTLE_MS = 30 * 60 * 1000;

export function setupAutoUpdater(getMainWindow: () => BrowserWindow | null): void {
  let snapshot: UpdaterSnapshot = INITIAL_SNAPSHOT;

  ipcMain.handle("updater:getVersion", () => app.getVersion());
  ipcMain.handle("updater:getState", () => snapshot);

  // Unpackaged / e2e have no app-update.yml. Register a no-op check surface and
  // return before touching electron-updater — instantiating it (Squirrel.Mac
  // native init on macOS) + its listeners + the interval all run before
  // createWindow and are pure startup cost when checks can't run anyway.
  if (!app.isPackaged || process.env.NODE_ENV === "test") {
    ipcMain.handle("updater:check", () => {});
    ipcMain.handle("updater:quitAndInstall", () => {});
    return;
  }

  const { autoUpdater } = electronUpdater;

  const send = (ev: UpdaterEvent) => {
    const next = reduceUpdater(snapshot, ev);
    if (next === snapshot) return; // unchanged — skip the push + a no-op renderer re-render
    snapshot = next;
    const win = getMainWindow();
    if (!win || win.isDestroyed()) return;
    win.webContents.send("updater:snapshot", snapshot);
  };

  autoUpdater.autoDownload = true;
  autoUpdater.autoInstallOnAppQuit = true;
  autoUpdater.allowPrerelease = false;

  autoUpdater.on("checking-for-update", () => send({ type: "checking" }));
  autoUpdater.on("update-available", (info: UpdateInfo) => {
    logEvent("info", "updater", `available ${info.version}`);
    send({ type: "available", version: info.version });
  });
  autoUpdater.on("update-not-available", () => send({ type: "not-available" }));
  autoUpdater.on("download-progress", (p: ProgressInfo) =>
    send({ type: "progress", percent: Math.round(p.percent) }),
  );
  autoUpdater.on("update-downloaded", (info: UpdateInfo) => {
    logEvent("info", "updater", `downloaded ${info.version}`);
    send({ type: "downloaded", version: info.version });
  });
  autoUpdater.on("error", (err: Error) => {
    logEvent("warn", "updater", `error: ${err.message}`);
    send({ type: "error", message: err.message });
  });

  let lastCheckAt = 0;
  const check = (reason: string): void => {
    lastCheckAt = Date.now();
    logEvent("info", "updater", `checking (${reason})`);
    void autoUpdater.checkForUpdates().catch((err: Error) => {
      logEvent("warn", "updater", `checkForUpdates failed: ${err.message}`);
      send({ type: "error", message: err.message });
    });
  };

  ipcMain.handle("updater:check", () => check("manual"));
  ipcMain.handle("updater:quitAndInstall", () => {
    logEvent("info", "updater", "quitAndInstall");
    autoUpdater.quitAndInstall();
  });

  setInterval(() => check("interval"), CHECK_INTERVAL_MS);
  const onReArm = () => {
    if (Date.now() - lastCheckAt >= FOCUS_THROTTLE_MS) check("re-arm");
  };
  powerMonitor.on("resume", onReArm);
  app.on("browser-window-focus", onReArm);

  check("boot");
}

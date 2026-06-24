import { BrowserWindow, shell } from "electron";
import path from "path";
import { type WindowRegistry, type WindowKind } from "./window_registry";

interface CreateOpts {
  kind: WindowKind;
  route?: string;
  width?: number;
  height?: number;
  onClosed?: (wcId: number) => void;
}

const isHeadlessTest = process.env.NODE_ENV === "test";

// Immersive titlebar: mac traffic lights float into the left rail's top drag
// strip (renderer reserves it via --titlebar-drag-height); Windows uses an
// overlay; Linux keeps the native frame.
function titlebarOpts(): Electron.BrowserWindowConstructorOptions {
  if (process.platform === "darwin")
    return { titleBarStyle: "hiddenInset", trafficLightPosition: { x: 14, y: 14 } };
  if (process.platform === "win32")
    return {
      titleBarStyle: "hidden",
      titleBarOverlay: { color: "#F6F8FA", symbolColor: "#656D76", height: 40 },
    };
  return {};
}

export function createAppWindow(registry: WindowRegistry, opts: CreateOpts): BrowserWindow {
  const win = new BrowserWindow({
    width: opts.width ?? 1280,
    height: opts.height ?? 800,
    minWidth: 900,
    minHeight: 600,
    title: "AgentsMesh",
    show: !isHeadlessTest,
    paintWhenInitiallyHidden: true,
    skipTaskbar: isHeadlessTest,
    ...titlebarOpts(),
    webPreferences: {
      preload: path.join(__dirname, "../preload/index.js"),
      contextIsolation: true,
      sandbox: false,
      nodeIntegration: false,
      // Read synchronously by preload to scope per-window UI-layout persistence.
      additionalArguments: [`--agentsmesh-window-kind=${opts.kind}`],
    },
  });

  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:\/\//i.test(url) || url.startsWith("mailto:") || url.startsWith("agentsmesh://")) {
      shell.openExternal(url);
    }
    return { action: "deny" };
  });

  const devUrl = process.env.ELECTRON_RENDERER_URL;
  if (devUrl) {
    win.loadURL(opts.route ? `${devUrl}#${opts.route}` : devUrl);
    // One detached devtools is enough; extra windows would stack noisy panels.
    if (opts.kind === "primary") win.webContents.openDevTools({ mode: "detach" });
  } else {
    const indexHtml = path.join(__dirname, "../renderer/index.html");
    win.loadFile(indexHtml, opts.route ? { hash: opts.route } : undefined);
  }

  const id = win.webContents.id;
  registry.register(win, opts.kind);
  win.on("closed", () => {
    registry.unregister(id);
    opts.onClosed?.(id);
  });
  return win;
}

import { ipcMain } from "electron";
import { createAppWindow } from "./create_window";
import { type WindowRegistry, type WindowKind } from "./window_registry";

// Renderer-driven popout windows: each kind maps to a WindowKind + a chromeless route.
const POPOUTS: Record<string, { kind: WindowKind; route: (p: Record<string, string>) => string | null }> = {
  channel: { kind: "channel", route: (p) => (p.id ? `/popout/channel/${p.id}` : null) },
};

export function registerWindowIpc(registry: WindowRegistry, onClosed: (wcId: number) => void): void {
  ipcMain.handle("window:openTerminal", (_e, podKey: string) => {
    const win = createAppWindow(registry, {
      kind: "terminal",
      route: `/popout/terminal/${podKey}`,
      width: 900,
      height: 640,
      onClosed,
    });
    return win.webContents.id;
  });

  ipcMain.handle("window:open", (_e, kind: string, params: Record<string, string> = {}) => {
    const spec = POPOUTS[kind];
    if (!spec) throw new Error(`window:open: unknown kind "${kind}"`);
    const route = spec.route(params);
    if (!route) throw new Error(`window:open: "${kind}" missing required params`);
    const win = createAppWindow(registry, { kind: spec.kind, route, onClosed });
    return win.webContents.id;
  });

  ipcMain.handle("window:isPrimary", (e) => registry.isPrimary(e.sender.id));
}

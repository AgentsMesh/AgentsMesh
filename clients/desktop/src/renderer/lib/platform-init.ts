import { createElectronServiceProvider } from "@agentsmesh/electron-adapter";
import {
  registerServiceProvider, markServiceReady, setPlatformInit,
} from "@agentsmesh/service-runtime";
import { installConsoleCapture } from "@/lib/console-capture";
import { installRealtimeMirror } from "./realtime-mirror";

// Desktop aliases @/lib/wasm-core to a shim (electron.vite.config.ts), so web's
// install site (wasm-core.ts) never runs here. Wire it explicitly — main.tsx
// imports this module before React renders — so renderer console.warn/error
// fans out via electronAPI.log → main core:log → Rust rolling file.
installConsoleCapture();
markDesktopPlatform();

setPlatformInit(async () => {
  const provider = createElectronServiceProvider();
  registerServiceProvider(provider);
  markServiceReady();
  installRealtimeMirror();
});

// Tag <html> so shared web components can gate desktop-only immersive-titlebar
// spacing via CSS — web never runs this module, so --titlebar-drag-height stays 0.
function markDesktopPlatform() {
  const root = document.documentElement;
  root.classList.add("platform-desktop");
  const platform = window.electronAPI?.platform;
  if (platform === "darwin") root.classList.add("platform-mac");
  else if (platform === "win32") root.classList.add("platform-win");
  else root.classList.add("platform-linux");
}

export function isElectron(): boolean {
  return typeof window !== "undefined" && !!(window as any).electronAPI;
}

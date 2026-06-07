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

setPlatformInit(async () => {
  const provider = createElectronServiceProvider();
  registerServiceProvider(provider);
  markServiceReady();
  installRealtimeMirror();
});

export function isElectron(): boolean {
  return typeof window !== "undefined" && !!(window as any).electronAPI;
}

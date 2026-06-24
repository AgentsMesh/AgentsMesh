import { ipcMain } from "electron";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import { type LocalRunnerStubMap } from "./local_runner_stubs";
import { IPC_ALLOWLIST, IPC_ALLOWLIST_SET } from "./ipc-allowlist.generated";

interface BindDeps {
  appState: AppState;
  stubs: LocalRunnerStubMap | null;
  // Owned by the caller and reused across rebinds so removeHandler matches what
  // was registered.
  tracked: Set<string>;
}

// Called at boot and after server switch (new AppState). MUST removeHandler
// first because ipcMain.handle throws on duplicate registration.
//
// Allowlist-driven (vs reflect-everything): the IPC channel set comes from
// `ipc-allowlist.generated.ts` — auto-generated from the same NAPI binary symbol
// enumeration that drives e2e contract specs, so the two stay in lock-step. Any
// NAPI method not in the allowlist is unreachable from the renderer even if it
// exists on AppState.prototype. Drift is logged (warn, not crash) so a Bazel
// rebuild lag doesn't block dev workflows.
export function bindAppStateHandlers({ appState, stubs, tracked }: BindDeps): void {
  for (const ch of [...tracked]) ipcMain.removeHandler(ch);
  tracked.clear();

  const proto = Object.getPrototypeOf(appState);
  const protoMethods = new Set(
    Object.getOwnPropertyNames(proto).filter(
      (k) => k !== "constructor" && typeof (appState as any)[k] === "function",
    ),
  );

  const missingFromBinary = IPC_ALLOWLIST.filter((n) => !protoMethods.has(n));
  const missingFromAllowlist = [...protoMethods].filter((n) => !IPC_ALLOWLIST_SET.has(n));
  if (missingFromBinary.length > 0) {
    logEvent("warn", "ipc", `allowlist drift: ${missingFromBinary.length} listed but missing from binary`);
    console.warn(`[electron] IPC allowlist drift: ${missingFromBinary.length} methods listed but not in AppState.prototype — regenerate with \`pnpm --filter desktop e2e:gen\``);
  }
  if (missingFromAllowlist.length > 0) {
    logEvent("warn", "ipc", `allowlist drift: ${missingFromAllowlist.length} methods denied (not in allowlist)`);
    console.warn(`[electron] IPC allowlist drift: ${missingFromAllowlist.length} AppState methods not in allowlist (denied to renderer) — regenerate with \`pnpm --filter desktop e2e:gen\``);
  }

  let registered = 0;
  for (const m of IPC_ALLOWLIST) {
    if (!protoMethods.has(m)) continue;
    ipcMain.handle(m, async (_e, ...args: unknown[]) => {
      try {
        if (stubs && m in stubs) return await stubs[m](...args);
        return await (appState as any)[m](...args);
      } catch (err) {
        const msg = err instanceof Error
          ? err.message
          : typeof err === "string" ? err : String(err);
        logEvent("warn", "ipc", `${m} failed: ${msg}`);
        throw err instanceof Error ? err : new Error(String(err));
      }
    });
    tracked.add(m);
    registered++;
  }
  logEvent("info", "ipc", `registered ${registered} handlers (allowlist ${IPC_ALLOWLIST.length}, methods ${protoMethods.size})`);
  console.log(`[electron] Registered ${registered} IPC handlers (allowlist: ${IPC_ALLOWLIST.length}, AppState methods: ${protoMethods.size})`);
}

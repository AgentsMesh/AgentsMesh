// IPC bridge exposed via preload contextBridge (src/preload/index.ts). Drop-in for @tauri-apps/api/core invoke.
export async function invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const api = (globalThis as any).window?.electronAPI;
  if (!api?.invoke) throw new Error("electronAPI not available");
  const values = args ? Object.values(args) : [];
  return api.invoke(cmd, ...values) as Promise<T>;
}

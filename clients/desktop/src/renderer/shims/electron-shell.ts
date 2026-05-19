export async function open(url: string): Promise<void> {
  const api = (globalThis as any).window?.electronAPI;
  if (api?.invoke) {
    await api.invoke("shellOpen", url);
  } else {
    window.open(url, "_blank", "noopener,noreferrer");
  }
}

export const Command = class {
  constructor(_program: string, _args?: string[]) {}
  async execute(): Promise<{ stdout: string; stderr: string; code: number }> {
    throw new Error("Command execution not supported in Electron renderer");
  }
};

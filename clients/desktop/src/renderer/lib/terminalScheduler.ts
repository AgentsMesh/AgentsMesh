import type { Terminal as XTerm } from "@xterm/xterm";

export class TerminalWriteScheduler {
  private pendingData: Uint8Array[] = [];
  private rafId: number | null = null;
  private terminal: XTerm | null = null;

  attach(terminal: XTerm): void {
    this.terminal = terminal;
  }

  schedule(data: Uint8Array): void {
    this.pendingData.push(data);

    if (this.rafId === null) {
      this.rafId = requestAnimationFrame(() => this.flush());
    }
  }

  private flush(): void {
    this.rafId = null;

    if (!this.terminal || this.pendingData.length === 0) {
      return;
    }

    const totalLength = this.pendingData.reduce((sum, d) => sum + d.length, 0);
    const combined = new Uint8Array(totalLength);
    let offset = 0;
    for (const d of this.pendingData) {
      combined.set(d, offset);
      offset += d.length;
    }

    this.terminal.write(combined);

    this.pendingData = [];
  }

  dispose(): void {
    if (this.rafId !== null) {
      cancelAnimationFrame(this.rafId);
      this.rafId = null;
    }
    this.pendingData = [];
    this.terminal = null;
  }
}

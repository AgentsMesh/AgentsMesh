import type { Terminal as XTerm } from "@xterm/xterm";

/**
 * TerminalWriteScheduler uses requestAnimationFrame to aggregate xterm writes.
 * This merges high-frequency WebSocket messages into a single animation frame,
 * reducing xterm.write() calls from 4000-6700/s to ~60/s (monitor refresh rate).
 *
 * This eliminates visual flickering caused by excessive render updates while
 * maintaining data integrity - no bytes are lost.
 */
export class TerminalWriteScheduler {
  private pendingData: Uint8Array[] = [];
  private rafId: number | null = null;
  private terminal: XTerm | null = null;

  /**
   * Attach this scheduler to an xterm instance.
   * Must be called before scheduling writes.
   */
  attach(terminal: XTerm): void {
    this.terminal = terminal;
  }

  /**
   * Schedule data to be written to the terminal.
   * Data is accumulated and written in the next animation frame.
   */
  schedule(data: Uint8Array): void {
    this.pendingData.push(data);

    // Only request one animation frame at a time
    if (this.rafId === null) {
      this.rafId = requestAnimationFrame(() => this.flush());
    }
  }

  /**
   * Flush all pending data to the terminal.
   * Called automatically via requestAnimationFrame.
   */
  private flush(): void {
    this.rafId = null;

    if (!this.terminal || this.pendingData.length === 0) {
      return;
    }

    // Combine all pending data into a single Uint8Array
    const totalLength = this.pendingData.reduce((sum, d) => sum + d.length, 0);
    const combined = new Uint8Array(totalLength);
    let offset = 0;
    for (const d of this.pendingData) {
      combined.set(d, offset);
      offset += d.length;
    }

    // Write combined data to terminal in one call
    this.terminal.write(combined);

    // Clear pending data
    this.pendingData = [];
  }

  /**
   * Dispose the scheduler, canceling any pending animation frame.
   * Should be called when the terminal is being destroyed.
   */
  dispose(): void {
    if (this.rafId !== null) {
      cancelAnimationFrame(this.rafId);
      this.rafId = null;
    }
    this.pendingData = [];
    this.terminal = null;
  }
}

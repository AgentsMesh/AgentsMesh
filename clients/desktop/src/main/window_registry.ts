import { type BrowserWindow } from "electron";
import { type WindowKind, windowTraits } from "../shared/window-kind";

export { type WindowKind } from "../shared/window-kind";

interface Entry {
  win: BrowserWindow;
  kind: WindowKind;
}

// `primaryId` is derived, recomputed through one path (recomputePrimary) so
// register and election can't drift; a primary is always primaryEligible, so
// chromeless popouts never receive oauth/updater/notifications.
export class WindowRegistry {
  private entries = new Map<number, Entry>();
  private primaryId: number | null = null;
  private onPrimaryChange?: () => void;

  setPrimaryChangeListener(cb: () => void): void {
    this.onPrimaryChange = cb;
  }

  register(win: BrowserWindow, kind: WindowKind): void {
    this.entries.set(win.webContents.id, { win, kind });
    this.recomputePrimary();
  }

  unregister(wcId: number): void {
    this.entries.delete(wcId);
    this.recomputePrimary();
  }

  getPrimary(): BrowserWindow | null {
    const e = this.primaryId === null ? undefined : this.entries.get(this.primaryId);
    return e && !e.win.isDestroyed() ? e.win : null;
  }

  isPrimary(wcId: number): boolean {
    return this.primaryId === wcId;
  }

  has(wcId: number): boolean {
    const e = this.entries.get(wcId);
    return e !== undefined && !e.win.isDestroyed();
  }

  hasWindows(): boolean {
    for (const e of this.entries.values()) if (!e.win.isDestroyed()) return true;
    return false;
  }

  all(): BrowserWindow[] {
    return [...this.entries.values()].map((e) => e.win).filter((w) => !w.isDestroyed());
  }

  // On a URL switch, chromeless popouts close instead of reloading — their pod/
  // channel lives on the old backend and can't migrate, so a reload only wedges.
  applyConfigChange(urlChanged: boolean): void {
    for (const e of [...this.entries.values()]) {
      if (e.win.isDestroyed()) continue;
      const t = windowTraits(e.kind);
      if (urlChanged && !t.primaryEligible) e.win.close();
      else if (urlChanged || t.reloadOnConfigEdit) e.win.reload();
    }
  }

  broadcast(channel: string, payload: unknown): void {
    for (const w of this.all()) w.webContents.send(channel, payload);
  }

  sendTo(wcId: number, channel: string, payload: unknown): void {
    const e = this.entries.get(wcId);
    if (e && !e.win.isDestroyed()) e.win.webContents.send(channel, payload);
  }

  private isLiveEligible(wcId: number): boolean {
    const e = this.entries.get(wcId);
    return e !== undefined && !e.win.isDestroyed() && windowTraits(e.kind).primaryEligible;
  }

  // No preemption: keep a live+eligible incumbent; re-elect only once it's gone.
  private recomputePrimary(): void {
    const prev = this.primaryId;
    const keep = this.primaryId !== null && this.isLiveEligible(this.primaryId);
    if (!keep) this.primaryId = this.electPrimary();
    if (this.primaryId !== prev) this.onPrimaryChange?.();
  }

  private electPrimary(): number | null {
    for (const id of this.entries.keys()) {
      if (this.isLiveEligible(id)) return id;
    }
    return null;
  }
}

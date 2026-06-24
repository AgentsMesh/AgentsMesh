import { describe, it, expect, vi } from "vitest";
import type { BrowserWindow } from "electron";
import { WindowRegistry, type WindowKind } from "./window_registry";

let nextId = 1;
function fakeWin(destroyed = false) {
  const send = vi.fn();
  const reload = vi.fn();
  const close = vi.fn();
  const id = nextId++;
  const win = {
    webContents: { id, send },
    isDestroyed: () => destroyed,
    reload,
    close,
  } as unknown as BrowserWindow;
  return { win, id, send, reload, close };
}

function reg(r: WindowRegistry, kind: WindowKind, destroyed = false) {
  const f = fakeWin(destroyed);
  r.register(f.win, kind);
  return f;
}

describe("WindowRegistry · primary election", () => {
  it("the first primary wins", () => {
    const r = new WindowRegistry();
    const p = reg(r, "primary");
    expect(r.getPrimary()).toBe(p.win);
    expect(r.isPrimary(p.id)).toBe(true);
  });

  it("a chromeless popout alone is NOT elected primary (#2)", () => {
    const r = new WindowRegistry();
    reg(r, "terminal");
    expect(r.getPrimary()).toBeNull();
    reg(r, "channel");
    expect(r.getPrimary()).toBeNull();
  });

  it("a terminal never wins even as the first window — register agrees with electPrimary (#5)", () => {
    const r = new WindowRegistry();
    const t = reg(r, "terminal");
    expect(r.isPrimary(t.id)).toBe(false);
    expect(r.getPrimary()).toBeNull();
  });

  it("a new secondary does not preempt a live primary", () => {
    const r = new WindowRegistry();
    const p = reg(r, "primary");
    reg(r, "secondary");
    expect(r.getPrimary()).toBe(p.win);
  });

  it("closing the primary re-elects the next eligible window", () => {
    const r = new WindowRegistry();
    const p = reg(r, "primary");
    const s = reg(r, "secondary");
    r.unregister(p.id);
    expect(r.getPrimary()).toBe(s.win);
  });

  it("with only popouts left, getPrimary is null (#2/#4)", () => {
    const r = new WindowRegistry();
    const p = reg(r, "primary");
    reg(r, "terminal");
    r.unregister(p.id);
    expect(r.getPrimary()).toBeNull();
  });

  it("a destroyed primary is never returned", () => {
    const r = new WindowRegistry();
    reg(r, "primary", /* destroyed */ true);
    expect(r.getPrimary()).toBeNull();
  });

  it("fires onPrimaryChange only when the designation changes", () => {
    const r = new WindowRegistry();
    const cb = vi.fn();
    r.setPrimaryChangeListener(cb);
    reg(r, "primary"); // null → primary
    expect(cb).toHaveBeenCalledTimes(1);
    reg(r, "secondary"); // incumbent kept, no change
    expect(cb).toHaveBeenCalledTimes(1);
    reg(r, "terminal"); // popout, no change
    expect(cb).toHaveBeenCalledTimes(1);
  });
});

describe("WindowRegistry · membership", () => {
  it("has / hasWindows reflect live membership", () => {
    const r = new WindowRegistry();
    expect(r.hasWindows()).toBe(false);
    const p = reg(r, "primary");
    expect(r.has(p.id)).toBe(true);
    expect(r.hasWindows()).toBe(true);
    r.unregister(p.id);
    expect(r.has(p.id)).toBe(false);
    expect(r.hasWindows()).toBe(false);
  });

  it("a destroyed window is not 'live' for has/hasWindows", () => {
    const r = new WindowRegistry();
    const d = reg(r, "terminal", true);
    expect(r.has(d.id)).toBe(false);
    expect(r.hasWindows()).toBe(false);
  });
});

describe("WindowRegistry · applyConfigChange", () => {
  it("server switch: chromeless popouts close, full windows reload (#4/C)", () => {
    const r = new WindowRegistry();
    const p = reg(r, "primary");
    const s = reg(r, "secondary");
    const t = reg(r, "terminal");
    const ch = reg(r, "channel");
    r.applyConfigChange(/* urlChanged */ true);
    expect(p.reload).toHaveBeenCalledTimes(1);
    expect(s.reload).toHaveBeenCalledTimes(1);
    expect(t.close).toHaveBeenCalledTimes(1);
    expect(ch.close).toHaveBeenCalledTimes(1);
    expect(t.reload).not.toHaveBeenCalled();
    expect(p.close).not.toHaveBeenCalled();
  });

  it("cosmetic same-URL edit: reload reloadOnConfigEdit windows, spare live terminals", () => {
    const r = new WindowRegistry();
    const p = reg(r, "primary");
    const t = reg(r, "terminal");
    const ch = reg(r, "channel");
    r.applyConfigChange(/* urlChanged */ false);
    expect(p.reload).toHaveBeenCalledTimes(1);
    expect(ch.reload).toHaveBeenCalledTimes(1);
    expect(t.reload).not.toHaveBeenCalled();
    expect(t.close).not.toHaveBeenCalled();
  });
});

describe("WindowRegistry · messaging", () => {
  it("sendTo targets one window; broadcast hits all live windows", () => {
    const r = new WindowRegistry();
    const a = reg(r, "primary");
    const b = reg(r, "secondary");
    r.sendTo(a.id, "ch", { x: 1 });
    expect(a.send).toHaveBeenCalledWith("ch", { x: 1 });
    expect(b.send).not.toHaveBeenCalled();
    r.broadcast("ev", null);
    expect(a.send).toHaveBeenCalledWith("ev", null);
    expect(b.send).toHaveBeenCalledWith("ev", null);
  });
});

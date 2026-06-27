import { create } from "zustand";

// Per-channel unsent message drafts. Client-local (no server / cross-device
// sync — YAGNI), persisted to localStorage so a draft survives reload. Kept in
// a zustand store (not Rust core) per the plan's sanctioned interim: this is
// pure composer view-state, and the sidebar needs it reactively for [Draft].
const STORAGE_KEY = "agentsmesh.channel-drafts";

function load(): Record<number, string> {
  if (typeof window === "undefined") return {};
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}") as Record<number, string>;
  } catch {
    return {};
  }
}

function persist(drafts: Record<number, string>) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(drafts));
  } catch {
    // quota / private mode — drafts are best-effort
  }
}

interface DraftState {
  drafts: Record<number, string>;
  setDraft: (channelId: number, text: string) => void;
  clearDraft: (channelId: number) => void;
}

export const useChannelDraftStore = create<DraftState>((set) => ({
  drafts: load(),
  setDraft: (channelId, text) =>
    set((s) => {
      const drafts = { ...s.drafts };
      if (text.trim()) drafts[channelId] = text;
      else delete drafts[channelId];
      persist(drafts);
      return { drafts };
    }),
  clearDraft: (channelId) =>
    set((s) => {
      if (!(channelId in s.drafts)) return s;
      const drafts = { ...s.drafts };
      delete drafts[channelId];
      persist(drafts);
      return { drafts };
    }),
}));

export const getChannelDraft = (channelId: number): string =>
  useChannelDraftStore.getState().drafts[channelId] ?? "";
export const setChannelDraft = (channelId: number, text: string) =>
  useChannelDraftStore.getState().setDraft(channelId, text);
export const clearChannelDraft = (channelId: number) =>
  useChannelDraftStore.getState().clearDraft(channelId);

// Narrow per-row selector: a row re-renders only when ITS draft flips, so a
// keystroke in one channel's composer doesn't re-render the whole sidebar.
export const useChannelHasDraft = (channelId: number): boolean =>
  useChannelDraftStore((s) => Boolean(s.drafts[channelId]?.trim()));

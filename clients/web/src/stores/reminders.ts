import { create } from "zustand";
import { useShallow } from "zustand/react/shallow";

export type ReminderTone = "info" | "warning" | "success";

export interface Reminder {
  id: string;
  tone: ReminderTone;
  message: string;
  onAction?: () => void;
}

interface ReminderState {
  reminders: Record<string, Reminder>;
  signatures: Record<string, string>;
  dismissed: Record<string, string>;
  setReminder: (reminder: Reminder, signature: string) => void;
  clearReminder: (id: string) => void;
  dismiss: (id: string) => void;
}

function omit<T extends Record<string, unknown>>(obj: T, key: string): T {
  if (!(key in obj)) return obj;
  const { [key]: _drop, ...rest } = obj;
  return rest as T;
}

export const useReminderStore = create<ReminderState>((set) => ({
  reminders: {},
  signatures: {},
  dismissed: {},
  setReminder: (reminder, signature) =>
    set((s) => {
      const dismissedSig = s.dismissed[reminder.id];
      // Resurface a dismissed reminder once its underlying state moved on
      // (new version / fresh disconnect episode), keyed by signature.
      const dismissed =
        dismissedSig !== undefined && dismissedSig !== signature
          ? omit(s.dismissed, reminder.id)
          : s.dismissed;
      return {
        reminders: { ...s.reminders, [reminder.id]: reminder },
        signatures: { ...s.signatures, [reminder.id]: signature },
        dismissed,
      };
    }),
  clearReminder: (id) =>
    set((s) => {
      if (!(id in s.reminders) && !(id in s.dismissed)) return s;
      return {
        reminders: omit(s.reminders, id),
        signatures: omit(s.signatures, id),
        dismissed: omit(s.dismissed, id),
      };
    }),
  dismiss: (id) =>
    set((s) => ({ dismissed: { ...s.dismissed, [id]: s.signatures[id] ?? "" } })),
}));

export function useVisibleReminders(): Reminder[] {
  return useReminderStore(
    useShallow((s) => Object.values(s.reminders).filter((r) => !(r.id in s.dismissed))),
  );
}

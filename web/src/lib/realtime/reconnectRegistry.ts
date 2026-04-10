type Priority = "immediate" | "deferred" | "low";

interface ReconnectEntry {
  name: string;
  fn: () => void;
  priority: Priority;
}

const PRIORITY_DELAY: Record<Priority, number> = {
  immediate: 0,
  deferred: 200,
  low: 500,
};

class ReconnectRegistry {
  private entries: ReconnectEntry[] = [];

  register(entry: ReconnectEntry): () => void {
    this.entries.push(entry);
    return () => {
      this.entries = this.entries.filter((e) => e !== entry);
    };
  }

  execute(): () => void {
    const timers: ReturnType<typeof setTimeout>[] = [];
    const grouped = new Map<Priority, ReconnectEntry[]>();
    for (const entry of this.entries) {
      const list = grouped.get(entry.priority) || [];
      list.push(entry);
      grouped.set(entry.priority, list);
    }
    for (const [priority, entries] of grouped) {
      const delay = PRIORITY_DELAY[priority];
      if (delay === 0) {
        entries.forEach((e) => e.fn());
      } else {
        timers.push(setTimeout(() => entries.forEach((e) => e.fn()), delay));
      }
    }
    return () => timers.forEach(clearTimeout);
  }
}

export const reconnectRegistry = new ReconnectRegistry();

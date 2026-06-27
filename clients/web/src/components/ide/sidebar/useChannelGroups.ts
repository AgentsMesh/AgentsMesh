import { useMemo } from "react";
import { getLastMessage, type Channel, type ChannelLastMessage } from "@/stores/channel";

// Activity-weighted sidebar group. Pinned takes precedence over the design's
// "Active / Linked / Quiet" buckets:
//   - Pinned: user-pinned (per-user), always on top
//   - Active: has any message in the last 24h
//   - Linked: tied to a ticket or repository (and not Active)
//   - Quiet:  everything else
export type ChannelGroup = "pinned" | "active" | "linked" | "quiet";
const DAY_MS = 24 * 60 * 60 * 1000;

function classifyChannel(channel: Channel, lastMsg: ChannelLastMessage | null, now: number): ChannelGroup {
  if (channel.is_pinned) return "pinned";
  const ts = lastMsg?.timestamp ?? channel.updated_at;
  if (ts) {
    const t = new Date(ts).getTime();
    if (!Number.isNaN(t) && now - t < DAY_MS) return "active";
  }
  if (channel.ticket || channel.repository) return "linked";
  return "quiet";
}

export interface GroupedRow {
  channel: Channel;
  lastMsg: ChannelLastMessage | null;
}

export type ChannelGroups = Record<ChannelGroup, GroupedRow[]>;

/** Group + sort visible channels: pinned first, then activity buckets, each
 *  newest-first. Re-derives on the WASM `_tick` invalidator. */
export function useChannelGroups(visible: Channel[], tick: number): ChannelGroups {
  return useMemo(() => {
    const now = Date.now();
    const rows: GroupedRow[] = visible.map((channel) => ({
      channel,
      lastMsg: getLastMessage(channel.id),
    }));
    rows.sort((a, b) => {
      const ta = a.lastMsg?.timestamp ?? a.channel.updated_at ?? "";
      const tb = b.lastMsg?.timestamp ?? b.channel.updated_at ?? "";
      return tb.localeCompare(ta);
    });
    const groups: ChannelGroups = { pinned: [], active: [], linked: [], quiet: [] };
    for (const row of rows) {
      groups[classifyChannel(row.channel, row.lastMsg, now)].push(row);
    }
    return groups;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, tick]);
}

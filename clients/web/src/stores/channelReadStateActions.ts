// Read-state action slice: unread/mention counts, mark read/unread, mute, pin.
// Composed into useChannelMessageStore via the zustand slices pattern. Split
// from channelMessageStore.ts for SRP (file size limit).

import type { StateCreator } from "zustand";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import { readCurrentOrg } from "@/stores/auth";
import { getChannelState } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import {
  getChannelUnreadCounts,
  markChannelRead,
  markChannelUnread as markChannelUnreadConnect,
  muteChannel as muteChannelConnect,
  pinChannel as pinChannelConnect,
} from "@/lib/api/facade/channelConnect";
import { ReplaceChannelUnreadCountsRequestSchema } from "@proto/channel_state/v1/mutations_pb";
import type { ChannelMessageState } from "./channelMessageTypes";

type ReadStateSlice = Pick<
  ChannelMessageState,
  | "fetchUnreadCounts" | "markRead" | "muteChannel" | "pinChannel"
  | "markUnread" | "incrementUnread" | "clearChannelUnread"
>;

const svc = () => getChannelState();
const orgSlug = () => readCurrentOrg()?.slug ?? "";

export const createReadStateActions: StateCreator<ChannelMessageState, [], [], ReadStateSlice> =
  (set, get) => ({
    fetchUnreadCounts: async () => {
      try {
        const summary = await getChannelUnreadCounts(orgSlug());
        const numKeyed = (m: Record<string, number>) =>
          Object.fromEntries(Object.entries(m).map(([k, v]) => [BigInt(k), v]));
        const req = protoCreate(ReplaceChannelUnreadCountsRequestSchema, {
          counts: numKeyed(summary.unread) as unknown as { [k: string]: number },
          mentions: numKeyed(summary.mentions) as unknown as { [k: string]: number },
          lastRead: Object.fromEntries(
            Object.entries(summary.lastRead).map(([k, v]) => [BigInt(k), BigInt(v)]),
          ) as unknown as { [k: string]: bigint },
          manuallyUnread: Object.fromEntries(
            Object.entries(summary.manuallyUnread).map(([k, v]) => [BigInt(k), v]),
          ) as unknown as { [k: string]: boolean },
        });
        svc().replace_channel_unread_counts(toBinary(ReplaceChannelUnreadCountsRequestSchema, req));
        set((s) => ({ _unreadTick: s._unreadTick + 1 }));
      } catch (error: unknown) {
        console.error("Failed to fetch unread counts:", getErrorMessage(error, "Unknown error"));
      }
    },

    markRead: async (channelId, messageId) => {
      // Dismiss locally BEFORE the network round-trip: a fast leave/re-enter must
      // see the advanced cursor + cleared counts (useChannelEntryAnchor snapshots
      // them synchronously). clear_channel_mentions too — reading an @ dismisses
      // it, matching select_channel; the markRead-only entry paths (mobile /
      // popout / IDE panel) would otherwise strand mention_count → a red "0".
      svc().clear_channel_unread(BigInt(channelId));
      svc().clear_channel_mentions(BigInt(channelId));
      svc().advance_last_read(BigInt(channelId), BigInt(messageId));
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
      try {
        await markChannelRead(orgSlug(), channelId, messageId);
      } catch (error: unknown) {
        console.error("Failed to mark channel as read:", getErrorMessage(error, "Unknown error"));
      }
    },

    muteChannel: async (channelId, muted) => {
      try {
        await muteChannelConnect(orgSlug(), channelId, muted);
      } catch (error: unknown) {
        console.error("Failed to update mute setting:", getErrorMessage(error, "Unknown error"));
        throw error;
      }
    },

    pinChannel: async (channelId, pinned) => {
      try {
        await pinChannelConnect(orgSlug(), channelId, pinned);
      } catch (error: unknown) {
        console.error("Failed to update pin setting:", getErrorMessage(error, "Unknown error"));
        throw error;
      }
    },

    markUnread: async (channelId) => {
      try {
        await markChannelUnreadConnect(orgSlug(), channelId);
        await get().fetchUnreadCounts();
      } catch (error: unknown) {
        console.error("Failed to mark channel unread:", getErrorMessage(error, "Unknown error"));
      }
    },

    incrementUnread: (channelId) => {
      svc().increment_unread(BigInt(channelId));
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
    },

    clearChannelUnread: (channelId) => {
      svc().clear_channel_unread(BigInt(channelId));
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
    },
  });

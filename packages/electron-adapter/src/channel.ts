import { invoke } from "./invoke";
import type { IChannelService } from "@agentsmesh/service-interface";
import { ChannelLocalState } from "./channel_state";

export class ElectronChannelService extends ChannelLocalState implements IChannelService {
  async fetch_channels(includeArchived?: boolean | null): Promise<string> {
    const result = await invoke<string>("channelFetchChannels", includeArchived);
    // Rust IPC returns the serialized `ChannelListResponse` envelope — unwrap
    // so `channels_json()` hands callers a bare array (matches WASM shape).
    try {
      const parsed = JSON.parse(result) as { channels?: unknown[] };
      this._channelsCache = JSON.stringify(Array.isArray(parsed.channels) ? parsed.channels : parsed);
    } catch {
      this._channelsCache = "[]";
    }
    return result;
  }

  async fetch_channel(id: bigint): Promise<string> {
    const result = await invoke<string>("channelFetchChannel", Number(id));
    this.update_channel_local(id, result);
    return result;
  }

  async fetch_messages(channelId: bigint, limit?: number | null, beforeId?: bigint | null): Promise<string> {
    const result = await invoke<string>(
      "channelFetchMessages",
      Number(channelId),
      limit,
      beforeId ? Number(beforeId) : null,
    );
    // Backend shape: { messages, has_more }. Mirror WASM cache format exactly
    // so the shared store's `readMessages` sees the same payload on both.
    const parsed = JSON.parse(result) as { messages?: unknown[]; has_more?: boolean };
    this._messagesCache.set(String(channelId), {
      messages: Array.isArray(parsed.messages) ? parsed.messages : [],
      has_more: parsed.has_more ?? false,
    });
    return result;
  }

  async fetch_unread_counts(): Promise<string> {
    const result = await invoke<string>("channelFetchUnreadCounts");
    this._unreadCountsCache = result;
    return result;
  }

  async create_channel(json: string): Promise<string> {
    const result = await invoke<string>("channelCreateChannel", json);
    this.add_channel_local(result);
    return result;
  }

  async update_channel(id: bigint, json: string): Promise<string> {
    const result = await invoke<string>("channelUpdateChannel", Number(id), json);
    this.update_channel_local(id, result);
    return result;
  }

  async archive_channel(id: bigint): Promise<void> {
    await invoke<void>("channelArchiveChannel", Number(id));
  }

  async unarchive_channel(id: bigint): Promise<void> {
    await invoke<void>("channelUnarchiveChannel", Number(id));
  }

  async send_message(channelId: bigint, json: string): Promise<string> {
    const result = await invoke<string>("channelSendMessage", Number(channelId), json);
    // Rust service unwraps the `{message:{}}` envelope — result is the bare
    // message JSON. Route through add_message so the de-dup logic runs.
    this.add_message(channelId, result);
    return result;
  }

  async edit_message(channelId: bigint, messageId: bigint, content: string): Promise<string> {
    const result = await invoke<string>(
      "channelEditMessage",
      Number(channelId),
      Number(messageId),
      content,
    );
    this.update_message_local(channelId, result);
    return result;
  }

  async delete_message(channelId: bigint, messageId: bigint): Promise<void> {
    await invoke<void>("channelDeleteMessage", Number(channelId), Number(messageId));
    this.remove_message_local(channelId, messageId);
  }

  async mark_read(channelId: bigint, messageId: bigint): Promise<void> {
    await invoke<void>("channelMarkRead", Number(channelId), Number(messageId));
    this.clear_channel_unread(channelId);
  }

  async mute_channel(channelId: bigint, muted: boolean): Promise<void> {
    await invoke<void>("channelMuteChannel", Number(channelId), muted);
  }

  async join_channel(channelId: bigint, podKey: string): Promise<string> {
    const result = await invoke<string>("channelJoinChannel", Number(channelId), podKey);
    // Rust ChannelService.join_channel already refreshed pods_by_channel on
    // the main side; refresh the renderer mirror so the synchronous
    // channel_pods_json reader (useChannelPods.readPodsFromRust) sees the
    // joined pod immediately without waiting for the next get_channel_pods.
    await this.get_channel_pods(channelId).catch(() => undefined);
    return result;
  }

  async leave_channel(channelId: bigint, podKey: string): Promise<string> {
    const result = await invoke<string>("channelLeaveChannel", Number(channelId), podKey);
    await this.get_channel_pods(channelId).catch(() => undefined);
    return result;
  }

  async get_channel_pods(id: bigint): Promise<string> {
    const result = await invoke<string>("channelGetChannelPods", Number(id));
    // Backend envelope: `{ pods: [...], total: N }`. Mirror into the
    // renderer-side pods cache so subsequent channel_pods_json reads
    // are synchronous (matches WASM behaviour).
    try {
      const parsed = JSON.parse(result) as { pods?: unknown[] };
      this.set_channel_pods(id, JSON.stringify(Array.isArray(parsed.pods) ? parsed.pods : []));
    } catch {
      this.set_channel_pods(id, "[]");
    }
    return result;
  }
}

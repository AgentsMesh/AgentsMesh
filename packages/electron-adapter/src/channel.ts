import { invoke } from "./invoke";
import type { IChannelService } from "@agentsmesh/service-interface";
import { ChannelLocalState } from "./channel_state";
import { fromBinary, create, toBinary } from "@bufbuild/protobuf";
import {
  InsertChannelRequestSchema,
  PatchChannelMemberCountRequestSchema,
  InsertChannelMessageRequestSchema,
  ApplyIncomingChannelMessageRequestSchema,
  ApplyChannelMessageEditedEventRequestSchema,
  ReplaceChannelUnreadCountsRequestSchema,
  ReplaceChannelPodsRequestSchema,
  ReplaceChannelMembersRequestSchema,
  RemoveChannelMemberRequestSchema,
} from "@agentsmesh/proto/channel_state/v1/mutations_pb";
import {
  ChannelSchema,
  ListChannelsResponseSchema,
  ListChannelMessagesResponseSchema,
  ListChannelMembersResponseSchema,
  ListChannelPodsResponseSchema,
} from "@agentsmesh/proto/channel/v1/channel_pb";
import { channelToCache, messageToCache, podToCache, memberToCache } from "./channel_proto_to_cache";
import {
  channelsBytes, channelBytes, currentChannelBytes, messagesBytes,
  lastMessageBytes, membersBytes, podsBytes,
} from "./channel_cache_to_bytes";

export class ElectronChannelService extends ChannelLocalState implements IChannelService {
  // Fetch→state (B): decode wire ListX response → renderer cache (sync, for
  // reactivity) + fire the SAME wire bytes to main so runtime.state (the SSOT
  // the realtime snapshot reads) folds the identical baseline. Mirrors the
  // wasm apply_fetched_* methods so the shared store drives both ends alike.
  apply_fetched_channels(respBytes: Uint8Array): void {
    const resp = fromBinary(ListChannelsResponseSchema, respBytes);
    this._channelsCache = JSON.stringify(resp.items.map(channelToCache));
    void invoke<void>("appChannelApplyFetchedChannels", Array.from(respBytes)).catch(() => undefined);
  }

  apply_fetched_messages(channelId: bigint, respBytes: Uint8Array): void {
    const resp = fromBinary(ListChannelMessagesResponseSchema, respBytes);
    this._messagesCache.set(String(channelId), {
      messages: resp.items.map(messageToCache), has_more: resp.hasMore,
    });
    void invoke<void>("appChannelApplyFetchedMessages", Number(channelId), Array.from(respBytes)).catch(() => undefined);
  }

  apply_fetched_messages_prepend(channelId: bigint, respBytes: Uint8Array): void {
    const resp = fromBinary(ListChannelMessagesResponseSchema, respBytes);
    const key = String(channelId);
    const entry = this._messagesCache.get(key) ?? { messages: [], has_more: false };
    const older = resp.items.map(messageToCache);
    const existingIds = new Set((entry.messages as { id: number }[]).map((m) => m.id));
    const merged = [...older.filter((m) => !existingIds.has(m.id as number)), ...entry.messages];
    this._messagesCache.set(key, { messages: merged, has_more: resp.hasMore });
    void invoke<void>("appChannelApplyFetchedMessagesPrepend", Number(channelId), Array.from(respBytes)).catch(() => undefined);
  }

  apply_fetched_members(channelId: bigint, respBytes: Uint8Array): void {
    const resp = fromBinary(ListChannelMembersResponseSchema, respBytes);
    this.set_channel_members(channelId, JSON.stringify(resp.items.map(memberToCache)));
    void invoke<void>("appChannelApplyFetchedMembers", Number(channelId), Array.from(respBytes)).catch(() => undefined);
  }

  apply_fetched_pods(channelId: bigint, respBytes: Uint8Array): void {
    const resp = fromBinary(ListChannelPodsResponseSchema, respBytes);
    this.set_channel_pods(channelId, JSON.stringify(resp.items.map(podToCache)));
    void invoke<void>("appChannelApplyFetchedPods", Number(channelId), Array.from(respBytes)).catch(() => undefined);
  }

  // Single-object fetch (B): decode wire GetChannel response (Channel) + upsert
  // into the renderer cache + fan the SAME wire bytes to main.
  apply_fetched_channel(respBytes: Uint8Array): void {
    const channel = fromBinary(ChannelSchema, respBytes);
    const c = channelToCache(channel);
    const list = JSON.parse(this._channelsCache) as { id: number }[];
    const idx = list.findIndex((x) => x.id === c.id);
    if (idx >= 0) list[idx] = { ...list[idx], ...c };
    else list.unshift(c as { id: number });
    this._channelsCache = JSON.stringify(list);
    void invoke<void>("appChannelApplyFetchedChannel", Array.from(respBytes)).catch(() => undefined);
  }

  // Read side (B, zero-JSON): re-encode the renderer cache into state proto
  // bytes so the shared selectors decode desktop and web identically.
  channels_bytes(): Uint8Array { return channelsBytes(this._channelsCache); }
  unread_counts_bytes(): Uint8Array {
    const counts = JSON.parse(this._unreadCountsCache) as Record<string, number>;
    return toBinary(ReplaceChannelUnreadCountsRequestSchema,
      create(ReplaceChannelUnreadCountsRequestSchema, { counts }));
  }
  get_channel_bytes(id: bigint): Uint8Array { return channelBytes(this._channelsCache, Number(id)); }
  current_channel_bytes(): Uint8Array { return currentChannelBytes(this._channelsCache, this._currentChannelId); }
  get_messages_bytes(channelId: bigint): Uint8Array {
    const entry = this._messagesCache.get(String(channelId));
    return messagesBytes(Number(channelId), (entry?.messages ?? []) as Record<string, unknown>[], entry?.has_more ?? false);
  }
  get_last_message_bytes(channelId: bigint): Uint8Array {
    const entry = this._messagesCache.get(String(channelId));
    return lastMessageBytes((entry?.messages ?? []) as Record<string, unknown>[]);
  }
  channel_members_bytes(channelId: bigint): Uint8Array {
    return membersBytes(Number(channelId), this.channel_members_json(channelId));
  }
  channel_pods_bytes(channelId: bigint): Uint8Array {
    return podsBytes(Number(channelId), this.channel_pods_json(channelId));
  }

  async create_channel(json: string): Promise<string> {
    const result = await invoke<string>("channelCreateChannel", json);
    this.add_channel_local(result);
    return result;
  }

  async update_channel(id: bigint, json: string): Promise<string> {
    const result = await invoke<string>("channelUpdateChannel", Number(id), json);
    this.upsert_channel_cache_from_json(id, result);
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
    try {
      const parsed = JSON.parse(result) as { pods?: unknown[] };
      this.set_channel_pods(id, JSON.stringify(Array.isArray(parsed.pods) ? parsed.pods : []));
    } catch {
      this.set_channel_pods(id, "[]");
    }
    return result;
  }

  // Local-cache helper used by fetch_channel / update_channel after a JSON
  // legacy IPC response. Decodes the JSON envelope and upserts into
  // _channelsCache. Distinct from insert_channel (which takes proto bytes).
  private upsert_channel_cache_from_json(id: bigint, json: string): void {
    let patch: Record<string, unknown> | null = null;
    try {
      patch = JSON.parse(json) as Record<string, unknown>;
    } catch {
      return;
    }
    const list = JSON.parse(this._channelsCache) as { id: number }[];
    const idx = list.findIndex((x) => x.id === Number(id));
    if (idx >= 0) list[idx] = { ...list[idx], ...patch };
    else if (patch && typeof patch.id === "number") list.unshift(patch as { id: number });
    this._channelsCache = JSON.stringify(list);
  }

  // Proto-bytes mutators decode locally into the JS-side cache so synchronous
  // readers (channels_json / get_messages_json / unread_counts_json) see the
  // mutation immediately. The fire-and-forget NAPI fan-out targets the `app_*`
  // commands so the SAME state the EventBus dispatch hook mutates
  // (runtime.state) gets the fetched baseline — that's what makes the
  // post-dispatch realtime snapshot (main/realtime.ts) complete instead of
  // realtime-only. Not awaited: IPC latency would defeat the sync-cache
  // invariant the renderer's _tick reactivity assumes.
  insert_channel(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(InsertChannelRequestSchema, reqBytes);
    if (req.channel) {
      const c = channelToCache(req.channel);
      const list = JSON.parse(this._channelsCache) as { id: number }[];
      const idx = list.findIndex((x) => x.id === c.id);
      if (idx >= 0) list[idx] = { ...list[idx], ...c };
      else list.unshift(c as { id: number });
      this._channelsCache = JSON.stringify(list);
    }
    void invoke<void>("appChannelInsertChannel", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  replace_channel_pods(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(ReplaceChannelPodsRequestSchema, reqBytes);
    this.set_channel_pods(req.channelId, JSON.stringify(req.pods.map(podToCache)));
    void invoke<void>("appChannelReplacePods", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  replace_channel_members(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(ReplaceChannelMembersRequestSchema, reqBytes);
    this.set_channel_members(req.channelId, JSON.stringify(req.members.map(memberToCache)));
    void invoke<void>("appChannelReplaceMembers", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  remove_channel_member(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(RemoveChannelMemberRequestSchema, reqBytes);
    const key = String(req.channelId);
    const json = this._membersByChannel.get(key) ?? "[]";
    const members = JSON.parse(json) as Array<{ user_id: number }>;
    const filtered = members.filter((m) => m.user_id !== Number(req.userId));
    this._membersByChannel.set(key, JSON.stringify(filtered));
    void invoke<void>("appChannelRemoveMember", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  patch_channel_member_count(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(PatchChannelMemberCountRequestSchema, reqBytes);
    const id = Number(req.channelId);
    const list = JSON.parse(this._channelsCache) as { id: number; member_count?: number }[];
    const idx = list.findIndex((x) => x.id === id);
    if (idx >= 0) {
      list[idx].member_count = Math.max(0, (list[idx].member_count ?? 0) + req.delta);
      this._channelsCache = JSON.stringify(list);
    }
    void invoke<void>("appChannelPatchMemberCount", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  insert_channel_message(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(InsertChannelMessageRequestSchema, reqBytes);
    if (req.message) {
      const key = String(req.channelId);
      const entry = this._messagesCache.get(key) ?? { messages: [], has_more: false };
      const msg = messageToCache(req.message);
      if (!entry.messages.some((m) => (m as { id: number }).id === msg.id)) {
        entry.messages.push(msg);
      }
      this._messagesCache.set(key, entry);
    }
    void invoke<void>("appChannelInsertMessage", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  apply_incoming_channel_message(reqBytes: Uint8Array): Promise<boolean> {
    const req = fromBinary(ApplyIncomingChannelMessageRequestSchema, reqBytes);
    if (!req.message) return Promise.resolve(false);
    const key = String(req.channelId);
    const entry = this._messagesCache.get(key) ?? { messages: [], has_more: false };
    const msg = messageToCache(req.message);
    const dup = entry.messages.some((m) => (m as { id: number }).id === msg.id);
    if (!dup) {
      entry.messages.push(msg);
      this._messagesCache.set(key, entry);
    }
    void invoke<void>("channelApplyIncomingChannelMessage", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve(!dup);
  }

  apply_channel_message_edited_event(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(ApplyChannelMessageEditedEventRequestSchema, reqBytes);
    const key = String(req.channelId);
    const entry = this._messagesCache.get(key);
    if (entry) {
      const idx = entry.messages.findIndex((m) => (m as { id: number }).id === Number(req.messageId));
      if (idx >= 0) {
        const cur = entry.messages[idx] as Record<string, unknown>;
        entry.messages[idx] = {
          ...cur,
          body: req.body,
          content_json: req.content || undefined,
          edited_at: req.editedAt || undefined,
        };
        this._messagesCache.set(key, entry);
      }
    }
    void invoke<void>("appChannelApplyMessageEdited", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  replace_channel_unread_counts(reqBytes: Uint8Array): Promise<void> {
    const req = fromBinary(ReplaceChannelUnreadCountsRequestSchema, reqBytes);
    const out: Record<string, number> = {};
    for (const [k, v] of Object.entries(req.counts)) {
      out[String(k)] = Number(v);
    }
    this._unreadCountsCache = JSON.stringify(out);
    void invoke<void>("appChannelReplaceUnreadCounts", Array.from(reqBytes)).catch(() => undefined);
    return Promise.resolve();
  }

  remove_message(channelId: bigint, messageId: bigint): void {
    void invoke<void>("appChannelRemoveMessage", Number(channelId), Number(messageId)).catch(() => undefined);
    super.remove_message_local(channelId, messageId);
  }

  // ── UI→Rust signals: forwarded to runtime.state so the main-process SSOT
  //    can compute unread with the self-message + active-channel rules. The
  //    base-class versions only touch the renderer-local cache. ──

  set_current_user_id(id?: bigint | null): void {
    void invoke<void>("appSetCurrentUser", id != null ? Number(id) : null).catch(() => undefined);
  }

  select_channel(id?: bigint | null): unknown {
    void invoke<void>("appSelectChannel", id != null ? Number(id) : null).catch(() => undefined);
    return super.select_channel(id);
  }

  set_current_channel(id?: bigint | null): void {
    void invoke<void>("appSetCurrentChannel", id != null ? Number(id) : null).catch(() => undefined);
    super.set_current_channel(id);
  }

  clear_channel_unread(channelId: bigint): void {
    void invoke<void>("appChannelClearUnread", Number(channelId)).catch(() => undefined);
    super.clear_channel_unread(channelId);
  }
}

// Channel state mirror for Electron. Shapes must align with the WASM service
// so the shared web store (read via get_messages_json, channels_json, etc.)
// receives identical payloads from both platforms.
interface MessageCacheEntry {
  messages: unknown[];
  has_more: boolean;
}

export class ChannelLocalState {
  _channelsCache = "[]";
  /** Current selected channel id — callers resolve the full object on read
   *  so updates to _channelsCache propagate without reassigning here. */
  _currentChannelId: number | null = null;
  // Per-channel `{ messages, has_more }` — matches Rust ChannelService.get_messages_json.
  _messagesCache = new Map<string, MessageCacheEntry>();
  _unreadCountsCache = "{}";
  _mentionCountsCache = "{}";
  // Per-channel pod array JSON — mirrors Rust ChannelState.pods_by_channel.
  // Populated by ElectronChannelService.get_channel_pods (after IPC fetch) and
  // by join/leave once they call get_channel_pods to refresh. Synchronous
  // readers (useChannelPods.readPodsFromRust) need this to surface joined pods
  // in the RightRail without an additional async round trip.
  _podsByChannel = new Map<string, string>();

  channels_json(): string { return this._channelsCache; }
  channel_pods_json(channelId: bigint): string {
    return this._podsByChannel.get(String(channelId)) ?? "[]";
  }
  set_channel_pods(channelId: bigint, json: string): void {
    this._podsByChannel.set(String(channelId), json);
  }
  current_channel_json(): unknown {
    if (this._currentChannelId == null) return null;
    const chs = JSON.parse(this._channelsCache) as { id: number }[];
    const c = chs.find(x => x.id === this._currentChannelId);
    return c ? JSON.stringify(c) : null;
  }
  unread_counts_json(): string { return this._unreadCountsCache; }
  mention_counts_json(): string { return this._mentionCountsCache; }

  get_channel_json(id: bigint): unknown {
    const chs = JSON.parse(this._channelsCache) as { id: number }[];
    const c = chs.find(x => x.id === Number(id));
    return c ? JSON.stringify(c) : null;
  }

  get_messages_json(channelId: bigint): unknown {
    const entry = this._messagesCache.get(String(channelId));
    if (!entry) return null;
    return JSON.stringify(entry);
  }

  get_last_message_json(channelId: bigint): unknown {
    const entry = this._messagesCache.get(String(channelId));
    if (!entry || entry.messages.length === 0) return null;
    return JSON.stringify(entry.messages[entry.messages.length - 1]);
  }

  get_unread_count(channelId: bigint): number {
    const counts = JSON.parse(this._unreadCountsCache) as Record<string, number>;
    return counts[String(channelId)] ?? 0;
  }

  get_mention_count(channelId: bigint): number {
    const counts = JSON.parse(this._mentionCountsCache) as Record<string, number>;
    return counts[String(channelId)] ?? 0;
  }

  total_unread_count(): number {
    const counts = JSON.parse(this._unreadCountsCache) as Record<string, number>;
    const channels = JSON.parse(this._channelsCache) as { id: number }[];
    const orgIds = new Set(channels.map((c) => c.id));
    let sum = 0;
    for (const [k, v] of Object.entries(counts)) {
      if (orgIds.has(Number(k))) sum += v;
    }
    return sum;
  }

  total_mention_count(): number {
    const counts = JSON.parse(this._mentionCountsCache) as Record<string, number>;
    const channels = JSON.parse(this._channelsCache) as { id: number }[];
    const orgIds = new Set(channels.map((c) => c.id));
    let sum = 0;
    for (const [k, v] of Object.entries(counts)) {
      if (orgIds.has(Number(k))) sum += v;
    }
    return sum;
  }

  filter_channels_json(_query: string, _includeArchived: boolean): string {
    return this._channelsCache;
  }

  sorted_channel_ids_json(_mode: string, _includeArchived: boolean): string {
    return "[]";
  }

  set_channels(json: string): void { this._channelsCache = json; }
  set_current_channel(id?: bigint | null): void { this._currentChannelId = id != null ? Number(id) : null; }
  set_current_user(_json: string): void {}
  set_current_user_id(_id?: bigint | null): void {}
  set_unread_counts(json: string): void { this._unreadCountsCache = json; }
  set_mention_counts(json: string): void { this._mentionCountsCache = json; }

  set_messages(channelId: bigint, json: string, hasMore: boolean): void {
    // Accept either a bare array JSON or the wrapped { messages, has_more } shape.
    let messages: unknown[];
    const parsed = JSON.parse(json);
    if (Array.isArray(parsed)) messages = parsed;
    else if (parsed && Array.isArray(parsed.messages)) messages = parsed.messages;
    else messages = [];
    this._messagesCache.set(String(channelId), { messages, has_more: hasMore });
  }

  set_last_message(_channelId: bigint, _json: string): void {}

  add_channel_local(json: string): void {
    const chs = JSON.parse(this._channelsCache) as unknown[];
    chs.push(JSON.parse(json));
    this._channelsCache = JSON.stringify(chs);
  }

  remove_channel_local(id: bigint): void {
    const chs = JSON.parse(this._channelsCache) as { id: number }[];
    this._channelsCache = JSON.stringify(chs.filter(x => x.id !== Number(id)));
  }

  update_channel_local(id: bigint, json: string): void {
    const chs = JSON.parse(this._channelsCache) as { id: number }[];
    const idx = chs.findIndex(x => x.id === Number(id));
    if (idx >= 0) chs[idx] = { ...chs[idx], ...JSON.parse(json) };
    this._channelsCache = JSON.stringify(chs);
  }

  add_message(channelId: bigint, json: string): void {
    const key = String(channelId);
    const entry = this._messagesCache.get(key) ?? { messages: [], has_more: false };
    const msg = JSON.parse(json) as { id: number };
    // De-dup by id — realtime + REST echo can land in any order.
    if (!entry.messages.some((m) => (m as { id: number }).id === msg.id)) {
      entry.messages.push(msg);
    }
    this._messagesCache.set(key, entry);
  }

  remove_message_local(channelId: bigint, messageId: bigint): void {
    const key = String(channelId);
    const entry = this._messagesCache.get(key);
    if (!entry) return;
    entry.messages = entry.messages.filter((m) => (m as { id: number }).id !== Number(messageId));
    this._messagesCache.set(key, entry);
  }

  update_message_local(channelId: bigint, json: string): void {
    const key = String(channelId);
    const entry = this._messagesCache.get(key);
    if (!entry) return;
    const patch = JSON.parse(json) as { id: number };
    const idx = entry.messages.findIndex((m) => (m as { id: number }).id === patch.id);
    // Merge rather than replace so partial updates (e.g. edited_at only) don't
    // wipe sender / content fields already in cache.
    if (idx >= 0) entry.messages[idx] = { ...(entry.messages[idx] as object), ...patch };
    this._messagesCache.set(key, entry);
  }

  prepend_messages(channelId: bigint, json: string, hasMore: boolean): void {
    const key = String(channelId);
    const entry = this._messagesCache.get(key) ?? { messages: [], has_more: false };
    const newer = JSON.parse(json) as unknown[];
    entry.messages = [...newer, ...entry.messages];
    entry.has_more = hasMore;
    this._messagesCache.set(key, entry);
  }

  // Realtime arrival — mirror Rust ChannelState.on_new_message which adds + dedups.
  on_new_message(json: string): boolean {
    const msg = JSON.parse(json) as { id: number; channel_id: number };
    const key = String(msg.channel_id);
    const entry = this._messagesCache.get(key) ?? { messages: [], has_more: false };
    if (entry.messages.some((m) => (m as { id: number }).id === msg.id)) {
      return false;
    }
    entry.messages.push(msg);
    this._messagesCache.set(key, entry);
    return true;
  }

  increment_unread(channelId: bigint): void {
    const counts = JSON.parse(this._unreadCountsCache) as Record<string, number>;
    counts[String(channelId)] = (counts[String(channelId)] ?? 0) + 1;
    this._unreadCountsCache = JSON.stringify(counts);
  }

  increment_mention(channelId: bigint): void {
    const counts = JSON.parse(this._mentionCountsCache) as Record<string, number>;
    counts[String(channelId)] = (counts[String(channelId)] ?? 0) + 1;
    this._mentionCountsCache = JSON.stringify(counts);
  }

  clear_channel_unread(channelId: bigint): void {
    const counts = JSON.parse(this._unreadCountsCache) as Record<string, number>;
    delete counts[String(channelId)];
    this._unreadCountsCache = JSON.stringify(counts);
  }

  clear_channel_mentions(channelId: bigint): void {
    const counts = JSON.parse(this._mentionCountsCache) as Record<string, number>;
    delete counts[String(channelId)];
    this._mentionCountsCache = JSON.stringify(counts);
  }

  select_channel(id?: bigint | null): unknown {
    this.set_current_channel(id);
    if (id == null) return null;
    return this.get_channel_json(id);
  }
}

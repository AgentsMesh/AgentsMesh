import '@testing-library/jest-dom'
import { vi, afterEach } from 'vitest'
import { fromBinary } from '@bufbuild/protobuf'
import {
  ReplaceCachedChannelsRequestSchema,
  InsertChannelRequestSchema,
  PatchChannelMemberCountRequestSchema,
  ReplaceCachedChannelMessagesRequestSchema,
  PrependCachedChannelMessagesRequestSchema,
  InsertChannelMessageRequestSchema,
  ApplyIncomingChannelMessageRequestSchema,
  ApplyChannelMessageEditedEventRequestSchema,
  ReplaceChannelUnreadCountsRequestSchema,
  ReplaceChannelPodsRequestSchema,
  ReplaceChannelMembersRequestSchema,
  RemoveChannelMemberRequestSchema,
} from '@proto/channel_state/v1/mutations_pb'
import {
  ApplySessionRequestSchema,
  SetCurrentOrgRequestSchema,
  SetOrganizationsRequestSchema,
} from '@proto/auth_state/v1/auth_state_pb'
import { createAcpManager } from './wasm-mock-acp'

const h = vi.hoisted(() => {
  const mkStore = () => ({ v: '' as string });
  const pod = { pods: '[]', current: '' };
  const runner = { list: '[]', available: '[]', current: '' };
  const channel = {
    list: '[]', current: null as bigint | null,
    msgs: new Map<string, { json: string; hasMore: boolean }>(),
    unread: new Map<string, number>(),
    pods: new Map<string, string>(),
    members: new Map<string, string>(),
  };
  const ticket = { list: '[]', labels: '[]', boardCols: '[]', current: '' };
  const mesh = { topo: '', selected: undefined as string | undefined };
  const loop = { list: '[]', current: '' };
  const gitProvider = mkStore();
  const repo = { list: '[]', current: '', branches: '[]' };
  const autopilot = {
    controllers: '[]', current: '', iterations: new Map<string, string>(),
    thinkings: new Map<string, string>(), thinkingHistory: new Map<string, string>(),
  };

  function reset() {
    pod.pods = '[]'; pod.current = '';
    runner.list = '[]'; runner.available = '[]'; runner.current = '';
    channel.list = '[]'; channel.current = null;
    channel.msgs.clear(); channel.unread.clear();
    channel.pods.clear(); channel.members.clear();
    ticket.list = '[]'; ticket.labels = '[]'; ticket.boardCols = '[]'; ticket.current = '';
    mesh.topo = ''; mesh.selected = undefined;
    loop.list = '[]'; loop.current = '';
    gitProvider.v = '';
    repo.list = '[]'; repo.current = ''; repo.branches = '[]';
    autopilot.controllers = '[]'; autopilot.current = '';
    autopilot.iterations.clear(); autopilot.thinkings.clear(); autopilot.thinkingHistory.clear();
  }

  return { pod, runner, channel, ticket, mesh, loop, gitProvider, repo, autopilot, reset };
})

const acpMgr = createAcpManager()

// Mock WASM Core
vi.mock('@/lib/wasm-core', () => {
  const fn = vi.fn

  const mockClient = {
    get: fn().mockResolvedValue('{}'),
    post: fn().mockResolvedValue('{}'),
    put: fn().mockResolvedValue('{}'),
    delete: fn().mockResolvedValue('{}'),
    patch: fn().mockResolvedValue('{}'),
    org_path: fn((p: string) => `/api/v1/orgs/test-org${p}`),
  }

  const authBox: { user: unknown; current_org: unknown; organizations: unknown[] } = {
    user: null, current_org: null, organizations: [],
  };
  // Mirror Rust auth_types::Organization: drop empty string fields so test
  // expectations matching the wasm path's serde-Default skip semantics pass.
  function orgProtoToDto(o: { id: bigint; name: string; slug: string; role?: string;
    logoUrl?: string; subscriptionPlan: string; subscriptionStatus: string }): Record<string, unknown> {
    const dto: Record<string, unknown> = {
      id: Number(o.id),
      name: o.name,
      slug: o.slug,
    };
    if (o.role !== undefined) dto.role = o.role;
    if (o.logoUrl !== undefined) dto.logo_url = o.logoUrl;
    if (o.subscriptionPlan) dto.subscription_plan = o.subscriptionPlan;
    if (o.subscriptionStatus) dto.subscription_status = o.subscriptionStatus;
    return dto;
  }
  const mockAuth = {
    login: fn().mockResolvedValue('{"token":"t","refresh_token":"r","user":{"id":1,"email":"test@test.com","username":"test"}}'),
    logout: fn().mockResolvedValue(undefined),
    refresh_token: fn().mockResolvedValue('{"token":"t2","refresh_token":"r2"}'),
    fetch_organizations: fn().mockResolvedValue('[]'),
    switch_org: fn(),
    is_authenticated: fn(() => authBox.user !== null),
    get_token: fn(),
    get_current_user_json: fn(() => authBox.user ? JSON.stringify(authBox.user) : null),
    get_current_org_json: fn(() => authBox.current_org ? JSON.stringify(authBox.current_org) : null),
    get_organizations_json: fn(() => JSON.stringify(authBox.organizations)),
    apply_session: fn((reqBytes: Uint8Array) => {
      try {
        const req = fromBinary(ApplySessionRequestSchema, reqBytes);
        if (!req.user) { authBox.user = null; return; }
        const u: Record<string, unknown> = {
          id: Number(req.user.id),
          email: req.user.email,
          username: req.user.username,
        };
        if (req.user.name !== undefined) u.name = req.user.name;
        if (req.user.avatarUrl !== undefined) u.avatar_url = req.user.avatarUrl;
        authBox.user = u;
      } catch { /* noop */ }
    }),
    set_organizations: fn((reqBytes: Uint8Array) => {
      try {
        const req = fromBinary(SetOrganizationsRequestSchema, reqBytes);
        authBox.organizations = (req.items ?? []).map((o) => orgProtoToDto(o));
        if (authBox.current_org == null && authBox.organizations.length > 0) {
          authBox.current_org = authBox.organizations[0];
        }
      } catch { /* noop */ }
    }),
    set_current_org: fn((reqBytes: Uint8Array) => {
      try {
        const req = fromBinary(SetCurrentOrgRequestSchema, reqBytes);
        authBox.current_org = req.org ? orgProtoToDto(req.org) : null;
      } catch { /* noop */ }
    }),
    clear_session: fn(() => {
      authBox.user = null; authBox.current_org = null; authBox.organizations = [];
    }),
    _reset: () => { authBox.user = null; authBox.current_org = null; authBox.organizations = []; },
  }

  // Mirrors the real `WasmPodState` (see clients/core/crates/wasm/src/state_pod.rs).
  // Every mutator that the production renderer hits is on this object,
  // accepting proto-encoded `Uint8Array` bytes. The body stays opaque to the
  // mock — tests that care about cache contents override
  // `pods_json`/`get_pod_json` returns directly via vi.mocked. Adding stale
  // methods here (`set_pods`, multi-arg `update_pod_status`, etc.) regressed
  // the test surface: production code stopped calling them but the mock
  // kept accepting them, hiding wasm-binding drift from vitest.
  const podState = {
    pods_json: fn(() => h.pod.pods),
    current_pod_json: fn(() => h.pod.current || undefined),
    get_pod_json: fn((key: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string }[]
      const p = list.find((x) => x.pod_key === key)
      return p ? JSON.stringify(p) : undefined
    }),
    // Proto-bytes mutators. Body is opaque (decoding would require pulling
    // the proto schemas into the mock); the in-memory pods list is left
    // untouched so tests that assert "the bridge was called" still pass.
    // Tests that assert cache contents post-mutation should override
    // `pods_json` / `get_pod_json` directly.
    insert_created_pod: fn((_bytes: Uint8Array) => undefined),
    patch_pod_perpetual: fn((_bytes: Uint8Array) => undefined),
    apply_pod_status_event: fn((_bytes: Uint8Array) => undefined),
    apply_pod_title_event: fn((_bytes: Uint8Array) => undefined),
    apply_pod_alias_event: fn((_bytes: Uint8Array) => undefined),
    apply_agent_status_event: fn((_bytes: Uint8Array) => undefined),
    replace_cached_pods: fn((_bytes: Uint8Array) => undefined),
    append_cached_pods: fn((_bytes: Uint8Array) => undefined),
    mark_pod_terminated: fn((_bytes: Uint8Array) => undefined),
    remove_pod: fn((key: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string }[]
      h.pod.pods = JSON.stringify(list.filter((x) => x.pod_key !== key))
    }),
    update_init_progress: fn(),
    clear_init_progress: fn(),
  }

  // Separate service mock — Connect-RPC binary lane only. Real wasm has a
  // WasmPodService that's distinct from WasmPodState; merging them in the
  // mock previously masked production splits between cache mutation and
  // network fetch.
  const podService = {
    list_pods_connect: fn().mockResolvedValue(new Uint8Array()),
    get_pod_connect: fn().mockResolvedValue(new Uint8Array()),
    create_pod_connect: fn().mockResolvedValue(new Uint8Array()),
    terminate_pod_connect: fn().mockResolvedValue(new Uint8Array()),
    update_pod_alias_connect: fn().mockResolvedValue(new Uint8Array()),
    update_pod_perpetual_connect: fn().mockResolvedValue(new Uint8Array()),
    get_pod_connection_connect: fn().mockResolvedValue(new Uint8Array()),
    send_pod_prompt_connect: fn().mockResolvedValue(new Uint8Array()),
    list_pods_by_ticket_connect: fn().mockResolvedValue(new Uint8Array()),
  }

  const runnerState = {
    set_runners: fn((j: string) => { h.runner.list = j }),
    runners_json: fn(() => h.runner.list),
    set_available_runners: fn((j: string) => { h.runner.available = j }),
    available_runners_json: fn(() => h.runner.available),
    set_current_runner: fn((j: string) => { h.runner.current = j }),
    current_runner_json: fn(() => h.runner.current || undefined),
    update_runner_local: fn((id: number, json: string) => {
      const updated = JSON.parse(json) as { id: number };
      const arr = JSON.parse(h.runner.list) as { id: number }[];
      const idx = arr.findIndex((x) => x.id === id);
      if (idx >= 0) arr[idx] = updated;
      h.runner.list = JSON.stringify(arr);
    }),
    apply_runner_status_event: fn((_bytes: Uint8Array) => {
      // No-op default; per-test overrides simulate state mutation when needed.
    }),
    remove_runner_local: fn((id: bigint) => {
      for (const field of ['list', 'available'] as const) {
        const arr = JSON.parse(h.runner[field]) as { id: number }[]
        h.runner[field] = JSON.stringify(arr.filter((x) => x.id !== Number(id)))
      }
    }),
    // Connect-RPC binary lane (proto.runner_api.v1.RunnerService).
    // Each method takes a binary Uint8Array, returns an empty
    // proto-encoded response (= zero bytes = default fields). Sufficient
    // for unit-test smoke coverage; integration paths are exercised by
    // e2e Playwright suites against a live backend.
    listRunnersConnect: fn().mockResolvedValue(new Uint8Array()),
    listAvailableRunnersConnect: fn().mockResolvedValue(new Uint8Array()),
    getRunnerConnect: fn().mockResolvedValue(new Uint8Array()),
    updateRunnerConnect: fn().mockResolvedValue(new Uint8Array()),
    deleteRunnerConnect: fn().mockResolvedValue(new Uint8Array()),
    upgradeRunnerConnect: fn().mockResolvedValue(new Uint8Array()),
    requestLogUploadConnect: fn().mockResolvedValue(new Uint8Array()),
    listRunnerLogsConnect: fn().mockResolvedValue(new Uint8Array()),
    querySandboxesConnect: fn().mockResolvedValue(new Uint8Array()),
    createRunnerTokenConnect: fn().mockResolvedValue(new Uint8Array()),
    listRunnerTokensConnect: fn().mockResolvedValue(new Uint8Array()),
    deleteRunnerTokenConnect: fn().mockResolvedValue(new Uint8Array()),
  }

  const cKey = (id: bigint | number) => String(id)
  const channelState = {
    set_channels: fn((j: string) => { h.channel.list = j }),
    channels_json: fn(() => h.channel.list),
    set_current_channel: fn((id: bigint | null) => { h.channel.current = id }),
    current_channel_json: fn(() => {
      if (h.channel.current === null) return undefined
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const ch = list.find((c) => c.id === Number(h.channel.current))
      return ch ? JSON.stringify(ch) : undefined
    }),
    get_channel_json: fn((id: bigint) => {
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const ch = list.find((c) => c.id === Number(id))
      return ch ? JSON.stringify(ch) : undefined
    }),
    add_channel: fn((json: string) => {
      const ch = JSON.parse(json) as { id: number }
      const list = JSON.parse(h.channel.list) as { id: number }[]
      if (!list.some((c) => c.id === ch.id)) {
        list.unshift(ch)
        h.channel.list = JSON.stringify(list)
      }
    }),
    update_channel: fn((id: bigint, json: string) => {
      const ch = JSON.parse(json)
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const idx = list.findIndex((c) => c.id === Number(id))
      if (idx >= 0) { list[idx] = ch; h.channel.list = JSON.stringify(list) }
    }),
    remove_channel: fn((id: bigint) => {
      const list = JSON.parse(h.channel.list) as { id: number }[]
      h.channel.list = JSON.stringify(list.filter((c) => c.id !== Number(id)))
    }),
    filter_channels_json: fn((query: string, includeArchived: boolean) => {
      const list = JSON.parse(h.channel.list) as { id: number; name: string; is_archived?: boolean; description?: string }[]
      const q = query.toLowerCase()
      return JSON.stringify(list.filter((c) => {
        if (!includeArchived && c.is_archived) return false
        if (!q) return true
        return c.name.toLowerCase().includes(q) || (c.description || '').toLowerCase().includes(q)
      }))
    }),
    select_channel: fn((id?: bigint) => {
      if (id === undefined) { h.channel.current = null; return undefined }
      h.channel.current = id
      h.channel.unread.delete(cKey(id))
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const ch = list.find((c) => c.id === Number(id))
      return ch ? JSON.stringify(ch) : undefined
    }),
    set_current_user: fn(),
    set_current_user_id: fn(),
    set_messages: fn((chId: bigint, json: string, hasMore: boolean) => {
      h.channel.msgs.set(cKey(chId), { json, hasMore })
    }),
    get_messages_json: fn((chId: bigint) => {
      const entry = h.channel.msgs.get(cKey(chId))
      if (!entry) return undefined
      return JSON.stringify({ messages: JSON.parse(entry.json), has_more: entry.hasMore })
    }),
    prepend_messages: fn((chId: bigint, json: string, hasMore: boolean) => {
      const k = cKey(chId)
      const entry = h.channel.msgs.get(k)
      const existing = entry ? JSON.parse(entry.json) as { id: number }[] : []
      const newMsgs = JSON.parse(json) as { id: number }[]
      const existingIds = new Set(existing.map((m) => m.id))
      const deduped = newMsgs.filter((m) => !existingIds.has(m.id))
      const merged = [...deduped, ...existing]
      merged.sort((a, b) => a.id - b.id)
      h.channel.msgs.set(k, { json: JSON.stringify(merged), hasMore })
    }),
    add_message: fn((chId: bigint, json: string) => {
      const k = cKey(chId)
      const entry = h.channel.msgs.get(k)
      const msgs = entry ? JSON.parse(entry.json) as { id: number }[] : []
      const msg = JSON.parse(json) as { id: number }
      if (!msgs.some((m) => m.id === msg.id)) {
        msgs.push(msg)
        h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry?.hasMore ?? false })
      }
    }),
    on_new_message: fn((json: string) => {
      const msg = JSON.parse(json) as { id: number; channel_id: number }
      const k = cKey(msg.channel_id)
      const entry = h.channel.msgs.get(k)
      const msgs = entry ? JSON.parse(entry.json) as { id: number }[] : []
      if (!msgs.some((m) => m.id === msg.id)) {
        msgs.push(msg)
        h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry?.hasMore ?? false })
        return true
      }
      return false
    }),
    update_message: fn((chId: bigint, json: string) => {
      const k = cKey(chId); const entry = h.channel.msgs.get(k); if (!entry) return
      const msg = JSON.parse(json); const msgs = JSON.parse(entry.json) as { id: number }[]
      const idx = msgs.findIndex((m) => m.id === msg.id)
      if (idx >= 0) msgs[idx] = { ...msgs[idx], ...msg }
      h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry.hasMore })
    }),
    remove_message: fn((chId: bigint, msgId: bigint) => {
      const k = cKey(chId); const entry = h.channel.msgs.get(k); if (!entry) return
      const msgs = JSON.parse(entry.json) as { id: number }[]
      h.channel.msgs.set(k, { json: JSON.stringify(msgs.filter((m) => m.id !== Number(msgId))), hasMore: entry.hasMore })
    }),
    set_unread_counts: fn((json: string) => {
      const counts = JSON.parse(json) as Record<string, number>
      h.channel.unread.clear()
      for (const [k, v] of Object.entries(counts)) h.channel.unread.set(k, v)
    }),
    get_unread_count: fn((chId: bigint) => h.channel.unread.get(cKey(chId)) || 0),
    increment_unread: fn((chId: bigint) => {
      const k = cKey(chId); h.channel.unread.set(k, (h.channel.unread.get(k) || 0) + 1)
    }),
    clear_channel_unread: fn((chId: bigint) => { h.channel.unread.delete(cKey(chId)) }),
    unread_counts_json: fn(() => {
      const obj: Record<string, number> = {}
      for (const [k, v] of h.channel.unread.entries()) { if (v > 0) obj[k] = v }
      return JSON.stringify(obj)
    }),
    increment_mention: fn(),
    clear_channel_mentions: fn(),
    get_mention_count: fn(() => 0),
    total_mention_count: fn(() => 0),
    set_mention_counts: fn(),
    mention_counts_json: fn(() => '{}'),
    sorted_channel_ids_json: fn(() => '[]'),
    total_unread_count: fn(() => {
      let total = 0; for (const v of h.channel.unread.values()) total += v; return total
    }),
    get_last_message_json: fn(() => undefined),
    set_last_message: fn(),
    // Service async methods (API calls via WASM)
    fetch_channels: fn().mockResolvedValue(JSON.stringify({ channels: [] })),
    fetch_channel: fn().mockResolvedValue('{}'),
    create_channel: fn().mockResolvedValue('{}'),
    archive_channel: fn().mockResolvedValue(undefined),
    unarchive_channel: fn().mockResolvedValue(undefined),
    join_channel: fn().mockResolvedValue('{}'),
    leave_channel: fn().mockResolvedValue('{}'),
    fetch_messages: fn().mockResolvedValue(JSON.stringify({ messages: [], has_more: false })),
    send_message: fn().mockResolvedValue('{}'),
    edit_message: fn().mockResolvedValue('{}'),
    delete_message: fn().mockResolvedValue(undefined),
    fetch_unread_counts: fn().mockResolvedValue('{}'),
    mark_read: fn().mockResolvedValue(undefined),
    mute_channel: fn().mockResolvedValue(undefined),
    fetch_channel_members: fn().mockResolvedValue('{"members":[],"total":0}'),
    invite_channel_members: fn().mockResolvedValue(undefined),
    channel_members_json: fn((id: bigint) => h.channel.members.get(String(id)) ?? '[]'),
    get_channel_pods: fn().mockResolvedValue('{"pods":[]}'),
    channel_pods_json: fn((id: bigint) => h.channel.pods.get(String(id)) ?? '[]'),
    update_message_local: fn((chId: bigint, json: string) => {
      const k = cKey(chId); const entry = h.channel.msgs.get(k); if (!entry) return
      const msg = JSON.parse(json); const msgs = JSON.parse(entry.json) as { id: number }[]
      const idx = msgs.findIndex((m) => m.id === msg.id)
      if (idx >= 0) msgs[idx] = { ...msgs[idx], ...msg }
      h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry.hasMore })
    }),
    remove_message_local: fn((chId: bigint, msgId: bigint) => {
      const k = cKey(chId); const entry = h.channel.msgs.get(k); if (!entry) return
      const msgs = JSON.parse(entry.json) as { id: number }[]
      h.channel.msgs.set(k, { json: JSON.stringify(msgs.filter((m) => m.id !== Number(msgId))), hasMore: entry.hasMore })
    }),
    add_channel_local: fn((json: string) => {
      const ch = JSON.parse(json) as { id: number }
      const list = JSON.parse(h.channel.list) as { id: number }[]
      if (!list.some((c) => c.id === ch.id)) {
        list.unshift(ch)
        h.channel.list = JSON.stringify(list)
      }
    }),
    remove_channel_local: fn((id: bigint) => {
      const list = JSON.parse(h.channel.list) as { id: number }[]
      h.channel.list = JSON.stringify(list.filter((c) => c.id !== Number(id)))
    }),
    // Proto-bytes mutators (channel store production path). The mocks here
    // decode the request via @bufbuild/protobuf and apply the same effect
    // as the legacy JSON helpers above, so behavioural tests don't change.
    replace_cached_channels: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ReplaceCachedChannelsRequestSchema, bytes)
        h.channel.list = JSON.stringify(req.channels.map((c) => decodeProtoChannel(c)))
      } catch { h.channel.list = '[]' }
    }),
    insert_channel: fn((bytes: Uint8Array) => {
      try {
        const { channel: c } = fromBinary(InsertChannelRequestSchema, bytes)
        if (!c) return
        const channel = decodeProtoChannel(c)
        const list = JSON.parse(h.channel.list) as { id: number }[]
        const idx = list.findIndex((x) => x.id === channel.id)
        if (idx >= 0) list[idx] = { ...list[idx], ...channel }
        else list.unshift(channel)
        h.channel.list = JSON.stringify(list)
      } catch { /* noop */ }
    }),
    patch_channel_member_count: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(PatchChannelMemberCountRequestSchema, bytes)
        const list = JSON.parse(h.channel.list) as { id: number; member_count: number }[]
        const ch = list.find((x) => x.id === Number(req.channelId))
        if (ch) ch.member_count = Math.max(0, (ch.member_count || 0) + req.delta)
        h.channel.list = JSON.stringify(list)
      } catch { /* noop */ }
    }),
    replace_cached_channel_messages: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ReplaceCachedChannelMessagesRequestSchema, bytes)
        const msgs = req.messages.map(decodeProtoMessage)
        h.channel.msgs.set(cKey(Number(req.channelId)), { json: JSON.stringify(msgs), hasMore: req.hasMore })
      } catch { /* noop */ }
    }),
    prepend_cached_channel_messages: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(PrependCachedChannelMessagesRequestSchema, bytes)
        const k = cKey(Number(req.channelId))
        const entry = h.channel.msgs.get(k)
        const existing = entry ? JSON.parse(entry.json) as { id: number }[] : []
        const incoming = req.messages.map(decodeProtoMessage)
        const ids = new Set(existing.map((m) => m.id))
        const merged = [...incoming.filter((m) => !ids.has(m.id)), ...existing]
        merged.sort((a, b) => a.id - b.id)
        h.channel.msgs.set(k, { json: JSON.stringify(merged), hasMore: req.hasMore })
      } catch { /* noop */ }
    }),
    insert_channel_message: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(InsertChannelMessageRequestSchema, bytes)
        if (!req.message) return
        const k = cKey(Number(req.channelId))
        const entry = h.channel.msgs.get(k)
        const msgs = entry ? JSON.parse(entry.json) as { id: number }[] : []
        const msg = decodeProtoMessage(req.message)
        if (!msgs.some((m) => m.id === msg.id)) {
          msgs.push(msg)
          h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry?.hasMore ?? false })
        }
      } catch { /* noop */ }
    }),
    apply_incoming_channel_message: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ApplyIncomingChannelMessageRequestSchema, bytes)
        if (!req.message) return false
        const k = cKey(Number(req.channelId))
        const entry = h.channel.msgs.get(k)
        const msgs = entry ? JSON.parse(entry.json) as { id: number }[] : []
        const msg = decodeProtoMessage(req.message)
        if (!msgs.some((m) => m.id === msg.id)) {
          msgs.push(msg)
          h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry?.hasMore ?? false })
          return true
        }
        return false
      } catch { return false }
    }),
    apply_channel_message_edited_event: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ApplyChannelMessageEditedEventRequestSchema, bytes)
        const k = cKey(Number(req.channelId))
        const entry = h.channel.msgs.get(k); if (!entry) return
        const msgs = JSON.parse(entry.json) as { id: number; body?: string; edited_at?: string; content_json?: string; mentions_json?: string }[]
        const idx = msgs.findIndex((m) => m.id === Number(req.messageId))
        if (idx >= 0) {
          if (req.body) msgs[idx].body = req.body
          msgs[idx].edited_at = req.editedAt
          if (req.content !== undefined) msgs[idx].content_json = req.content
          if (Object.keys(req.mentions).length > 0) msgs[idx].mentions_json = JSON.stringify(req.mentions)
        }
        h.channel.msgs.set(k, { json: JSON.stringify(msgs), hasMore: entry.hasMore })
      } catch { /* noop */ }
    }),
    replace_channel_unread_counts: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ReplaceChannelUnreadCountsRequestSchema, bytes)
        h.channel.unread.clear()
        for (const [k, v] of Object.entries(req.counts)) h.channel.unread.set(k, v as number)
      } catch { /* noop */ }
    }),
    replace_channel_pods: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ReplaceChannelPodsRequestSchema, bytes)
        const pods = req.pods.map((p) => ({
          id: Number(p.id), pod_key: p.podKey, alias: p.alias,
          status: p.status, agent_status: p.agentStatus,
        }))
        h.channel.pods.set(String(req.channelId), JSON.stringify(pods))
      } catch { /* noop */ }
    }),
    replace_channel_members: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(ReplaceChannelMembersRequestSchema, bytes)
        const members = req.members.map((m) => ({
          channel_id: Number(m.channelId), user_id: Number(m.userId),
          role: m.role, is_muted: m.isMuted, joined_at: m.joinedAt,
        }))
        h.channel.members.set(String(req.channelId), JSON.stringify(members))
      } catch { /* noop */ }
    }),
    remove_channel_member: fn((bytes: Uint8Array) => {
      try {
        const req = fromBinary(RemoveChannelMemberRequestSchema, bytes)
        const key = String(req.channelId)
        const existing = JSON.parse(h.channel.members.get(key) ?? "[]") as Array<{ user_id: number }>
        h.channel.members.set(key, JSON.stringify(existing.filter((m) => m.user_id !== Number(req.userId))))
      } catch { /* noop */ }
    }),
  }

  // Helper: proto.channel_state.v1.Channel → web Channel (snake_case). Only
  // emits fields that the proto carried — keeps deep-equality assertions
  // against `mockChannel` stable across the proto round-trip.
  function decodeProtoChannel(c: {
    id: bigint; organizationId?: bigint; name: string; description?: string;
    document?: string; visibility?: string; isArchived: boolean; isMember: boolean;
    memberCount?: bigint; agentCount?: bigint; createdAt?: string; updatedAt?: string;
  }): { id: number; name: string; is_archived: boolean; is_member: boolean; member_count: number; [k: string]: unknown } {
    const out: Record<string, unknown> = {
      id: Number(c.id), name: c.name,
      is_archived: c.isArchived, is_member: c.isMember,
      member_count: c.memberCount !== undefined ? Number(c.memberCount) : 0,
    }
    if (c.organizationId !== undefined) out.organization_id = Number(c.organizationId)
    if (c.description !== undefined) out.description = c.description
    if (c.document !== undefined) out.document = c.document
    if (c.visibility !== undefined) out.visibility = c.visibility
    if (c.agentCount !== undefined) out.agent_count = Number(c.agentCount)
    if (c.createdAt !== undefined) out.created_at = c.createdAt
    if (c.updatedAt !== undefined) out.updated_at = c.updatedAt
    return out as { id: number; name: string; is_archived: boolean; is_member: boolean; member_count: number; [k: string]: unknown }
  }

  // Helper: proto.channel_state.v1.ChannelMessage (camelCase) → web wasm
  // projection (snake_case). Mirrors the renderer projection so cached
  // messages keep the same shape callers expect.
  function decodeProtoMessage(m: {
    id: bigint; channelId: bigint; body?: string; senderPod?: string;
    senderUserId?: bigint; messageType?: string; contentJson?: string;
    mentionsJson?: string; replyTo?: bigint; editedAt?: string; createdAt?: string;
    isDeleted?: boolean;
    senderUser?: { id: bigint; username: string; name?: string; avatarUrl?: string };
    senderPodInfo?: { podKey: string; alias?: string };
  }): { id: number; channel_id: number; body?: string; [k: string]: unknown } {
    return {
      id: Number(m.id), channel_id: Number(m.channelId),
      body: m.body, sender_pod: m.senderPod,
      sender_user_id: m.senderUserId !== undefined ? Number(m.senderUserId) : undefined,
      message_type: m.messageType,
      content_json: m.contentJson, mentions_json: m.mentionsJson,
      reply_to: m.replyTo !== undefined ? Number(m.replyTo) : undefined,
      edited_at: m.editedAt, created_at: m.createdAt, is_deleted: m.isDeleted,
      sender_user: m.senderUser ? {
        id: Number(m.senderUser.id), username: m.senderUser.username,
        name: m.senderUser.name, avatar_url: m.senderUser.avatarUrl,
      } : undefined,
      sender_pod_info: m.senderPodInfo ? {
        pod_key: m.senderPodInfo.podKey, alias: m.senderPodInfo.alias,
      } : undefined,
    }
  }

  // ticketState now mirrors WasmTicketState (proto bytes mutators + JSON
  // reads). Tests that need to observe cache state can override these via
  // vi.mocked(...).mockImplementation; defaults are no-ops so the bridge
  // doesn't crash when store actions fire.
  const ticketState = {
    tickets_json: fn(() => h.ticket.list),
    board_columns_json: fn(() => h.ticket.boardCols),
    labels_json: fn(() => h.ticket.labels),
    current_ticket_json: fn(() => h.ticket.current || undefined),
    apply_ticket_status_event: fn((_b: Uint8Array) => undefined),
    apply_ticket_deleted_event: fn((_b: Uint8Array) => undefined),
    replace_cached_tickets: fn((_b: Uint8Array) => undefined),
    insert_created_ticket: fn((_b: Uint8Array) => undefined),
    patch_cached_ticket: fn((_b: Uint8Array) => undefined),
    replace_board_columns: fn((_b: Uint8Array) => undefined),
    append_board_column_tickets: fn((_b: Uint8Array) => undefined),
    set_current_ticket: fn((_b: Uint8Array) => undefined),
    replace_cached_labels: fn((_b: Uint8Array) => undefined),
    insert_created_label: fn((_b: Uint8Array) => undefined),
    remove_cached_label: fn((_b: Uint8Array) => undefined),
    filter_tickets: fn((_b: Uint8Array) => new Uint8Array()),
  }

  // ticketService retains only the ticket-pods cache + Connect-RPC bridge.
  // State mutation moved to ticketState above per the proto-state contract.
  const ticketService = {
    get_ticket_pods: fn().mockResolvedValue(JSON.stringify({ pods: [] })),
    ticket_pods_json: fn(() => '[]'),
    // Connect-RPC binary wire — every adapter call resolves to an empty
    // Uint8Array (decodes to the proto default = empty list / no-op).
    list_tickets_connect: fn().mockResolvedValue(new Uint8Array()),
    get_ticket_connect: fn().mockResolvedValue(new Uint8Array()),
    create_ticket_connect: fn().mockResolvedValue(new Uint8Array()),
    update_ticket_connect: fn().mockResolvedValue(new Uint8Array()),
    delete_ticket_connect: fn().mockResolvedValue(new Uint8Array()),
    update_ticket_status_connect: fn().mockResolvedValue(new Uint8Array()),
    get_active_tickets_connect: fn().mockResolvedValue(new Uint8Array()),
    get_board_connect: fn().mockResolvedValue(new Uint8Array()),
    get_sub_tickets_connect: fn().mockResolvedValue(new Uint8Array()),
    add_assignee_connect: fn().mockResolvedValue(new Uint8Array()),
    remove_assignee_connect: fn().mockResolvedValue(new Uint8Array()),
    list_labels_connect: fn().mockResolvedValue(new Uint8Array()),
    create_label_connect: fn().mockResolvedValue(new Uint8Array()),
    update_label_connect: fn().mockResolvedValue(new Uint8Array()),
    delete_label_connect: fn().mockResolvedValue(new Uint8Array()),
    add_label_connect: fn().mockResolvedValue(new Uint8Array()),
    remove_label_connect: fn().mockResolvedValue(new Uint8Array()),
  }

  const meshState = {
    topology_json: fn(() => h.mesh.topo || undefined),
    clear_topology: fn(() => { h.mesh.topo = '' }),
    select_node: fn((key?: string) => { h.mesh.selected = key }),
    selected_node: fn(() => h.mesh.selected),
    get_node_json: fn(),
    get_edges_for_node_json: fn(() => '[]'),
    get_channels_for_node_json: fn(() => '[]'),
    get_nodes_by_runner_json: fn(() => '[]'),
    get_active_nodes_json: fn(() => '[]'),
    get_runner_info_json: fn(),
    fetch_topology: fn().mockResolvedValue(JSON.stringify({ nodes: [], edges: [], channels: [], runners: [] })),
    // Proto-bytes mutator (mirror state_mesh.rs).
    replace_topology: fn((_b: Uint8Array) => undefined),
    // Connect-RPC bridge — empty Uint8Array decodes to default-valued proto.
    getMeshTopologyConnect: fn().mockResolvedValue(new Uint8Array()),
    getTicketPodsConnect: fn().mockResolvedValue(new Uint8Array()),
    batchGetTicketPodsConnect: fn().mockResolvedValue(new Uint8Array()),
    createPodForTicketConnect: fn().mockResolvedValue(new Uint8Array()),
  }

  const loopState = {
    loops_json: fn(() => h.loop.list),
    current_loop_json: fn(() => h.loop.current || undefined),
    runs_json: fn(() => '[]'),
    get_loop_by_slug_json: fn((slug: string) => {
      const arr = JSON.parse(h.loop.list) as { slug: string }[]
      const l = arr.find((x) => x.slug === slug)
      return l ? JSON.stringify(l) : undefined
    }),
    // Proto-state mutations (binary wire) — TS store uses these.
    replace_cached_loops: fn(), set_current_loop: fn(), clear_current_loop: fn(),
    patch_loop_from_action: fn(), insert_loop_run: fn(),
    replace_cached_runs: fn(), append_cached_runs: fn(),
    patch_loop_run_status: fn(), clear_loop_runs: fn(),
    // Connect-RPC binary lane (proto.loop.v1.LoopService).
    listLoopsConnect: fn().mockResolvedValue(new Uint8Array()),
    getLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    createLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    updateLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    deleteLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    enableLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    disableLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    triggerLoopConnect: fn().mockResolvedValue(new Uint8Array()),
    listRunsConnect: fn().mockResolvedValue(new Uint8Array()),
    cancelRunConnect: fn().mockResolvedValue(new Uint8Array()),
  }

  const repoState = {
    set_repositories: fn(), repositories_json: fn(() => h.repo.list),
    set_current_repo: fn(), current_repo_json: fn(() => h.repo.current || undefined),
    add_repository: fn(), update_repository: fn(), remove_repository: fn(),
    set_branches: fn(), branches_json: fn(() => h.repo.branches),
    // Proto-bytes mutators (mirror state_repo.rs).
    replace_cached_repositories: fn((_b: Uint8Array) => undefined),
    set_current_repo_proto: fn((_b: Uint8Array) => undefined),
    replace_branches: fn((_b: Uint8Array) => undefined),
    insert_repository: fn((_b: Uint8Array) => undefined),
    patch_repository: fn((_b: Uint8Array) => undefined),
  }

  const autopilotState = {
    set_controllers: fn(), controllers_json: fn(() => h.autopilot.controllers),
    set_current_controller: fn(), current_controller_json: fn(() => h.autopilot.current || undefined),
    add_controller: fn(), update_controller: fn(), remove_controller: fn(),
    add_iteration: fn(), set_iterations: fn(),
    get_iterations_json: fn(), update_thinking: fn(),
    get_thinking_json: fn(), get_thinking_history_json: fn(),
    get_controller_by_pod_key_json: fn(),
    fetch_controllers: fn().mockResolvedValue('[]'),
    fetch_controller: fn().mockResolvedValue('{}'),
    create_controller: fn().mockResolvedValue('{}'),
    pause_controller: fn().mockResolvedValue(undefined),
    resume_controller: fn().mockResolvedValue(undefined),
    stop_controller: fn().mockResolvedValue(undefined),
    approve_controller: fn().mockResolvedValue(undefined),
    takeover_controller: fn().mockResolvedValue(undefined),
    handback_controller: fn().mockResolvedValue(undefined),
    fetch_iterations: fn().mockResolvedValue('[]'),
  }

  return {
    initWasmCore: fn().mockResolvedValue(undefined),
    getApiClient: fn(() => mockClient),
    getAuthManager: fn(() => mockAuth),
    getPodState: fn(() => podState),
    getPodService: fn(() => podService),
    getTicketService: fn(() => ticketService),
    getChannelService: fn(() => channelState),
    getRunnerService: fn(() => runnerState),
    getTicketState: fn(() => ticketState),
    getChannelState: fn(() => channelState),
    getRunnerState: fn(() => runnerState),
    getLoopState: fn(() => loopState),
    getLoopService: fn(() => loopState),
    getMeshState: fn(() => meshState),
    getMeshService: fn(() => meshState),
    getAcpManager: fn(() => acpMgr),
    getRepoState: fn(() => repoState),
    getAutopilotState: fn(() => autopilotState),
    getAutopilotService: fn(() => autopilotState),
    getRelayManager: fn(() => ({
      set_token: fn(), clear: fn(),
    })),
    getBillingService: fn(() => ({
      get_overview: fn().mockResolvedValue('{}'),
      get_subscription: fn().mockResolvedValue('{}'),
      list_plans: fn().mockResolvedValue('[]'),
      create_subscription: fn().mockResolvedValue('{}'),
      update_subscription: fn().mockResolvedValue('{}'),
      cancel_subscription: fn().mockResolvedValue('{}'),
      get_usage: fn().mockResolvedValue('{}'),
      check_quota: fn().mockResolvedValue('{}'),
      create_checkout: fn().mockResolvedValue('{}'),
      get_checkout_status: fn().mockResolvedValue('{}'),
      request_cancel: fn().mockResolvedValue('{}'),
      reactivate: fn().mockResolvedValue('{}'),
      upgrade: fn().mockResolvedValue('{}'),
      change_cycle: fn().mockResolvedValue('{}'),
      update_auto_renew: fn().mockResolvedValue('{}'),
      get_seat_usage: fn().mockResolvedValue('{}'),
      purchase_seats: fn().mockResolvedValue('{}'),
      list_invoices: fn().mockResolvedValue('[]'),
      get_customer_portal: fn().mockResolvedValue('{}'),
      get_deployment_info: fn().mockResolvedValue('{}'),
      get_public_pricing: fn().mockResolvedValue('{}'),
      get_public_deployment_info: fn().mockResolvedValue('{}'),
      // Connect-RPC (binary wire) — return empty Uint8Array for tests
      get_overview_connect: fn().mockResolvedValue(new Uint8Array()),
      list_plans_connect: fn().mockResolvedValue(new Uint8Array()),
      get_subscription_connect: fn().mockResolvedValue(new Uint8Array()),
      create_subscription_connect: fn().mockResolvedValue(new Uint8Array()),
      update_subscription_connect: fn().mockResolvedValue(new Uint8Array()),
      cancel_subscription_connect: fn().mockResolvedValue(new Uint8Array()),
      request_cancel_connect: fn().mockResolvedValue(new Uint8Array()),
      reactivate_connect: fn().mockResolvedValue(new Uint8Array()),
      upgrade_connect: fn().mockResolvedValue(new Uint8Array()),
      change_cycle_connect: fn().mockResolvedValue(new Uint8Array()),
      update_auto_renew_connect: fn().mockResolvedValue(new Uint8Array()),
      get_seat_usage_connect: fn().mockResolvedValue(new Uint8Array()),
      purchase_seats_connect: fn().mockResolvedValue(new Uint8Array()),
      list_invoices_connect: fn().mockResolvedValue(new Uint8Array()),
      create_checkout_connect: fn().mockResolvedValue(new Uint8Array()),
      get_checkout_status_connect: fn().mockResolvedValue(new Uint8Array()),
      get_deployment_info_connect: fn().mockResolvedValue(new Uint8Array()),
      get_public_pricing_connect: fn().mockResolvedValue(new Uint8Array()),
      get_public_deployment_info_connect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getRepositoryService: fn(() => ({
      list: fn().mockResolvedValue('{"repositories":[]}'),
      get: fn().mockResolvedValue('{}'),
      create: fn().mockResolvedValue('{}'),
      update: fn().mockResolvedValue('{}'),
      delete: fn().mockResolvedValue(undefined),
      list_branches: fn().mockResolvedValue('{"branches":[]}'),
      sync_branches: fn().mockResolvedValue('{"branches":[]}'),
      register_webhook: fn().mockResolvedValue(undefined),
      delete_webhook: fn().mockResolvedValue(undefined),
      get_webhook_status: fn().mockResolvedValue('{}'),
      get_webhook_secret: fn().mockResolvedValue('{}'),
      list_merge_requests: fn().mockResolvedValue('{"merge_requests":[]}'),
      // Connect-RPC binary methods. Tests that exercise Connect paths
      // override these with proto-encoded fixtures via their own mocks.
      list_repositories_connect: fn().mockResolvedValue(new Uint8Array()),
      get_repository_connect: fn().mockResolvedValue(new Uint8Array()),
      create_repository_connect: fn().mockResolvedValue(new Uint8Array()),
      update_repository_connect: fn().mockResolvedValue(new Uint8Array()),
      delete_repository_connect: fn().mockResolvedValue(new Uint8Array()),
      list_repository_branches_connect: fn().mockResolvedValue(new Uint8Array()),
      sync_repository_branches_connect: fn().mockResolvedValue(new Uint8Array()),
      list_repository_merge_requests_connect: fn().mockResolvedValue(new Uint8Array()),
      register_repository_webhook_connect: fn().mockResolvedValue(new Uint8Array()),
      delete_repository_webhook_connect: fn().mockResolvedValue(new Uint8Array()),
      get_repository_webhook_status_connect: fn().mockResolvedValue(new Uint8Array()),
      get_repository_webhook_secret_connect: fn().mockResolvedValue(new Uint8Array()),
      mark_repository_webhook_configured_connect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getExtensionService: fn(() => ({
      // SkillRegistryService — Connect-RPC (binary wire)
      listSkillRegistriesConnect: fn().mockResolvedValue(new Uint8Array()),
      createSkillRegistryConnect: fn().mockResolvedValue(new Uint8Array()),
      syncSkillRegistryConnect: fn().mockResolvedValue(new Uint8Array()),
      togglePlatformRegistryConnect: fn().mockResolvedValue(new Uint8Array()),
      deleteSkillRegistryConnect: fn().mockResolvedValue(new Uint8Array()),
      listSkillRegistryOverridesConnect: fn().mockResolvedValue(new Uint8Array()),
      // MarketService — Connect-RPC (binary wire)
      listMarketSkillsConnect: fn().mockResolvedValue(new Uint8Array()),
      listMarketMcpServersConnect: fn().mockResolvedValue(new Uint8Array()),
      // RepoSkillService — Connect-RPC (binary wire)
      listRepoSkillsConnect: fn().mockResolvedValue(new Uint8Array()),
      installSkillFromMarketConnect: fn().mockResolvedValue(new Uint8Array()),
      installSkillFromGithubConnect: fn().mockResolvedValue(new Uint8Array()),
      updateSkillConnect: fn().mockResolvedValue(new Uint8Array()),
      uninstallSkillConnect: fn().mockResolvedValue(new Uint8Array()),
      // RepoMcpService — Connect-RPC (binary wire)
      listRepoMcpServersConnect: fn().mockResolvedValue(new Uint8Array()),
      installMcpFromMarketConnect: fn().mockResolvedValue(new Uint8Array()),
      installCustomMcpServerConnect: fn().mockResolvedValue(new Uint8Array()),
      updateMcpServerConnect: fn().mockResolvedValue(new Uint8Array()),
      uninstallMcpServerConnect: fn().mockResolvedValue(new Uint8Array()),
      // Skill upload (Presign + InstallFromUploaded) — Connect-RPC
      presignSkillUploadConnect: fn().mockResolvedValue(new Uint8Array()),
      installSkillFromUploadedFileConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getInvitationService: fn(() => ({
      list: fn().mockResolvedValue('{"invitations":[]}'),
      create: fn().mockResolvedValue('{}'),
      revoke: fn().mockResolvedValue(undefined),
      resend: fn().mockResolvedValue(undefined),
      get_by_token: fn().mockResolvedValue('{}'),
      accept: fn().mockResolvedValue(undefined),
      list_pending: fn().mockResolvedValue('{"invitations":[]}'),
      // Connect-RPC (binary wire) — return empty Uint8Array for tests
      listInvitationsConnect: fn().mockResolvedValue(new Uint8Array()),
      createInvitationConnect: fn().mockResolvedValue(new Uint8Array()),
      revokeInvitationConnect: fn().mockResolvedValue(new Uint8Array()),
      resendInvitationConnect: fn().mockResolvedValue(new Uint8Array()),
      acceptInvitationConnect: fn().mockResolvedValue(new Uint8Array()),
      listPendingInvitationsConnect: fn().mockResolvedValue(new Uint8Array()),
      getInvitationByTokenConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getApiKeyService: fn(() => ({
      // Connect-RPC (binary wire) — return empty Uint8Array for tests
      listApiKeysConnect: fn().mockResolvedValue(new Uint8Array()),
      getApiKeyConnect: fn().mockResolvedValue(new Uint8Array()),
      createApiKeyConnect: fn().mockResolvedValue(new Uint8Array()),
      updateApiKeyConnect: fn().mockResolvedValue(new Uint8Array()),
      revokeApiKeyConnect: fn().mockResolvedValue(new Uint8Array()),
      deleteApiKeyConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getBindingService: fn(() => ({
      // Connect-RPC (binary wire) — return empty Uint8Array for tests
      requestBindingConnect: fn().mockResolvedValue(new Uint8Array()),
      acceptBindingConnect: fn().mockResolvedValue(new Uint8Array()),
      rejectBindingConnect: fn().mockResolvedValue(new Uint8Array()),
      unbindConnect: fn().mockResolvedValue(new Uint8Array()),
      requestScopesConnect: fn().mockResolvedValue(new Uint8Array()),
      approveScopesConnect: fn().mockResolvedValue(new Uint8Array()),
      listBindingsConnect: fn().mockResolvedValue(new Uint8Array()),
      getPendingBindingsConnect: fn().mockResolvedValue(new Uint8Array()),
      getBoundPodsConnect: fn().mockResolvedValue(new Uint8Array()),
      checkBindingConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getNotificationService: fn(() => ({
      get_preferences: fn().mockResolvedValue('{"preferences":[]}'),
      set_preference: fn().mockResolvedValue('{}'),
      listPreferencesConnect: fn().mockResolvedValue(new Uint8Array()),
      setPreferenceConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getPromoCodeService: fn(() => ({
      validatePromoCodeConnect: fn().mockResolvedValue(new Uint8Array()),
      redeemPromoCodeConnect: fn().mockResolvedValue(new Uint8Array()),
      getRedemptionHistoryConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getTokenUsageService: fn(() => ({
      get_dashboard: fn().mockResolvedValue('{}'),
    })),
    getSSOService: fn(() => ({
      discoverConnect: fn().mockResolvedValue(new Uint8Array()),
      ldapAuthConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getUserApiService: fn(() => ({
      getMeConnect: fn().mockResolvedValue(new Uint8Array()),
      updateMeConnect: fn().mockResolvedValue(new Uint8Array()),
      changePasswordConnect: fn().mockResolvedValue(new Uint8Array()),
      listIdentitiesConnect: fn().mockResolvedValue(new Uint8Array()),
      deleteIdentityConnect: fn().mockResolvedValue(new Uint8Array()),
      searchUsersConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getUserCredentialService: fn(() => ({
      list_git_credentials: fn().mockResolvedValue('{"credentials":[]}'),
      create_git_credential: fn().mockResolvedValue('{}'),
      get_git_credential: fn().mockResolvedValue('{}'),
      update_git_credential: fn().mockResolvedValue('{}'),
      delete_git_credential: fn().mockResolvedValue(undefined),
      get_default_git_credential: fn().mockResolvedValue('{}'),
      set_default_git_credential: fn().mockResolvedValue(undefined),
      clear_default_git_credential: fn().mockResolvedValue(undefined),
      list_repo_providers: fn().mockResolvedValue('{"providers":[]}'),
      create_repo_provider: fn().mockResolvedValue('{}'),
      get_repo_provider: fn().mockResolvedValue('{}'),
      update_repo_provider: fn().mockResolvedValue('{}'),
      delete_repo_provider: fn().mockResolvedValue(undefined),
      set_default_repo_provider: fn().mockResolvedValue(undefined),
      test_repo_provider: fn().mockResolvedValue(undefined),
      list_provider_repositories: fn().mockResolvedValue('{"repositories":[]}'),
    })),
    getEnvBundleService: fn(() => ({
      listEnvBundlesConnect: fn().mockResolvedValue(new Uint8Array()),
      getEnvBundleConnect: fn().mockResolvedValue(new Uint8Array()),
      createEnvBundleConnect: fn().mockResolvedValue(new Uint8Array()),
      updateEnvBundleConnect: fn().mockResolvedValue(new Uint8Array()),
      deleteEnvBundleConnect: fn().mockResolvedValue(new Uint8Array()),
      setPrimaryEnvBundleConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getOrgApiService: fn(() => ({
      list: fn().mockResolvedValue('{"organizations":[]}'),
      get: fn().mockResolvedValue('{}'),
      create: fn().mockResolvedValue('{}'),
      update: fn().mockResolvedValue('{}'),
      delete: fn().mockResolvedValue(undefined),
      list_members: fn().mockResolvedValue('{"members":[]}'),
      invite_member: fn().mockResolvedValue('{}'),
      remove_member: fn().mockResolvedValue(undefined),
      update_member_role: fn().mockResolvedValue('{}'),
      // Connect (binary) lane — mocked as empty Uint8Array; per-test
      // overrides can supply protobuf-encoded payloads as needed.
      listMyOrgsConnect: fn().mockResolvedValue(new Uint8Array()),
      createOrgConnect: fn().mockResolvedValue(new Uint8Array()),
      createPersonalOrgConnect: fn().mockResolvedValue(new Uint8Array()),
      getOrgConnect: fn().mockResolvedValue(new Uint8Array()),
      updateOrgConnect: fn().mockResolvedValue(new Uint8Array()),
      deleteOrgConnect: fn().mockResolvedValue(new Uint8Array()),
      listMembersConnect: fn().mockResolvedValue(new Uint8Array()),
      inviteMemberConnect: fn().mockResolvedValue(new Uint8Array()),
      removeMemberConnect: fn().mockResolvedValue(new Uint8Array()),
      updateMemberRoleConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getAgentService: fn(() => ({
      get_agentpod_settings: fn().mockResolvedValue('{}'),
      update_agentpod_settings: fn().mockResolvedValue('{}'),
      list_providers: fn().mockResolvedValue('{"providers":[]}'),
      create_provider: fn().mockResolvedValue('{}'),
      update_provider: fn().mockResolvedValue('{}'),
      delete_provider: fn().mockResolvedValue(undefined),
      set_default_provider: fn().mockResolvedValue(undefined),
      // Connect-RPC (binary wire) — empty Uint8Array decodes to default proto
      // messages, matching the legacy JSON `{}` semantics. Per-test mocks can
      // override with prost-encoded payloads via @bufbuild/protobuf toBinary.
      list_agents_connect: fn().mockResolvedValue(new Uint8Array()),
      get_agent_connect: fn().mockResolvedValue(new Uint8Array()),
      get_agent_config_schema_connect: fn().mockResolvedValue(new Uint8Array()),
      create_custom_agent_connect: fn().mockResolvedValue(new Uint8Array()),
      update_custom_agent_connect: fn().mockResolvedValue(new Uint8Array()),
      delete_custom_agent_connect: fn().mockResolvedValue(new Uint8Array()),
      list_user_agent_configs_connect: fn().mockResolvedValue(new Uint8Array()),
      get_user_agent_config_connect: fn().mockResolvedValue(new Uint8Array()),
      set_user_agent_config_connect: fn().mockResolvedValue(new Uint8Array()),
      delete_user_agent_config_connect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getTicketRelationsService: fn(() => ({
      // Connect-RPC (binary wire) — empty Uint8Array decodes to default
      // proto messages so call sites that don't override get sensible
      // defaults instead of TypeErrors.
      list_relations_connect: fn().mockResolvedValue(new Uint8Array()),
      create_relation_connect: fn().mockResolvedValue(new Uint8Array()),
      delete_relation_connect: fn().mockResolvedValue(new Uint8Array()),
      list_commits_connect: fn().mockResolvedValue(new Uint8Array()),
      link_commit_connect: fn().mockResolvedValue(new Uint8Array()),
      unlink_commit_connect: fn().mockResolvedValue(new Uint8Array()),
      list_merge_requests_connect: fn().mockResolvedValue(new Uint8Array()),
      list_comments_connect: fn().mockResolvedValue(new Uint8Array()),
      create_comment_connect: fn().mockResolvedValue(new Uint8Array()),
      update_comment_connect: fn().mockResolvedValue(new Uint8Array()),
      delete_comment_connect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getFileService: fn(() => ({
      presign_upload: fn().mockResolvedValue('{}'),
    })),
    getSupportTicketService: fn(() => ({
      list: fn().mockResolvedValue('{"tickets":[]}'),
      get_detail: fn().mockResolvedValue('{}'),
      get_attachment_url: fn().mockResolvedValue('{}'),
      create_ticket: fn().mockResolvedValue('{}'),
      add_message: fn().mockResolvedValue('{}'),
      // Connect-RPC (binary wire) — empty Uint8Array decodes to default
      // proto messages so call sites that don't override the mock get
      // sensible defaults instead of TypeErrors.
      listSupportTicketsConnect: fn().mockResolvedValue(new Uint8Array()),
      getSupportTicketConnect: fn().mockResolvedValue(new Uint8Array()),
      getAttachmentUrlConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    getAuthConnectService: fn(() => ({
      // Connect-RPC (binary wire) — empty Uint8Array decodes to default
      // proto messages so call sites that don't override get sensible
      // defaults instead of TypeErrors.
      loginConnect: fn().mockResolvedValue(new Uint8Array()),
      registerConnect: fn().mockResolvedValue(new Uint8Array()),
      refreshTokenConnect: fn().mockResolvedValue(new Uint8Array()),
      verifyEmailConnect: fn().mockResolvedValue(new Uint8Array()),
      resendVerificationConnect: fn().mockResolvedValue(new Uint8Array()),
      forgotPasswordConnect: fn().mockResolvedValue(new Uint8Array()),
      resetPasswordConnect: fn().mockResolvedValue(new Uint8Array()),
      oauthRedirectConnect: fn().mockResolvedValue(new Uint8Array()),
      oauthCallbackConnect: fn().mockResolvedValue(new Uint8Array()),
      logoutConnect: fn().mockResolvedValue(new Uint8Array()),
    })),
    isWasmReady: fn(() => true),
    parseWasmAny: fn((v: unknown) => v ? (typeof v === 'string' ? JSON.parse(v as string) : v) : null),
    relay_encode_input: fn((d: Uint8Array) => new Uint8Array([0x03, ...d])),
    relay_decode_message: fn((d: Uint8Array) => {
      if (d.length === 0) return { type: 0, payload: new Uint8Array(0) }
      return { type: d[0], payload: d.slice(1) }
    }),
    relay_encode_resize: fn((cols: number, rows: number) => {
      const buf = new Uint8Array(5)
      buf[0] = 0x04
      buf[1] = (cols >> 8) & 0xff; buf[2] = cols & 0xff
      buf[3] = (rows >> 8) & 0xff; buf[4] = rows & 0xff
      return buf
    }),
    relay_encode_ping: fn(() => new Uint8Array([0x05])),
    relay_encode_control: fn((d: Uint8Array) => new Uint8Array([0x07, ...d])),
    relay_encode_resync: fn(() => new Uint8Array([0x0a])),
    relay_encode_acp_command: fn((d: Uint8Array) => new Uint8Array([0x0c, ...d])),
  }
})

vi.mock('agentsmesh-wasm', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  version: vi.fn(() => '0.1.0-test'),
}))

const createLocalStorageMock = () => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => { store[key] = value },
    removeItem: (key: string) => { delete store[key] },
    clear: () => { store = {} },
    get length() { return Object.keys(store).length },
    key: (index: number) => Object.keys(store)[index] || null,
  }
}

Object.defineProperty(window, 'localStorage', { value: createLocalStorageMock(), writable: true })
Object.defineProperty(window, 'sessionStorage', { value: createLocalStorageMock(), writable: true })

global.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} } as unknown as typeof ResizeObserver
global.IntersectionObserver = class { observe() {} unobserve() {} disconnect() {} } as unknown as typeof IntersectionObserver
Element.prototype.scrollIntoView = vi.fn()
Element.prototype.hasPointerCapture = Element.prototype.hasPointerCapture ?? (() => false)
Element.prototype.setPointerCapture = Element.prototype.setPointerCapture ?? (() => {})
Element.prototype.releasePointerCapture = Element.prototype.releasePointerCapture ?? (() => {})

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false, media: query, onchange: null,
    addListener: vi.fn(), removeListener: vi.fn(),
    addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
  })),
})

afterEach(() => { h.reset(); acpMgr._reset(); vi.clearAllMocks() })

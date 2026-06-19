import '@testing-library/jest-dom'
import { vi, afterEach } from 'vitest'
import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import {
  ReplaceCachedChannelsRequestSchema,
  InsertChannelRequestSchema,
  PatchChannelMemberCountRequestSchema,
  ReplaceCachedChannelMessagesRequestSchema,
  InsertChannelMessageRequestSchema,
  ApplyIncomingChannelMessageRequestSchema,
  ApplyChannelMessageEditedEventRequestSchema,
  ReplaceChannelUnreadCountsRequestSchema,
  ReplaceChannelPodsRequestSchema,
  ReplaceChannelMembersRequestSchema,
  RemoveChannelMemberRequestSchema,
} from '@proto/channel_state/v1/mutations_pb'
import {
  ChannelSchema, ListChannelsResponseSchema, ListChannelMessagesResponseSchema,
  ListChannelMembersResponseSchema, ListChannelPodsResponseSchema,
} from '@proto/channel/v1/channel_pb'
import { MessagePreviewSchema } from '@proto/channel_state/v1/channel_state_pb'
import {
  ListPodsResponseSchema, PodSchema, PodRunnerInfoSchema, PodCreatedByInfoSchema,
} from '@proto/pod/v1/pod_pb'
import { ReplaceCachedPodsRequestSchema } from '@proto/pod_state/v1/pod_state_pb'
import {
  ListRunnersResponseSchema, ListAvailableRunnersResponseSchema, RunnerSchema, GetRunnerResponseSchema,
} from '@proto/runner_api/v1/runner_pb'
import {
  ReplaceCachedRunnersRequestSchema, ReplaceAvailableRunnersRequestSchema,
} from '@proto/runner_state/v1/runner_state_pb'
import { ListTicketsResponseSchema, ListLabelsResponseSchema, TicketSchema, BoardColumnSchema, LabelSchema } from '@proto/ticket/v1/ticket_pb'
import { ReplaceCachedTicketsRequestSchema, SetCurrentTicketRequestSchema, ReplaceBoardColumnsRequestSchema, ReplaceCachedLabelsRequestSchema } from '@proto/ticket_state/v1/ticket_state_pb'
import {
  ListLoopsResponseSchema, ListRunsResponseSchema, LoopSchema, LoopRunSchema,
} from '@proto/loop/v1/loop_pb'
import {
  ReplaceCachedLoopsRequestSchema, ReplaceCachedRunsRequestSchema, SetCurrentLoopRequestSchema,
} from '@proto/loop_state/v1/loop_state_pb'
import { ListRepositoriesResponseSchema, RepositorySchema } from '@proto/repository/v1/repository_pb'
import { ReplaceCachedRepositoriesRequestSchema } from '@proto/repo_state/v1/repo_state_pb'
import {
  ApplySessionRequestSchema,
  SetCurrentOrgRequestSchema,
  SetOrganizationsRequestSchema,
} from '@proto/auth_state/v1/auth_state_pb'
import {
  ListAutopilotControllersResponseSchema, GetIterationsResponseSchema,
} from '@proto/autopilot/v1/autopilot_pb'
import {
  ReplaceCachedControllersRequestSchema, ReplaceCachedIterationsRequestSchema,
  SetCurrentControllerRequestSchema,
  AutopilotControllerSnapshotSchema, AutopilotIterationSnapshotSchema,
} from '@proto/autopilot_state/v1/autopilot_state_pb'
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
  const loop = { list: '[]', current: '', runs: '[]' };
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
    loop.list = '[]'; loop.current = ''; loop.runs = '[]';
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
  // Minimal cache↔proto pod mapping for the mock — covers the fields the pod
  // store tests exercise (mockPod/mockPod2). Not the full PodData surface; the
  // real round-trip is covered by electron-adapter pod_cache_to_bytes.test.ts.
  type PodObj = Record<string, unknown>
  const protoPodToCacheMin = (p: ReturnType<typeof create<typeof PodSchema>>): PodObj => ({
    id: Number(p.id), pod_key: p.podKey, status: p.status,
    agent_status: p.agentStatus || undefined, created_at: p.createdAt || undefined,
    runner: p.runner ? { id: Number(p.runner.id), node_id: p.runner.nodeId, status: p.runner.status } : undefined,
    created_by: p.createdBy ? { id: Number(p.createdBy.id), username: p.createdBy.username, name: p.createdBy.name } : undefined,
  })
  const cacheToProtoPodMin = (p: PodObj) => {
    const r = p.runner as PodObj | undefined
    const cb = p.created_by as PodObj | undefined
    return create(PodSchema, {
      id: p.id != null ? BigInt(p.id as number) : 0n, podKey: (p.pod_key as string) ?? '',
      status: (p.status as string) ?? '', agentStatus: (p.agent_status as string) ?? '',
      createdAt: (p.created_at as string) ?? '',
      runner: r ? create(PodRunnerInfoSchema, { id: r.id != null ? BigInt(r.id as number) : undefined, nodeId: r.node_id as string, status: r.status as string }) : undefined,
      createdBy: cb ? create(PodCreatedByInfoSchema, { id: cb.id != null ? BigInt(cb.id as number) : undefined, username: cb.username as string, name: cb.name as string }) : undefined,
    })
  }

  const podState = {
    pods_json: fn(() => h.pod.pods),
    current_pod_json: fn(() => h.pod.current || undefined),
    get_pod_json: fn((key: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string }[]
      const p = list.find((x) => x.pod_key === key)
      return p ? JSON.stringify(p) : undefined
    }),
    // Proto-bytes mutators. Status/title/alias/perpetual bodies stay opaque
    // (the in-memory list is left untouched; tests assert "the bridge was
    // called" or override pods_json). The fetch→state apply_* and the bytes
    // readers DO model the backing store, since usePods/readPods now flow
    // through them — a no-op there would diverge the mock from real WasmPodState.
    insert_created_pod: fn((_bytes: Uint8Array) => undefined),
    patch_pod_perpetual: fn((_bytes: Uint8Array) => undefined),
    apply_pod_status_event: fn((_bytes: Uint8Array) => undefined),
    apply_pod_title_event: fn((_bytes: Uint8Array) => undefined),
    apply_pod_alias_event: fn((_bytes: Uint8Array) => undefined),
    apply_agent_status_event: fn((_bytes: Uint8Array) => undefined),
    apply_fetched_pods: fn((bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListPodsResponseSchema, bytes)
        h.pod.pods = JSON.stringify(resp.items.map(protoPodToCacheMin))
      } catch { /* noop */ }
    }),
    apply_appended_pods: fn((bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListPodsResponseSchema, bytes)
        const existing = JSON.parse(h.pod.pods) as { pod_key: string }[]
        const seen = new Set(existing.map((p) => p.pod_key))
        for (const p of resp.items) {
          const c = protoPodToCacheMin(p)
          if (!seen.has(c.pod_key)) existing.push(c)
        }
        h.pod.pods = JSON.stringify(existing)
      } catch { /* noop */ }
    }),
    pods_bytes: fn(() => {
      const list = JSON.parse(h.pod.pods) as Record<string, unknown>[]
      return toBinary(ReplaceCachedPodsRequestSchema,
        create(ReplaceCachedPodsRequestSchema, { pods: list.map(cacheToProtoPodMin) }))
    }),
    current_pod_bytes: fn(() => {
      if (!h.pod.current) return new Uint8Array()
      return toBinary(PodSchema, cacheToProtoPodMin(JSON.parse(h.pod.current) as Record<string, unknown>))
    }),
    get_pod_bytes: fn((key: string) => {
      const list = JSON.parse(h.pod.pods) as Record<string, unknown>[]
      const p = list.find((x) => (x.pod_key as string) === key)
      return p ? toBinary(PodSchema, cacheToProtoPodMin(p)) : new Uint8Array()
    }),
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

  // Minimal runner proto↔cache for the B fetch/read mock — covers the fields
  // runner store tests exercise; full round-trip is in runner_cache_to_bytes.test.ts.
  type RunnerObj = Record<string, unknown>
  const protoRunnerToCacheMin = (r: ReturnType<typeof create<typeof RunnerSchema>>): RunnerObj => ({
    id: Number(r.id), node_id: r.nodeId, status: r.status, is_enabled: r.isEnabled,
    current_pods: r.currentPods, max_concurrent_pods: r.maxConcurrentPods,
    last_heartbeat: r.lastHeartbeat || undefined, runner_version: r.runnerVersion || undefined,
    description: r.description || undefined, visibility: r.visibility || undefined,
    created_at: r.createdAt || undefined, updated_at: r.updatedAt || undefined,
  })
  const cacheToProtoRunnerMin = (r: RunnerObj) => create(RunnerSchema, {
    id: BigInt((r.id as number) ?? 0), nodeId: (r.node_id as string) ?? '', status: (r.status as string) ?? '',
    isEnabled: !!r.is_enabled, currentPods: (r.current_pods as number) ?? 0,
    maxConcurrentPods: (r.max_concurrent_pods as number) ?? 0, lastHeartbeat: (r.last_heartbeat as string) ?? '',
    runnerVersion: (r.runner_version as string) ?? '', description: (r.description as string) ?? '',
    visibility: (r.visibility as string) ?? '', createdAt: (r.created_at as string) ?? '', updatedAt: (r.updated_at as string) ?? '',
  })
  const runnerState = {
    set_runners: fn((j: string) => { h.runner.list = j }),
    runners_json: fn(() => h.runner.list),
    set_available_runners: fn((j: string) => { h.runner.available = j }),
    available_runners_json: fn(() => h.runner.available),
    set_current_runner: fn((j: string) => { h.runner.current = j }),
    current_runner_json: fn(() => h.runner.current || undefined),
    apply_fetched_runners: fn((bytes: Uint8Array) => {
      try { h.runner.list = JSON.stringify(fromBinary(ListRunnersResponseSchema, bytes).items.map(protoRunnerToCacheMin)) } catch { /* noop */ }
    }),
    apply_fetched_available_runners: fn((bytes: Uint8Array) => {
      try { h.runner.available = JSON.stringify(fromBinary(ListAvailableRunnersResponseSchema, bytes).items.map(protoRunnerToCacheMin)) } catch { /* noop */ }
    }),
    apply_fetched_current_runner: fn((bytes: Uint8Array) => {
      try { const r = fromBinary(GetRunnerResponseSchema, bytes).runner; h.runner.current = r ? JSON.stringify(protoRunnerToCacheMin(r)) : '' } catch { /* noop */ }
    }),
    runners_bytes: fn(() => toBinary(ReplaceCachedRunnersRequestSchema,
      create(ReplaceCachedRunnersRequestSchema, { runners: (JSON.parse(h.runner.list) as RunnerObj[]).map(cacheToProtoRunnerMin) }))),
    available_runners_bytes: fn(() => toBinary(ReplaceAvailableRunnersRequestSchema,
      create(ReplaceAvailableRunnersRequestSchema, { runners: (JSON.parse(h.runner.available) as RunnerObj[]).map(cacheToProtoRunnerMin) }))),
    current_runner_bytes: fn(() => h.runner.current
      ? toBinary(RunnerSchema, cacheToProtoRunnerMin(JSON.parse(h.runner.current) as RunnerObj)) : new Uint8Array()),
    update_runner_local: fn((id: number, json: string) => {
      const updated = JSON.parse(json) as { id: number };
      const arr = JSON.parse(h.runner.list) as { id: number }[];
      const idx = arr.findIndex((x) => x.id === id);
      if (idx >= 0) arr[idx] = updated;
      h.runner.list = JSON.stringify(arr);
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
    channels_bytes: fn(() => {
      const list = JSON.parse(h.channel.list) as Array<Record<string, unknown>>
      return toBinary(ReplaceCachedChannelsRequestSchema, create(ReplaceCachedChannelsRequestSchema, {
        channels: list.map(chToState),
      }))
    }),
    set_current_channel: fn((id: bigint | null) => { h.channel.current = id }),
    current_channel_json: fn(() => {
      if (h.channel.current === null) return undefined
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const ch = list.find((c) => c.id === Number(h.channel.current))
      return ch ? JSON.stringify(ch) : undefined
    }),
    current_channel_bytes: fn(() => {
      if (h.channel.current === null) return new Uint8Array()
      const list = JSON.parse(h.channel.list) as Array<Record<string, unknown>>
      const ch = list.find((c) => Number(c.id) === Number(h.channel.current))
      if (!ch) return new Uint8Array()
      return toBinary(InsertChannelRequestSchema, create(InsertChannelRequestSchema, { channel: chToState(ch) }))
    }),
    get_channel_json: fn((id: bigint) => {
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const ch = list.find((c) => c.id === Number(id))
      return ch ? JSON.stringify(ch) : undefined
    }),
    get_channel_bytes: fn((id: bigint) => {
      const list = JSON.parse(h.channel.list) as Array<Record<string, unknown>>
      const ch = list.find((c) => Number(c.id) === Number(id))
      if (!ch) return new Uint8Array()
      return toBinary(InsertChannelRequestSchema, create(InsertChannelRequestSchema, { channel: chToState(ch) }))
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
    get_messages_bytes: fn((chId: bigint) => {
      const entry = h.channel.msgs.get(cKey(chId))
      const messages = entry ? (JSON.parse(entry.json) as Array<Record<string, unknown>>) : []
      return toBinary(ReplaceCachedChannelMessagesRequestSchema, create(ReplaceCachedChannelMessagesRequestSchema, {
        channelId: chId,
        hasMore: entry?.hasMore ?? false,
        messages: messages.map((m) => ({
          id: BigInt(m.id as number), channelId: BigInt(m.channel_id as number),
          body: (m.body as string) ?? '', messageType: (m.message_type as string) ?? '',
          senderUserId: m.sender_user_id !== undefined ? BigInt(m.sender_user_id as number) : undefined,
          createdAt: (m.created_at as string) ?? '',
          contentJson: m.content_json as string | undefined,
          mentionsJson: m.mentions_json as string | undefined,
        })),
      }))
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
    get_last_message_bytes: fn(() => new Uint8Array()),
    set_last_message: fn(),
    // Service async methods (API calls via WASM)
    create_channel: fn().mockResolvedValue('{}'),
    archive_channel: fn().mockResolvedValue(undefined),
    unarchive_channel: fn().mockResolvedValue(undefined),
    join_channel: fn().mockResolvedValue('{}'),
    leave_channel: fn().mockResolvedValue('{}'),
    send_message: fn().mockResolvedValue('{}'),
    edit_message: fn().mockResolvedValue('{}'),
    delete_message: fn().mockResolvedValue(undefined),
    mark_read: fn().mockResolvedValue(undefined),
    mute_channel: fn().mockResolvedValue(undefined),
    fetch_channel_members: fn().mockResolvedValue('{"members":[],"total":0}'),
    invite_channel_members: fn().mockResolvedValue(undefined),
    channel_members_json: fn((id: bigint) => h.channel.members.get(String(id)) ?? '[]'),
    channel_members_bytes: fn((id: bigint) => {
      const members = JSON.parse(h.channel.members.get(String(id)) ?? '[]') as Array<Record<string, unknown>>
      return toBinary(ReplaceChannelMembersRequestSchema, create(ReplaceChannelMembersRequestSchema, {
        channelId: id,
        members: members.map((m) => ({
          channelId: BigInt(m.channel_id as number), userId: BigInt(m.user_id as number),
          role: m.role as string, isMuted: !!m.is_muted, joinedAt: m.joined_at as string,
        })),
      }))
    }),
    get_channel_pods: fn().mockResolvedValue('{"pods":[]}'),
    channel_pods_json: fn((id: bigint) => h.channel.pods.get(String(id)) ?? '[]'),
    channel_pods_bytes: fn((id: bigint) => {
      const pods = JSON.parse(h.channel.pods.get(String(id)) ?? '[]') as Array<Record<string, unknown>>
      return toBinary(ReplaceChannelPodsRequestSchema, create(ReplaceChannelPodsRequestSchema, {
        channelId: id,
        pods: pods.map((p) => ({
          id: BigInt((p.id as number) ?? 0), podKey: p.pod_key as string,
          alias: p.alias as string | undefined, status: p.status as string,
          agentStatus: p.agent_status as string,
        })),
      }))
    }),
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
    // Fetch→state path: decode wire ListChannelsResponse (channel/v1) directly,
    // mirroring Rust apply_fetched_channels.
    apply_fetched_channels: fn((bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListChannelsResponseSchema, bytes)
        h.channel.list = JSON.stringify(resp.items.map((c) => decodeProtoChannel(c)))
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
    apply_fetched_channel: fn((bytes: Uint8Array) => {
      try {
        const channel = decodeProtoChannel(fromBinary(ChannelSchema, bytes))
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
    // Fetch→state path: decode wire ListChannelMessagesResponse (channel/v1).
    apply_fetched_messages: fn((chId: bigint, bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListChannelMessagesResponseSchema, bytes)
        const msgs = resp.items.map(decodeProtoMessage)
        h.channel.msgs.set(cKey(Number(chId)), { json: JSON.stringify(msgs), hasMore: resp.hasMore })
      } catch { /* noop */ }
    }),
    apply_fetched_messages_prepend: fn((chId: bigint, bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListChannelMessagesResponseSchema, bytes)
        const k = cKey(Number(chId))
        const entry = h.channel.msgs.get(k)
        const existing = entry ? JSON.parse(entry.json) as { id: number }[] : []
        const incoming = resp.items.map(decodeProtoMessage)
        const ids = new Set(existing.map((m) => m.id))
        const merged = [...incoming.filter((m) => !ids.has(m.id)), ...existing]
        merged.sort((a, b) => a.id - b.id)
        h.channel.msgs.set(k, { json: JSON.stringify(merged), hasMore: resp.hasMore })
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
    // Fetch→state path: decode wire ListChannelPods/MembersResponse (channel/v1).
    apply_fetched_pods: fn((chId: bigint, bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListChannelPodsResponseSchema, bytes)
        const pods = resp.items.map((p) => ({
          id: Number(p.id), pod_key: p.podKey, alias: p.alias,
          status: p.status, agent_status: p.agentStatus,
        }))
        h.channel.pods.set(String(chId), JSON.stringify(pods))
      } catch { /* noop */ }
    }),
    apply_fetched_members: fn((chId: bigint, bytes: Uint8Array) => {
      try {
        const resp = fromBinary(ListChannelMembersResponseSchema, bytes)
        const members = resp.items.map((m) => ({
          channel_id: Number(m.channelId), user_id: Number(m.userId),
          role: m.role, is_muted: m.isMuted, joined_at: m.joinedAt,
        }))
        h.channel.members.set(String(chId), JSON.stringify(members))
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

  // Reverse of decodeProtoChannel: web snake_case Channel → state proto literal
  // for prost-bytes read mocks (channels_bytes / current_channel_bytes /
  // get_channel_bytes). Single SSOT so the three read mocks can't drift.
  function chToState(c: Record<string, unknown>) {
    return {
      id: BigInt(c.id as number), name: c.name as string,
      organizationId: c.organization_id !== undefined ? BigInt(c.organization_id as number) : undefined,
      isArchived: !!c.is_archived, isMember: !!c.is_member,
      memberCount: c.member_count !== undefined ? BigInt(c.member_count as number) : undefined,
      visibility: c.visibility as string,
      createdAt: c.created_at as string, updatedAt: c.updated_at as string,
    }
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

  // Minimal ticket proto↔cache for the B fetch/read mock — covers the fields
  // ticket store tests exercise; full round-trip is in ticket_cache_to_bytes.test.ts.
  type TicketObj = Record<string, unknown>
  const protoTicketToCacheMin = (t: ReturnType<typeof create<typeof TicketSchema>>): TicketObj => ({
    id: Number(t.id), number: t.number, slug: t.slug, title: t.title, content: t.content,
    status: t.status, priority: t.priority, severity: t.severity, estimate: t.estimate,
    due_date: t.dueDate, started_at: t.startedAt, completed_at: t.completedAt,
    created_at: t.createdAt, updated_at: t.updatedAt,
    repository_id: t.repositoryId !== undefined ? Number(t.repositoryId) : undefined,
  })
  const cacheToProtoTicketMin = (t: TicketObj) => create(TicketSchema, {
    id: BigInt((t.id as number) ?? 0), number: (t.number as number) ?? 0, slug: (t.slug as string) ?? '',
    title: (t.title as string) ?? '', content: (t.content as string) ?? '', status: (t.status as string) ?? '',
    priority: (t.priority as string) ?? '', severity: (t.severity as string) ?? '', estimate: (t.estimate as number) ?? 0,
    dueDate: (t.due_date as string) ?? '', startedAt: (t.started_at as string) ?? '', completedAt: (t.completed_at as string) ?? '',
    createdAt: (t.created_at as string) ?? '', updatedAt: (t.updated_at as string) ?? '',
    repositoryId: t.repository_id !== undefined ? BigInt(t.repository_id as number) : undefined,
  })

  // ticketState now mirrors WasmTicketState (proto bytes mutators + JSON
  // reads). Tests that need to observe cache state can override these via
  // vi.mocked(...).mockImplementation; defaults are no-ops so the bridge
  // doesn't crash when store actions fire.
  const ticketState = {
    tickets_json: fn(() => h.ticket.list),
    tickets_bytes: fn(() => toBinary(ReplaceCachedTicketsRequestSchema,
      create(ReplaceCachedTicketsRequestSchema, { tickets: (JSON.parse(h.ticket.list) as TicketObj[]).map(cacheToProtoTicketMin) }))),
    apply_fetched_tickets: fn((bytes: Uint8Array) => {
      try { h.ticket.list = JSON.stringify(fromBinary(ListTicketsResponseSchema, bytes).items.map(protoTicketToCacheMin)) } catch { /* noop */ }
    }),
    board_columns_json: fn(() => h.ticket.boardCols),
    labels_json: fn(() => h.ticket.labels),
    current_ticket_json: fn(() => h.ticket.current || undefined),
    // Read side (B): encode helper-store JSON into the state proto wrappers.
    current_ticket_bytes: fn(() => {
      const cur = h.ticket.current ? JSON.parse(h.ticket.current) as TicketObj : null
      return cur ? toBinary(SetCurrentTicketRequestSchema, create(SetCurrentTicketRequestSchema, { ticket: cacheToProtoTicketMin(cur) })) : new Uint8Array()
    }),
    board_columns_bytes: fn(() => toBinary(ReplaceBoardColumnsRequestSchema, create(ReplaceBoardColumnsRequestSchema, {
      columns: (JSON.parse(h.ticket.boardCols) as { status: string; count?: number; tickets: TicketObj[] }[]).map((c) =>
        create(BoardColumnSchema, { status: c.status, totalCount: BigInt(c.count ?? 0), tickets: c.tickets.map(cacheToProtoTicketMin) })),
    }))),
    labels_bytes: fn(() => toBinary(ReplaceCachedLabelsRequestSchema, create(ReplaceCachedLabelsRequestSchema, {
      labels: (JSON.parse(h.ticket.labels) as { id: number; name: string; color: string }[]).map((l) =>
        create(LabelSchema, { id: BigInt(l.id), name: l.name, color: l.color })),
    }))),
    apply_ticket_status_event: fn((_b: Uint8Array) => undefined),
    apply_ticket_deleted_event: fn((_b: Uint8Array) => undefined),
    apply_fetched_current_ticket: fn((bytes: Uint8Array) => {
      try { h.ticket.current = JSON.stringify(protoTicketToCacheMin(fromBinary(TicketSchema, bytes))) } catch { /* noop */ }
    }),
    apply_fetched_board_columns: fn((_b: Uint8Array) => undefined),
    apply_appended_board_column_tickets: fn((_s: string, _b: Uint8Array) => undefined),
    apply_fetched_labels: fn((bytes: Uint8Array) => {
      try { h.ticket.labels = JSON.stringify(fromBinary(ListLabelsResponseSchema, bytes).items.map((l) => ({ id: Number(l.id), name: l.name, color: l.color }))) } catch { /* noop */ }
    }),
    insert_created_ticket: fn((_b: Uint8Array) => undefined),
    patch_cached_ticket: fn((_b: Uint8Array) => undefined),
    replace_board_columns: fn((_b: Uint8Array) => undefined),
    append_board_column_tickets: fn((_b: Uint8Array) => undefined),
    set_current_ticket: fn((_b: Uint8Array) => undefined),
    replace_cached_labels: fn((_b: Uint8Array) => undefined),
    insert_created_label: fn((_b: Uint8Array) => undefined),
    remove_cached_label: fn((_b: Uint8Array) => undefined),
    filter_tickets: fn((_b: Uint8Array) => new Uint8Array()),
    ticket_pods_bytes: fn((_s: string) => toBinary(ReplaceCachedPodsRequestSchema, create(ReplaceCachedPodsRequestSchema, { pods: [] }))),
    set_ticket_pods: fn((_s: string, _j: string) => undefined),
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
    // Read side (B): empty bytes = no topology (replace_topology mock is a no-op,
    // so tests never populate real topology data through this mock).
    topology_bytes: fn(() => new Uint8Array()),
    clear_topology: fn(() => { h.mesh.topo = '' }),
    select_node: fn((key?: string) => { h.mesh.selected = key }),
    selected_node: fn(() => h.mesh.selected),
    fetch_topology: fn().mockResolvedValue(JSON.stringify({ nodes: [], edges: [], channels: [], runners: [] })),
    // Proto-bytes mutator (mirror state_mesh.rs).
    replace_topology: fn((_b: Uint8Array) => undefined),
    // Connect-RPC bridge — empty Uint8Array decodes to default-valued proto.
    getMeshTopologyConnect: fn().mockResolvedValue(new Uint8Array()),
    getTicketPodsConnect: fn().mockResolvedValue(new Uint8Array()),
    batchGetTicketPodsConnect: fn().mockResolvedValue(new Uint8Array()),
    createPodForTicketConnect: fn().mockResolvedValue(new Uint8Array()),
  }

  // Minimal loop proto↔cache for the B fetch/read mock — covers the fields loop
  // store tests exercise; full round-trip is in loop_cache_to_bytes.test.ts.
  type LoopObj = Record<string, unknown>
  const protoLoopToCacheMin = (l: ReturnType<typeof create<typeof LoopSchema>>): LoopObj => ({
    id: Number(l.id), slug: l.slug, name: l.name, status: l.status,
    permission_mode: l.permissionMode, prompt_template: l.promptTemplate,
    execution_mode: l.executionMode, sandbox_strategy: l.sandboxStrategy,
    session_persistence: l.sessionPersistence, concurrency_policy: l.concurrencyPolicy,
    max_concurrent_runs: l.maxConcurrentRuns, max_retained_runs: l.maxRetainedRuns,
    timeout_minutes: l.timeoutMinutes, total_runs: Number(l.totalRuns),
    successful_runs: Number(l.successfulRuns), failed_runs: Number(l.failedRuns),
    active_run_count: Number(l.activeRunCount), used_env_bundles: l.usedEnvBundles ?? [],
    autopilot_config: {}, created_at: l.createdAt, updated_at: l.updatedAt,
  })
  const cacheToProtoLoopMin = (l: LoopObj) => create(LoopSchema, {
    id: BigInt((l.id as number) ?? 0), slug: (l.slug as string) ?? '', name: (l.name as string) ?? '',
    status: (l.status as string) ?? '', permissionMode: (l.permission_mode as string) ?? '',
    promptTemplate: (l.prompt_template as string) ?? '', executionMode: (l.execution_mode as string) ?? '',
    sandboxStrategy: (l.sandbox_strategy as string) ?? '', sessionPersistence: !!l.session_persistence,
    concurrencyPolicy: (l.concurrency_policy as string) ?? '', maxConcurrentRuns: (l.max_concurrent_runs as number) ?? 0,
    maxRetainedRuns: (l.max_retained_runs as number) ?? 0, timeoutMinutes: (l.timeout_minutes as number) ?? 0,
    totalRuns: BigInt((l.total_runs as number) ?? 0), successfulRuns: BigInt((l.successful_runs as number) ?? 0),
    failedRuns: BigInt((l.failed_runs as number) ?? 0), activeRunCount: BigInt((l.active_run_count as number) ?? 0),
    usedEnvBundles: (l.used_env_bundles as string[]) ?? [], createdAt: (l.created_at as string) ?? '', updatedAt: (l.updated_at as string) ?? '',
  })
  const protoRunToCacheMin = (r: ReturnType<typeof create<typeof LoopRunSchema>>): LoopObj => ({
    id: Number(r.id), loop_id: Number(r.loopId), run_number: Number(r.runNumber), status: r.status,
    pod_key: r.podKey, started_at: r.startedAt, finished_at: r.completedAt,
    error_message: r.errorMessage, created_at: r.createdAt, trigger_type: '',
  })
  const cacheToProtoRunMin = (r: LoopObj) => create(LoopRunSchema, {
    id: BigInt((r.id as number) ?? 0), loopId: BigInt((r.loop_id as number) ?? 0),
    runNumber: BigInt((r.run_number as number) ?? 0), status: (r.status as string) ?? '',
    podKey: (r.pod_key as string) ?? '', startedAt: (r.started_at as string) ?? '',
    completedAt: (r.finished_at as string) ?? '', errorMessage: (r.error_message as string) ?? '', createdAt: (r.created_at as string) ?? '',
  })

  const loopState = {
    loops_json: fn(() => h.loop.list),
    current_loop_json: fn(() => h.loop.current || undefined),
    runs_json: fn(() => h.loop.runs),
    get_loop_by_slug_json: fn((slug: string) => {
      const arr = JSON.parse(h.loop.list) as { slug: string }[]
      const l = arr.find((x) => x.slug === slug)
      return l ? JSON.stringify(l) : undefined
    }),
    // Read side (B): cache → state proto bytes; fetch→state: wire response → cache.
    loops_bytes: fn(() => toBinary(ReplaceCachedLoopsRequestSchema,
      create(ReplaceCachedLoopsRequestSchema, { loops: (JSON.parse(h.loop.list) as LoopObj[]).map(cacheToProtoLoopMin) }))),
    runs_bytes: fn(() => toBinary(ReplaceCachedRunsRequestSchema,
      create(ReplaceCachedRunsRequestSchema, { runs: (JSON.parse(h.loop.runs) as LoopObj[]).map(cacheToProtoRunMin) }))),
    current_loop_bytes: fn(() => h.loop.current
      ? toBinary(SetCurrentLoopRequestSchema, create(SetCurrentLoopRequestSchema, { loop: cacheToProtoLoopMin(JSON.parse(h.loop.current) as LoopObj) }))
      : new Uint8Array()),
    apply_fetched_loops: fn((bytes: Uint8Array) => {
      try { h.loop.list = JSON.stringify(fromBinary(ListLoopsResponseSchema, bytes).items.map(protoLoopToCacheMin)) } catch { /* noop */ }
    }),
    apply_fetched_current_loop: fn((bytes: Uint8Array) => {
      try { h.loop.current = JSON.stringify(protoLoopToCacheMin(fromBinary(LoopSchema, bytes))) } catch { /* noop */ }
    }),
    apply_fetched_runs: fn((bytes: Uint8Array) => {
      try { h.loop.runs = JSON.stringify(fromBinary(ListRunsResponseSchema, bytes).items.map(protoRunToCacheMin)) } catch { /* noop */ }
    }),
    apply_appended_runs: fn((bytes: Uint8Array) => {
      try {
        const existing = JSON.parse(h.loop.runs) as LoopObj[]
        h.loop.runs = JSON.stringify([...existing, ...fromBinary(ListRunsResponseSchema, bytes).items.map(protoRunToCacheMin)])
      } catch { /* noop */ }
    }),
    // Proto-state mutations (binary wire) — TS store uses these.
    set_current_loop: fn(), clear_current_loop: fn(),
    patch_loop_from_action: fn(), insert_loop_run: fn(),
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

  // Minimal repository proto↔cache for the B fetch/read mock — covers the fields
  // repo store tests exercise; full round-trip is in repository_cache_to_bytes.test.ts.
  type RepoObj = Record<string, unknown>
  const protoRepoToCacheMin = (r: ReturnType<typeof create<typeof RepositorySchema>>): RepoObj => ({
    id: Number(r.id), organization_id: Number(r.organizationId), provider_type: r.providerType,
    provider_base_url: r.providerBaseUrl, external_id: r.externalId, name: r.name, slug: r.slug,
    default_branch: r.defaultBranch, ticket_prefix: r.ticketPrefix, visibility: r.visibility,
    is_active: r.isActive, created_at: r.createdAt, updated_at: r.updatedAt,
  })
  const cacheToProtoRepoMin = (r: RepoObj) => create(RepositorySchema, {
    id: BigInt((r.id as number) ?? 0), organizationId: BigInt((r.organization_id as number) ?? 0),
    providerType: (r.provider_type as string) ?? '', providerBaseUrl: (r.provider_base_url as string) ?? '',
    externalId: (r.external_id as string) ?? '', name: (r.name as string) ?? '', slug: (r.slug as string) ?? '',
    defaultBranch: (r.default_branch as string) ?? '', ticketPrefix: (r.ticket_prefix as string) ?? '',
    visibility: (r.visibility as string) ?? '', isActive: !!r.is_active,
    createdAt: (r.created_at as string) ?? '', updatedAt: (r.updated_at as string) ?? '',
  })

  const repoState = {
    set_repositories: fn(), repositories_json: fn(() => h.repo.list),
    set_current_repo: fn(), current_repo_json: fn(() => h.repo.current || undefined),
    add_repository: fn(), update_repository: fn(), remove_repository: fn(),
    set_branches: fn(), branches_json: fn(() => h.repo.branches),
    repositories_bytes: fn(() => toBinary(ReplaceCachedRepositoriesRequestSchema,
      create(ReplaceCachedRepositoriesRequestSchema, { repositories: (JSON.parse(h.repo.list) as RepoObj[]).map(cacheToProtoRepoMin) }))),
    apply_fetched_repositories: fn((bytes: Uint8Array) => {
      try { h.repo.list = JSON.stringify(fromBinary(ListRepositoriesResponseSchema, bytes).items.map(protoRepoToCacheMin)) } catch { /* noop */ }
    }),
    // Proto-bytes mutators (mirror state_repo.rs).
    set_current_repo_proto: fn((_b: Uint8Array) => undefined),
    replace_branches: fn((_b: Uint8Array) => undefined),
    insert_repository: fn((_b: Uint8Array) => undefined),
    patch_repository: fn((_b: Uint8Array) => undefined),
  }

  const autopilotState = {
    set_controllers: fn(), controllers_json: fn(() => h.autopilot.controllers),
    set_current_controller: fn(), current_controller_json: fn(() => h.autopilot.current || undefined),
    apply_fetched_controllers: fn(), apply_fetched_current_controller: fn(), apply_fetched_iterations: fn(),
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

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false, media: query, onchange: null,
    addListener: vi.fn(), removeListener: vi.fn(),
    addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
  })),
})

afterEach(() => { h.reset(); acpMgr._reset(); vi.clearAllMocks() })

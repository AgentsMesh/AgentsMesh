import '@testing-library/jest-dom'
import { vi, afterEach } from 'vitest'
import { createAcpManager } from './wasm-mock-acp'

// ---------------------------------------------------------------------------
// Hoisted state: survives vi.mock hoisting, resets between tests
// ---------------------------------------------------------------------------
const h = vi.hoisted(() => {
  // Simple JSON state store
  const mkStore = () => ({ v: '' as string });
  const pod = { pods: '[]', current: '' };
  const runner = { list: '[]', available: '[]', current: '' };
  const org = { orgs: '[]', current: '', members: '[]' };
  const user = { profile: '' };
  const channel = {
    list: '[]', current: null as bigint | null,
    msgs: new Map<string, { json: string; hasMore: boolean }>(),
    unread: new Map<string, number>(),
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
    org.orgs = '[]'; org.current = ''; org.members = '[]';
    user.profile = '';
    channel.list = '[]'; channel.current = null;
    channel.msgs.clear(); channel.unread.clear();
    ticket.list = '[]'; ticket.labels = '[]'; ticket.boardCols = '[]'; ticket.current = '';
    mesh.topo = ''; mesh.selected = undefined;
    loop.list = '[]'; loop.current = '';
    gitProvider.v = '';
    repo.list = '[]'; repo.current = ''; repo.branches = '[]';
    autopilot.controllers = '[]'; autopilot.current = '';
    autopilot.iterations.clear(); autopilot.thinkings.clear(); autopilot.thinkingHistory.clear();
  }

  return { pod, runner, org, user, channel, ticket, mesh, loop, gitProvider, repo, autopilot, reset };
})

const acpMgr = createAcpManager()

// ---------------------------------------------------------------------------
// Mock WASM Core
// ---------------------------------------------------------------------------
vi.mock('@/lib/wasm-core', () => {
  const fn = vi.fn

  const mockClient = {
    get: fn().mockResolvedValue('{}'),
    post: fn().mockResolvedValue('{}'),
    put: fn().mockResolvedValue('{}'),
    delete: fn().mockResolvedValue('{}'),
    patch: fn().mockResolvedValue('{}'),
    set_token: fn(),
    set_org_slug: fn(),
    clear_auth: fn(),
    org_path: fn((p: string) => `/api/v1/orgs/test-org${p}`),
  }

  const authBox: { user: unknown; current_org: unknown; organizations: unknown[] } = {
    user: null, current_org: null, organizations: [],
  };
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
    apply_session: fn((sessionJson: string) => {
      try {
        const s = JSON.parse(sessionJson);
        authBox.user = s.user ?? null;
      } catch { /* noop */ }
    }),
    set_organizations: fn((orgsJson: string) => {
      try {
        const orgs = JSON.parse(orgsJson);
        authBox.organizations = Array.isArray(orgs) ? orgs : [];
        if (authBox.current_org == null && authBox.organizations.length > 0) {
          authBox.current_org = authBox.organizations[0];
        }
      } catch { /* noop */ }
    }),
    set_current_org: fn((orgJson: string) => {
      if (orgJson === '') { authBox.current_org = null; return; }
      try { authBox.current_org = JSON.parse(orgJson); } catch { /* noop */ }
    }),
    clear_session: fn(() => {
      authBox.user = null; authBox.current_org = null; authBox.organizations = [];
    }),
    _reset: () => { authBox.user = null; authBox.current_org = null; authBox.organizations = []; },
  }

  // --- Pod state / service ---
  const podState = {
    set_pods: fn((j: string) => { h.pod.pods = j }),
    pods_json: fn(() => h.pod.pods),
    upsert_pod: fn((j: string) => {
      const p = JSON.parse(j) as { pod_key: string }
      const list = JSON.parse(h.pod.pods) as { pod_key: string }[]
      const idx = list.findIndex((x) => x.pod_key === p.pod_key)
      if (idx >= 0) list[idx] = p; else list.push(p)
      h.pod.pods = JSON.stringify(list)
    }),
    update_pod_status: fn((key: string, status: string, agentStatus?: string, errorCode?: string, errorMsg?: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string; status: string; agent_status?: string; error_code?: string; error_message?: string }[]
      const p = list.find((x) => x.pod_key === key)
      if (p) { p.status = status; if (agentStatus !== undefined) p.agent_status = agentStatus; if (errorCode !== undefined) p.error_code = errorCode; if (errorMsg !== undefined) p.error_message = errorMsg }
      h.pod.pods = JSON.stringify(list)
    }),
    update_pod_title: fn((key: string, title: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string; title?: string }[]
      const p = list.find((x) => x.pod_key === key); if (p) p.title = title
      h.pod.pods = JSON.stringify(list)
    }),
    update_pod_alias: fn((key: string, alias: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string; alias?: string }[]
      const p = list.find((x) => x.pod_key === key); if (p) p.alias = alias
      h.pod.pods = JSON.stringify(list)
    }),
    update_agent_status: fn((key: string, agentStatus: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string; agent_status?: string }[]
      const p = list.find((x) => x.pod_key === key); if (p) p.agent_status = agentStatus
      h.pod.pods = JSON.stringify(list)
    }),
    remove_pod: fn((key: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string }[]
      h.pod.pods = JSON.stringify(list.filter((x) => x.pod_key !== key))
    }),
    set_current_pod: fn((j: string) => { h.pod.current = j }),
    current_pod_json: fn(() => h.pod.current || undefined),
    get_pod_json: fn((key: string) => {
      const list = JSON.parse(h.pod.pods) as { pod_key: string }[]
      const p = list.find((x) => x.pod_key === key)
      return p ? JSON.stringify(p) : undefined
    }),
    // Service async methods (return resolved promises by default)
    fetch_pods: fn().mockResolvedValue(JSON.stringify({ pods: [], total: 0 })),
    fetch_pod: fn().mockResolvedValue('{}'),
    fetch_sidebar_pods: fn().mockResolvedValue(JSON.stringify({ pods: [], total: 0, hasMore: false })),
    load_more_pods: fn().mockResolvedValue(JSON.stringify({ newPods: [], total: 0, hasMore: false, allCount: 0 })),
    create_pod: fn().mockResolvedValue('{}'),
    terminate_pod: fn().mockResolvedValue(undefined),
    update_pod_alias_api: fn().mockResolvedValue(undefined),
    get_pod_connection: fn().mockResolvedValue('{}'),
  }

  // --- Runner state ---
  const runnerState = {
    set_runners: fn((j: string) => { h.runner.list = j }),
    runners_json: fn(() => h.runner.list),
    set_available_runners: fn((j: string) => { h.runner.available = j }),
    available_runners_json: fn(() => h.runner.available),
    set_current_runner: fn((j: string) => { h.runner.current = j }),
    current_runner_json: fn(() => h.runner.current || undefined),
    update_runner: fn((id: number, json: string) => {
      const updated = JSON.parse(json) as { id: number };
      const arr = JSON.parse(h.runner.list) as { id: number }[];
      const idx = arr.findIndex((x) => x.id === id);
      if (idx >= 0) arr[idx] = updated;
      h.runner.list = JSON.stringify(arr);
    }),
    update_runner_status: fn((id: bigint, status: string) => {
      for (const field of ['list', 'available'] as const) {
        const arr = JSON.parse(h.runner[field]) as { id: number; status: string }[]
        const r = arr.find((x) => x.id === Number(id)); if (r) r.status = status
        h.runner[field] = JSON.stringify(arr)
      }
      if (status !== 'online') {
        const avail = JSON.parse(h.runner.available) as { id: number }[]
        h.runner.available = JSON.stringify(avail.filter((x) => x.id !== Number(id)))
      }
    }),
    remove_runner: fn((id: bigint) => {
      for (const field of ['list', 'available'] as const) {
        const arr = JSON.parse(h.runner[field]) as { id: number }[]
        h.runner[field] = JSON.stringify(arr.filter((x) => x.id !== Number(id)))
      }
    }),
  }

  // --- Org state ---
  const orgState = {
    set_organizations: fn((j: string) => { h.org.orgs = j }),
    organizations_json: fn(() => h.org.orgs),
    add_organization: fn((j: string) => {
      const arr = JSON.parse(h.org.orgs); arr.push(JSON.parse(j)); h.org.orgs = JSON.stringify(arr)
    }),
    update_organization: fn((id: number, j: string) => {
      const arr = JSON.parse(h.org.orgs) as { id: number }[]
      const idx = arr.findIndex((o) => o.id === id)
      if (idx >= 0) { arr[idx] = JSON.parse(j); h.org.orgs = JSON.stringify(arr) }
      const cur = h.org.current ? JSON.parse(h.org.current) as { id: number } : null
      if (cur?.id === id) h.org.current = j
    }),
    remove_organization: fn((id: number) => {
      const arr = JSON.parse(h.org.orgs) as { id: number }[]
      h.org.orgs = JSON.stringify(arr.filter((o) => o.id !== id))
      const cur = h.org.current ? JSON.parse(h.org.current) as { id: number } : null
      if (cur?.id === id) h.org.current = ''
    }),
    set_current_org: fn((j: string) => { h.org.current = j }),
    current_org_json: fn(() => h.org.current || undefined),
    set_members: fn((j: string) => { h.org.members = j }),
    members_json: fn(() => h.org.members),
    add_member: fn((j: string) => {
      const arr = JSON.parse(h.org.members); arr.push(JSON.parse(j)); h.org.members = JSON.stringify(arr)
    }),
    update_member: fn((userId: number, j: string) => {
      const arr = JSON.parse(h.org.members) as { user_id: number }[]
      const idx = arr.findIndex((m) => m.user_id === userId)
      if (idx >= 0) { arr[idx] = JSON.parse(j); h.org.members = JSON.stringify(arr) }
    }),
    remove_member: fn((id: string) => {
      const arr = JSON.parse(h.org.members) as { user_id?: number; id?: number }[]
      h.org.members = JSON.stringify(arr.filter((m) => String(m.user_id) !== id && String(m.id) !== id))
    }),
  }

  // --- User state ---
  const userState = {
    set_profile: fn((j: string) => { h.user.profile = j }),
    profile_json: fn(() => h.user.profile || undefined),
    add_identity: fn((j: string) => {
      if (!h.user.profile) return
      const prof = JSON.parse(h.user.profile)
      prof.identities = [...(prof.identities || []), JSON.parse(j)]
      h.user.profile = JSON.stringify(prof)
    }),
    remove_identity: fn(),
    identities_json: fn(() => {
      if (!h.user.profile) return '[]'
      return JSON.stringify(JSON.parse(h.user.profile).identities || [])
    }),
  }

  // --- Channel state ---
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
    // Single channel CRUD
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
    // Atomic select
    select_channel: fn((id?: bigint) => {
      if (id === undefined) { h.channel.current = null; return undefined }
      h.channel.current = id
      h.channel.unread.delete(cKey(id))
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const ch = list.find((c) => c.id === Number(id))
      return ch ? JSON.stringify(ch) : undefined
    }),
    // Current user
    set_current_user: fn(),
    set_current_user_id: fn(),
    // Messages
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
    // Unread counts
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
    // Mention counts (stub)
    increment_mention: fn(),
    clear_channel_mentions: fn(),
    get_mention_count: fn(() => 0),
    total_mention_count: fn(() => 0),
    set_mention_counts: fn(),
    mention_counts_json: fn(() => '{}'),
    // Sorting
    sorted_channel_ids_json: fn(() => '[]'),
    total_unread_count: fn(() => {
      let total = 0; for (const v of h.channel.unread.values()) total += v; return total
    }),
    // Preview
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
    remove_channel_member: fn().mockResolvedValue(undefined),
    channel_members_json: fn(() => '[]'),
    get_channel_pods: fn().mockResolvedValue('{"pods":[]}'),
    channel_pods_json: fn(() => '[]'),
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
    update_channel_local: fn((id: bigint, json: string) => {
      const ch = JSON.parse(json)
      const list = JSON.parse(h.channel.list) as { id: number }[]
      const idx = list.findIndex((c) => c.id === Number(id))
      if (idx >= 0) { list[idx] = ch; h.channel.list = JSON.stringify(list) }
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
  }

  // --- Ticket state ---
  const ticketState = {
    set_tickets: fn((j: string) => { h.ticket.list = j }),
    tickets_json: fn(() => h.ticket.list),
    add_ticket: fn((j: string) => {
      const arr = JSON.parse(h.ticket.list); arr.push(JSON.parse(j)); h.ticket.list = JSON.stringify(arr)
    }),
    update_ticket: fn((slug: string, j: string) => {
      const arr = JSON.parse(h.ticket.list) as { slug: string }[]
      const idx = arr.findIndex((t) => t.slug === slug)
      if (idx >= 0) { arr[idx] = JSON.parse(j); h.ticket.list = JSON.stringify(arr) }
      const cur = h.ticket.current ? JSON.parse(h.ticket.current) as { slug: string } : null
      if (cur?.slug === slug) h.ticket.current = j
    }),
    update_ticket_status: fn((slug: string, status: string) => {
      const arr = JSON.parse(h.ticket.list) as { slug: string; status: string }[]
      const t = arr.find((t) => t.slug === slug)
      if (t) { t.status = status; h.ticket.list = JSON.stringify(arr) }
      if (h.ticket.current) {
        const cur = JSON.parse(h.ticket.current) as { slug: string; status: string }
        if (cur.slug === slug) { cur.status = status; h.ticket.current = JSON.stringify(cur) }
      }
    }),
    remove_ticket: fn((slug: string) => {
      const arr = JSON.parse(h.ticket.list) as { slug: string }[]
      h.ticket.list = JSON.stringify(arr.filter((t) => t.slug !== slug))
      if (h.ticket.current) {
        const cur = JSON.parse(h.ticket.current) as { slug: string }
        if (cur.slug === slug) h.ticket.current = ''
      }
    }),
    get_ticket_by_slug_json: fn((slug: string) => {
      const arr = JSON.parse(h.ticket.list) as { slug: string }[]
      const t = arr.find((t) => t.slug === slug)
      return t ? JSON.stringify(t) : undefined
    }),
    filter_tickets_json: fn((search: string, statusesJson: string, prioritiesJson: string, repoIdsJson: string) => {
      const tickets = JSON.parse(h.ticket.list) as { slug: string; title: string; status: string; priority: string; repository_id?: number }[]
      const statuses = JSON.parse(statusesJson) as string[]
      const priorities = JSON.parse(prioritiesJson) as string[]
      const repoIds = JSON.parse(repoIdsJson) as number[]
      const q = search.toLowerCase()
      return JSON.stringify(tickets.filter((t) => {
        if (q && !t.title.toLowerCase().includes(q) && !t.slug.toLowerCase().includes(q)) return false
        if (statuses.length && !statuses.includes(t.status)) return false
        if (priorities.length && !priorities.includes(t.priority)) return false
        if (repoIds.length && !repoIds.includes(t.repository_id ?? 0)) return false
        return true
      }))
    }),
    // Board columns
    board_columns_json: fn(() => h.ticket.boardCols),
    set_board_columns: fn((j: string) => {
      h.ticket.boardCols = j
      const cols = JSON.parse(j) as { tickets: unknown[] }[]
      h.ticket.list = JSON.stringify(cols.flatMap((c) => c.tickets))
    }),
    append_column_tickets: fn((status: string, j: string) => {
      const cols = JSON.parse(h.ticket.boardCols) as { status: string; tickets: unknown[] }[]
      const col = cols.find((c) => c.status === status)
      if (col) { col.tickets.push(...JSON.parse(j)); h.ticket.boardCols = JSON.stringify(cols) }
      h.ticket.list = JSON.stringify(cols.flatMap((c) => c.tickets))
    }),
    // Labels
    labels_json: fn(() => h.ticket.labels),
    set_labels: fn((j: string) => { h.ticket.labels = j }),
    add_label: fn((j: string) => {
      const arr = JSON.parse(h.ticket.labels); arr.push(JSON.parse(j)); h.ticket.labels = JSON.stringify(arr)
    }),
    remove_label: fn((id: number) => {
      const arr = JSON.parse(h.ticket.labels) as { id: number }[]
      h.ticket.labels = JSON.stringify(arr.filter((l) => l.id !== id))
    }),
    // Current ticket
    current_ticket_json: fn(() => h.ticket.current || undefined),
    set_current_ticket: fn((j: string) => { h.ticket.current = j }),
    // Service async methods
    fetch_tickets: fn().mockResolvedValue(JSON.stringify({ tickets: [], total: 0 })),
    fetch_ticket: fn().mockResolvedValue('{}'),
    create_ticket: fn().mockResolvedValue('{}'),
    update_ticket_api: fn().mockResolvedValue('{}'),
    delete_ticket: fn().mockResolvedValue(undefined),
    update_status: fn().mockResolvedValue(undefined),
    fetch_board: fn().mockResolvedValue(JSON.stringify({ board: { columns: [], priority_counts: {} } })),
    fetch_labels: fn().mockResolvedValue(JSON.stringify({ labels: [] })),
    get_sub_tickets: fn().mockResolvedValue(JSON.stringify({ sub_tickets: [] })),
    get_pods: fn().mockResolvedValue(JSON.stringify({ pods: [] })),
    get_ticket_pods: fn().mockResolvedValue(JSON.stringify({ pods: [] })),
    ticket_pods_json: fn(() => '[]'),
  }

  // --- Mesh state ---
  const meshState = {
    set_topology: fn((j: string) => { h.mesh.topo = j }),
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
  }

  // --- Loop state ---
  const loopState = {
    set_loops: fn((j: string) => { h.loop.list = j }),
    loops_json: fn(() => h.loop.list),
    set_current_loop: fn((j: string) => { h.loop.current = j }),
    current_loop_json: fn(() => h.loop.current || undefined),
    get_loop_by_slug_json: fn((slug: string) => {
      const arr = JSON.parse(h.loop.list) as { slug: string }[]
      const l = arr.find((x) => x.slug === slug)
      return l ? JSON.stringify(l) : undefined
    }),
    update_loop_local: fn(),
    add_run: fn(), set_runs: fn(), append_runs: fn(),
    update_run_status: fn(), runs_json: fn(() => '[]'), clear_runs: fn(),
    // Service async methods
    fetch_loops: fn().mockResolvedValue(JSON.stringify({ loops: [], total: 0 })),
    fetch_loop: fn().mockResolvedValue('{}'),
    create_loop: fn().mockResolvedValue('{}'),
    update_loop: fn().mockResolvedValue('{}'),
    delete_loop: fn().mockResolvedValue(undefined),
    enable_loop: fn().mockResolvedValue('{}'),
    disable_loop: fn().mockResolvedValue('{}'),
    trigger_loop: fn().mockResolvedValue('{}'),
    fetch_runs: fn().mockResolvedValue(JSON.stringify({ runs: [], total: 0 })),
    cancel_run: fn().mockResolvedValue(undefined),
  }

  // --- Git provider state ---
  const gitProviderState = {
    set_providers: fn(), providers_json: fn(() => '[]'),
    set_current_provider: fn(), current_provider_json: fn(),
    add_provider: fn(), update_provider: fn(), remove_provider: fn(),
    set_available_projects: fn(), available_projects_json: fn(() => '[]'),
  }

  // --- Repo state ---
  const repoState = {
    set_repositories: fn(), repositories_json: fn(() => h.repo.list),
    set_current_repo: fn(), current_repo_json: fn(() => h.repo.current || undefined),
    add_repository: fn(), update_repository: fn(), remove_repository: fn(),
    set_branches: fn(), branches_json: fn(() => h.repo.branches),
  }

  // --- Autopilot state ---
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
    getPodService: fn(() => podState),
    getTicketService: fn(() => ticketState),
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
    getOrgState: fn(() => orgState),
    getUserState: fn(() => userState),
    getGitProviderState: fn(() => gitProviderState),
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
    })),
    getExtensionService: fn(() => ({
      list_skill_registries: fn().mockResolvedValue('{"registries":[]}'),
      create_skill_registry: fn().mockResolvedValue('{}'),
      sync_skill_registry: fn().mockResolvedValue(undefined),
      toggle_skill_registry: fn().mockResolvedValue('{}'),
      delete_skill_registry: fn().mockResolvedValue(undefined),
      list_skill_registry_overrides: fn().mockResolvedValue('{"overrides":[]}'),
      list_market_skills: fn().mockResolvedValue('{"skills":[]}'),
      list_market_mcp_servers: fn().mockResolvedValue('{"servers":[]}'),
      list_repo_skills: fn().mockResolvedValue('{"installs":[]}'),
      install_skill_from_market: fn().mockResolvedValue('{}'),
      install_skill_from_github: fn().mockResolvedValue('{}'),
      update_skill: fn().mockResolvedValue('{}'),
      uninstall_skill: fn().mockResolvedValue(undefined),
      list_repo_mcp_servers: fn().mockResolvedValue('{"installs":[]}'),
      install_mcp_from_market: fn().mockResolvedValue('{}'),
      install_custom_mcp_server: fn().mockResolvedValue('{}'),
      update_mcp_server: fn().mockResolvedValue('{}'),
      uninstall_mcp_server: fn().mockResolvedValue(undefined),
    })),
    getInvitationService: fn(() => ({
      list: fn().mockResolvedValue('{"invitations":[]}'),
      create: fn().mockResolvedValue('{}'),
      revoke: fn().mockResolvedValue(undefined),
      resend: fn().mockResolvedValue(undefined),
      get_by_token: fn().mockResolvedValue('{}'),
      accept: fn().mockResolvedValue(undefined),
      list_pending: fn().mockResolvedValue('{"invitations":[]}'),
    })),
    getApiKeyService: fn(() => ({
      list: fn().mockResolvedValue('{"api_keys":[]}'),
      get: fn().mockResolvedValue('{}'),
      create: fn().mockResolvedValue('{}'),
      update: fn().mockResolvedValue('{}'),
      delete: fn().mockResolvedValue(undefined),
      revoke: fn().mockResolvedValue(undefined),
    })),
    getBindingService: fn(() => ({
      request_binding: fn().mockResolvedValue('{}'),
      accept_binding: fn().mockResolvedValue('{}'),
      reject_binding: fn().mockResolvedValue(undefined),
      request_scopes: fn().mockResolvedValue('{}'),
      approve_scopes: fn().mockResolvedValue('{}'),
      unbind: fn().mockResolvedValue(undefined),
      list_bindings: fn().mockResolvedValue('{"bindings":[]}'),
      get_pending_bindings: fn().mockResolvedValue('{"bindings":[]}'),
      get_bound_pods: fn().mockResolvedValue('{"pods":[]}'),
      check_binding: fn().mockResolvedValue('{}'),
    })),
    getMessageService: fn(() => ({
      send_message: fn().mockResolvedValue('{}'),
      get_messages: fn().mockResolvedValue('{"messages":[]}'),
      get_unread_count: fn().mockResolvedValue('{}'),
      get_message: fn().mockResolvedValue('{}'),
      mark_read: fn().mockResolvedValue(undefined),
      mark_all_read: fn().mockResolvedValue(undefined),
      get_conversation: fn().mockResolvedValue('{"messages":[]}'),
      get_sent_messages: fn().mockResolvedValue('{"messages":[]}'),
      get_dead_letters: fn().mockResolvedValue('{"entries":[]}'),
      replay_dead_letter: fn().mockResolvedValue(undefined),
    })),
    getNotificationService: fn(() => ({
      get_preferences: fn().mockResolvedValue('{"preferences":[]}'),
      set_preference: fn().mockResolvedValue('{}'),
    })),
    getPromoCodeService: fn(() => ({
      validate: fn().mockResolvedValue('{}'),
      redeem: fn().mockResolvedValue(undefined),
      get_history: fn().mockResolvedValue('{"codes":[]}'),
    })),
    getTokenUsageService: fn(() => ({
      get_dashboard: fn().mockResolvedValue('{}'),
    })),
    getSSOService: fn(() => ({
      discover: fn().mockResolvedValue('{}'),
      ldap_auth: fn().mockResolvedValue('{}'),
    })),
    getUserApiService: fn(() => ({
      get_me: fn().mockResolvedValue('{}'),
      get_organizations: fn().mockResolvedValue('{"organizations":[]}'),
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
      list_agent_credentials: fn().mockResolvedValue('{"credentials":[]}'),
      list_agent_credentials_for_agent: fn().mockResolvedValue('{"credentials":[]}'),
      create_agent_credential: fn().mockResolvedValue('{}'),
      get_agent_credential: fn().mockResolvedValue('{}'),
      update_agent_credential: fn().mockResolvedValue('{}'),
      delete_agent_credential: fn().mockResolvedValue(undefined),
      set_default_agent_credential: fn().mockResolvedValue(undefined),
      list_repo_providers: fn().mockResolvedValue('{"providers":[]}'),
      create_repo_provider: fn().mockResolvedValue('{}'),
      get_repo_provider: fn().mockResolvedValue('{}'),
      update_repo_provider: fn().mockResolvedValue('{}'),
      delete_repo_provider: fn().mockResolvedValue(undefined),
      set_default_repo_provider: fn().mockResolvedValue(undefined),
      test_repo_provider: fn().mockResolvedValue(undefined),
      list_provider_repositories: fn().mockResolvedValue('{"repositories":[]}'),
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
    })),
    getAgentService: fn(() => ({
      list_agents: fn().mockResolvedValue('{"builtin_agents":[],"custom_agents":[]}'),
      get_config_schema: fn().mockResolvedValue('{}'),
      list_user_configs: fn().mockResolvedValue('{"configs":[]}'),
      get_user_config: fn().mockResolvedValue('{}'),
      set_user_config: fn().mockResolvedValue('{}'),
      delete_user_config: fn().mockResolvedValue(undefined),
      get_agentpod_settings: fn().mockResolvedValue('{}'),
      update_agentpod_settings: fn().mockResolvedValue('{}'),
      list_providers: fn().mockResolvedValue('{"providers":[]}'),
      create_provider: fn().mockResolvedValue('{}'),
      update_provider: fn().mockResolvedValue('{}'),
      delete_provider: fn().mockResolvedValue(undefined),
      set_default_provider: fn().mockResolvedValue(undefined),
    })),
    getTicketRelationsService: fn(() => ({
      list_relations: fn().mockResolvedValue('{"relations":[]}'),
      create_relation: fn().mockResolvedValue('{}'),
      delete_relation: fn().mockResolvedValue(undefined),
      list_commits: fn().mockResolvedValue('{"commits":[]}'),
      link_commit: fn().mockResolvedValue('{}'),
      unlink_commit: fn().mockResolvedValue(undefined),
      list_merge_requests: fn().mockResolvedValue('{"merge_requests":[]}'),
      list_comments: fn().mockResolvedValue('{"comments":[]}'),
      create_comment: fn().mockResolvedValue('{}'),
      update_comment: fn().mockResolvedValue('{}'),
      delete_comment: fn().mockResolvedValue(undefined),
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
    })),
    getAuthApiService: fn(() => ({
      register: fn().mockResolvedValue('{}'),
      verify_email: fn().mockResolvedValue('{}'),
      resend_verification: fn().mockResolvedValue('{}'),
      forgot_password: fn().mockResolvedValue('{}'),
      reset_password: fn().mockResolvedValue('{}'),
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

// ---------------------------------------------------------------------------
// Browser mocks
// ---------------------------------------------------------------------------
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

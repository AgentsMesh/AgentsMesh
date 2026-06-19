import { describe, it, expect, beforeEach, vi } from 'vitest'
import { create, toBinary, fromBinary } from '@bufbuild/protobuf'
import {
  ListRunnersResponseSchema,
  ListAvailableRunnersResponseSchema,
  GetRunnerResponseSchema,
  RunnerSchema,
  RunnerTokenSchema,
  DeleteRunnerResponseSchema,
  type Runner as ProtoRunner,
} from '@proto/runner_api/v1/runner_pb'
import {
  ReplaceCachedRunnersRequestSchema,
  ReplaceAvailableRunnersRequestSchema,
  SetCurrentRunnerRequestSchema,
  PatchCachedRunnerRequestSchema,
  RemoveCachedRunnerRequestSchema,
} from '@proto/runner_state/v1/runner_state_pb'
import { getRunnerStatusInfo, formatHostInfo, Runner, useRunnerStore } from '../runner'

let mockRunnersList: Runner[] = []
let mockAvailableRunners: Runner[] = []
let mockCurrentRunner: Runner | null = null

function protoRunnerToData(r: ProtoRunner): Runner {
  const out: Runner = {
    id: Number(r.id),
    node_id: r.nodeId,
    status: r.status as Runner['status'],
    last_heartbeat: r.lastHeartbeat,
    current_pods: r.currentPods,
    max_concurrent_pods: r.maxConcurrentPods,
    is_enabled: r.isEnabled,
    created_at: r.createdAt,
    updated_at: r.updatedAt,
  }
  if (r.description) out.description = r.description
  if (r.runnerVersion) out.runner_version = r.runnerVersion
  if (r.visibility) out.visibility = r.visibility as Runner['visibility']
  return out
}

const mockService = {
  replace_cached_runners: vi.fn((bytes: Uint8Array) => {
    const req = fromBinary(ReplaceCachedRunnersRequestSchema, bytes)
    mockRunnersList = req.runners.map(protoRunnerToData)
  }),
  replace_available_runners: vi.fn((bytes: Uint8Array) => {
    const req = fromBinary(ReplaceAvailableRunnersRequestSchema, bytes)
    mockAvailableRunners = req.runners.map(protoRunnerToData)
  }),
  set_current_runner_proto: vi.fn((bytes: Uint8Array) => {
    const req = fromBinary(SetCurrentRunnerRequestSchema, bytes)
    mockCurrentRunner = req.runner ? protoRunnerToData(req.runner) : null
  }),
  patch_cached_runner: vi.fn((bytes: Uint8Array) => {
    const req = fromBinary(PatchCachedRunnerRequestSchema, bytes)
    if (req.runner) {
      const r = protoRunnerToData(req.runner)
      const idx = mockRunnersList.findIndex((x) => x.id === r.id)
      if (idx >= 0) mockRunnersList[idx] = r
      else mockRunnersList.push(r)
    }
  }),
  remove_cached_runner: vi.fn((bytes: Uint8Array) => {
    const req = fromBinary(RemoveCachedRunnerRequestSchema, bytes)
    const id = Number(req.runnerId)
    mockRunnersList = mockRunnersList.filter((x) => x.id !== id)
    mockAvailableRunners = mockAvailableRunners.filter((x) => x.id !== id)
  }),
  // Fetch→state (B): decode wire response into the backing list.
  apply_fetched_runners: vi.fn((bytes: Uint8Array) => {
    mockRunnersList = fromBinary(ListRunnersResponseSchema, bytes).items.map(protoRunnerToData)
  }),
  apply_fetched_available_runners: vi.fn((bytes: Uint8Array) => {
    mockAvailableRunners = fromBinary(ListAvailableRunnersResponseSchema, bytes).items.map(protoRunnerToData)
  }),
  apply_fetched_current_runner: vi.fn((bytes: Uint8Array) => {
    const r = fromBinary(GetRunnerResponseSchema, bytes).runner
    mockCurrentRunner = r ? protoRunnerToData(r) : null
  }),
  runners_json: vi.fn(() => JSON.stringify(mockRunnersList)),
  available_runners_json: vi.fn(() => JSON.stringify(mockAvailableRunners)),
  current_runner_json: vi.fn(() => (mockCurrentRunner ? JSON.stringify(mockCurrentRunner) : null)),
  // Read side (B): re-encode the backing list into state proto bytes.
  runners_bytes: vi.fn(() => toBinary(ReplaceCachedRunnersRequestSchema,
    create(ReplaceCachedRunnersRequestSchema, { runners: mockRunnersList.map(toProtoRunner) }))),
  available_runners_bytes: vi.fn(() => toBinary(ReplaceAvailableRunnersRequestSchema,
    create(ReplaceAvailableRunnersRequestSchema, { runners: mockAvailableRunners.map(toProtoRunner) }))),
  current_runner_bytes: vi.fn(() => mockCurrentRunner
    ? toBinary(RunnerSchema, toProtoRunner(mockCurrentRunner)) : new Uint8Array()),
  // Connect-RPC binary lane
  listRunnersConnect: vi.fn().mockResolvedValue(new Uint8Array()),
  listAvailableRunnersConnect: vi.fn().mockResolvedValue(new Uint8Array()),
  getRunnerConnect: vi.fn().mockResolvedValue(new Uint8Array()),
  updateRunnerConnect: vi.fn().mockResolvedValue(new Uint8Array()),
  deleteRunnerConnect: vi.fn().mockResolvedValue(new Uint8Array()),
  createRunnerTokenConnect: vi.fn().mockResolvedValue(new Uint8Array()),
}

vi.mock('@/lib/wasm-core', () => ({
  getRunnerService: () => mockService,
  getRunnerState: () => mockService,
}))

vi.mock('@/stores/auth', () => ({
  readCurrentOrg: () => ({ slug: 'test-org' }),
}))

const createMockRunner = (overrides: Partial<Runner> = {}): Runner => ({
  id: 1,
  node_id: 'test-runner',
  status: 'online',
  is_enabled: true,
  current_pods: 0,
  max_concurrent_pods: 5,
  last_heartbeat: '2024-01-01T00:00:00Z',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
})

function toProtoRunner(r: Runner): ProtoRunner {
  return create(RunnerSchema, {
    id: BigInt(r.id),
    nodeId: r.node_id,
    status: r.status,
    isEnabled: r.is_enabled,
    currentPods: r.current_pods,
    maxConcurrentPods: r.max_concurrent_pods,
    lastHeartbeat: r.last_heartbeat ?? '',
    createdAt: r.created_at ?? '',
    updatedAt: r.updated_at ?? '',
    description: r.description ?? '',
  })
}

function mockListRunners(runners: Runner[]) {
  const resp = create(ListRunnersResponseSchema, {
    items: runners.map(toProtoRunner),
    total: BigInt(runners.length),
    limit: 0,
    offset: 0,
  })
  mockService.listRunnersConnect.mockResolvedValue(toBinary(ListRunnersResponseSchema, resp))
}

function mockListAvailable(runners: Runner[]) {
  const resp = create(ListAvailableRunnersResponseSchema, {
    items: runners.map(toProtoRunner),
    total: BigInt(runners.length),
  })
  mockService.listAvailableRunnersConnect.mockResolvedValue(toBinary(ListAvailableRunnersResponseSchema, resp))
}

function mockGetRunner(runner: Runner | null) {
  const resp = create(GetRunnerResponseSchema, {
    runner: runner ? toProtoRunner(runner) : undefined,
    relayConnections: [],
  })
  mockService.getRunnerConnect.mockResolvedValue(toBinary(GetRunnerResponseSchema, resp))
}

function mockUpdateRunner(runner: Runner) {
  mockService.updateRunnerConnect.mockResolvedValue(toBinary(RunnerSchema, toProtoRunner(runner)))
}

function mockDeleteRunner() {
  mockService.deleteRunnerConnect.mockResolvedValue(
    toBinary(DeleteRunnerResponseSchema, create(DeleteRunnerResponseSchema, {})),
  )
}

function mockCreateToken(token: string) {
  const t = create(RunnerTokenSchema, {
    id: BigInt(1),
    name: 'test',
    token,
    expiresAt: '2024-12-31T23:59:59Z',
    createdAt: '2024-01-01T00:00:00Z',
  })
  mockService.createRunnerTokenConnect.mockResolvedValue(toBinary(RunnerTokenSchema, t))
}

const getRunners = (): Runner[] => JSON.parse(mockService.runners_json())
const getAvailableRunners = (): Runner[] => JSON.parse(mockService.available_runners_json())
const getCurrentRunner = (): Runner | null => {
  const raw = mockService.current_runner_json()
  return raw ? JSON.parse(raw) : null
}

beforeEach(() => {
  mockRunnersList = []
  mockAvailableRunners = []
  mockCurrentRunner = null
  useRunnerStore.setState({
    _tick: 0,
    loading: false,
    error: null,
  })
  vi.clearAllMocks()
})

describe('Runner Store Actions', () => {
  describe('fetchRunners', () => {
    it('should fetch runners successfully', async () => {
      const runners = [createMockRunner({ id: 1 }), createMockRunner({ id: 2, node_id: 'runner-2' })]
      mockListRunners(runners)

      await useRunnerStore.getState().fetchRunners()

      expect(mockService.listRunnersConnect).toHaveBeenCalled()
      expect(getRunners()).toHaveLength(2)
      expect(useRunnerStore.getState().loading).toBe(false)
    })

    it('should filter runners by status', async () => {
      mockListRunners([])

      await useRunnerStore.getState().fetchRunners('online')

      expect(mockService.listRunnersConnect).toHaveBeenCalled()
    })

    it('should handle fetch error', async () => {
      mockService.listRunnersConnect.mockRejectedValue(new Error('Network error'))

      await useRunnerStore.getState().fetchRunners()

      expect(useRunnerStore.getState().error).toBe('Network error')
      expect(useRunnerStore.getState().loading).toBe(false)
    })
  })

  describe('fetchAvailableRunners', () => {
    it('should fetch available runners successfully', async () => {
      const runners = [createMockRunner()]
      mockListAvailable(runners)

      await useRunnerStore.getState().fetchAvailableRunners()

      expect(getAvailableRunners()).toHaveLength(1)
    })

    it('should handle fetch error', async () => {
      mockService.listAvailableRunnersConnect.mockRejectedValue(new Error('Network error'))

      await useRunnerStore.getState().fetchAvailableRunners()

      expect(useRunnerStore.getState().error).toBe('Network error')
    })
  })

  describe('fetchRunner', () => {
    it('should fetch single runner successfully', async () => {
      const runner = createMockRunner()
      mockGetRunner(runner)

      await useRunnerStore.getState().fetchRunner(1)

      expect(mockService.getRunnerConnect).toHaveBeenCalled()
      expect(getCurrentRunner()?.id).toBe(1)
    })

    it('should handle fetch error', async () => {
      mockService.getRunnerConnect.mockRejectedValue(new Error('Not found'))

      await useRunnerStore.getState().fetchRunner(999)

      expect(useRunnerStore.getState().error).toBe('Not found')
    })
  })

  describe('updateRunner', () => {
    it('should update runner successfully', async () => {
      const existingRunner = createMockRunner()
      const updatedRunner = { ...existingRunner, description: 'Updated description' }

      mockListRunners([existingRunner])
      await useRunnerStore.getState().fetchRunners()

      mockUpdateRunner(updatedRunner)

      const result = await useRunnerStore.getState().updateRunner(1, { description: 'Updated description' })

      expect(result.description).toBe('Updated description')
      expect(getRunners()[0].description).toBe('Updated description')
    })

    it('should handle update error', async () => {
      mockService.updateRunnerConnect.mockRejectedValue(new Error('Update failed'))

      await expect(useRunnerStore.getState().updateRunner(1, { description: 'test' })).rejects.toThrow()
      expect(useRunnerStore.getState().error).toBe('Update failed')
    })
  })

  describe('deleteRunner', () => {
    it('should delete runner successfully', async () => {
      const runner = createMockRunner()
      mockListRunners([runner])
      await useRunnerStore.getState().fetchRunners()
      mockListAvailable([runner])
      await useRunnerStore.getState().fetchAvailableRunners()

      mockDeleteRunner()

      await useRunnerStore.getState().deleteRunner(1)

      expect(getRunners()).toHaveLength(0)
      expect(getAvailableRunners()).toHaveLength(0)
    })

    it('should handle delete error', async () => {
      mockService.deleteRunnerConnect.mockRejectedValue(new Error('Delete failed'))

      await expect(useRunnerStore.getState().deleteRunner(1)).rejects.toThrow()
    })
  })

  describe('createToken', () => {
    it('should create token successfully', async () => {
      mockCreateToken('new-token-123')

      const token = await useRunnerStore.getState().createToken()

      expect(token).toBe('new-token-123')
      expect(mockService.createRunnerTokenConnect).toHaveBeenCalled()
    })

    it('should handle create token error', async () => {
      mockService.createRunnerTokenConnect.mockRejectedValue(new Error('Failed to create token'))

      await expect(useRunnerStore.getState().createToken()).rejects.toThrow()
      expect(useRunnerStore.getState().error).toBe('Failed to create token')
    })
  })

  describe('setCurrentRunner', () => {
    it('should set current runner', () => {
      const runner = createMockRunner()
      useRunnerStore.getState().setCurrentRunner(runner)
      expect(getCurrentRunner()).toEqual(runner)
    })

    it('should clear current runner', () => {
      mockCurrentRunner = createMockRunner()
      useRunnerStore.getState().setCurrentRunner(null)
      expect(getCurrentRunner()).toBeNull()
    })

  })

  describe('clearError', () => {
    it('should clear error', () => {
      useRunnerStore.setState({ error: 'Some error' })
      useRunnerStore.getState().clearError()

      expect(useRunnerStore.getState().error).toBeNull()
    })
  })
})

describe('Runner Store Helper Functions', () => {
  describe('getRunnerStatusInfo', () => {
    it('should return correct info for online status', () => {
      const info = getRunnerStatusInfo('online')
      expect(info).toEqual({
        label: 'Online',
        color: 'text-green-600 dark:text-green-400',
        dotColor: 'bg-green-500',
      })
    })

    it('should return correct info for offline status', () => {
      const info = getRunnerStatusInfo('offline')
      expect(info).toEqual({
        label: 'Offline',
        color: 'text-gray-500 dark:text-gray-400',
        dotColor: 'bg-gray-400',
      })
    })

    it('should return correct info for maintenance status', () => {
      const info = getRunnerStatusInfo('maintenance')
      expect(info).toEqual({
        label: 'Maintenance',
        color: 'text-yellow-600 dark:text-yellow-400',
        dotColor: 'bg-yellow-500',
      })
    })

    it('should return correct info for busy status', () => {
      const info = getRunnerStatusInfo('busy')
      expect(info).toEqual({
        label: 'Busy',
        color: 'text-orange-600 dark:text-orange-400',
        dotColor: 'bg-orange-500',
      })
    })
  })

  describe('formatHostInfo', () => {
    it('should return "Unknown" for undefined host_info', () => {
      expect(formatHostInfo(undefined)).toBe('Unknown')
    })

    it('should return "Unknown" for empty host_info', () => {
      expect(formatHostInfo({})).toBe('Unknown')
    })

    it('should format os only', () => {
      const result = formatHostInfo({ os: 'linux' })
      expect(result).toBe('linux')
    })

    it('should format arch only', () => {
      const result = formatHostInfo({ arch: 'amd64' })
      expect(result).toBe('amd64')
    })

    it('should format cpu_cores only', () => {
      const result = formatHostInfo({ cpu_cores: 8 })
      expect(result).toBe('8 cores')
    })

    it('should format memory only', () => {
      const result = formatHostInfo({ memory: 17179869184 })
      expect(result).toBe('16.0GB RAM')
    })

    it('should format all fields', () => {
      const result = formatHostInfo({
        os: 'linux',
        arch: 'amd64',
        cpu_cores: 8,
        memory: 17179869184,
      })
      expect(result).toBe('linux / amd64 / 8 cores / 16.0GB RAM')
    })

    it('should format partial fields', () => {
      const result = formatHostInfo({
        os: 'darwin',
        cpu_cores: 4,
      })
      expect(result).toBe('darwin / 4 cores')
    })

    it('should handle small memory values', () => {
      const result = formatHostInfo({ memory: 1073741824 })
      expect(result).toBe('1.0GB RAM')
    })

    it('should handle large memory values', () => {
      const result = formatHostInfo({ memory: 137438953472 })
      expect(result).toBe('128.0GB RAM')
    })
  })
})

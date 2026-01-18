import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getRunnerStatusInfo, canAcceptPods, formatHostInfo, Runner, useRunnerStore } from '../runner'

// Mock the API client
vi.mock('@/lib/api/client', () => ({
  runnerApi: {
    list: vi.fn(),
    listAvailable: vi.fn(),
    get: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    regenerateAuthToken: vi.fn(),
    createToken: vi.fn(),
  },
}))

import { runnerApi } from '@/lib/api/client'

// Helper to create mock runner
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

// Reset store before each test
beforeEach(() => {
  useRunnerStore.setState({
    runners: [],
    availableRunners: [],
    currentRunner: null,
    loading: false,
    error: null,
  })
  vi.clearAllMocks()
})

describe('Runner Store Actions', () => {
  describe('fetchRunners', () => {
    it('should fetch runners successfully', async () => {
      const mockRunners = [createMockRunner({ id: 1 }), createMockRunner({ id: 2, node_id: 'runner-2' })]
      vi.mocked(runnerApi.list).mockResolvedValue({ runners: mockRunners })

      await useRunnerStore.getState().fetchRunners()

      expect(runnerApi.list).toHaveBeenCalled()
      expect(useRunnerStore.getState().runners).toEqual(mockRunners)
      expect(useRunnerStore.getState().loading).toBe(false)
    })

    it('should filter runners by status', async () => {
      vi.mocked(runnerApi.list).mockResolvedValue({ runners: [] })

      await useRunnerStore.getState().fetchRunners('online')

      expect(runnerApi.list).toHaveBeenCalledWith('online')
    })

    it('should handle fetch error', async () => {
      vi.mocked(runnerApi.list).mockRejectedValue(new Error('Network error'))

      await useRunnerStore.getState().fetchRunners()

      expect(useRunnerStore.getState().error).toBe('Network error')
      expect(useRunnerStore.getState().loading).toBe(false)
    })
  })

  describe('fetchAvailableRunners', () => {
    it('should fetch available runners successfully', async () => {
      const mockRunners = [createMockRunner()]
      vi.mocked(runnerApi.listAvailable).mockResolvedValue({ runners: mockRunners })

      await useRunnerStore.getState().fetchAvailableRunners()

      expect(useRunnerStore.getState().availableRunners).toEqual(mockRunners)
    })

    it('should handle fetch error', async () => {
      vi.mocked(runnerApi.listAvailable).mockRejectedValue(new Error('Network error'))

      await useRunnerStore.getState().fetchAvailableRunners()

      expect(useRunnerStore.getState().error).toBe('Network error')
    })
  })

  describe('fetchRunner', () => {
    it('should fetch single runner successfully', async () => {
      const mockRunner = createMockRunner()
      vi.mocked(runnerApi.get).mockResolvedValue({ runner: mockRunner })

      await useRunnerStore.getState().fetchRunner(1)

      expect(runnerApi.get).toHaveBeenCalledWith(1)
      expect(useRunnerStore.getState().currentRunner).toEqual(mockRunner)
    })

    it('should handle fetch error', async () => {
      vi.mocked(runnerApi.get).mockRejectedValue(new Error('Not found'))

      await useRunnerStore.getState().fetchRunner(999)

      expect(useRunnerStore.getState().error).toBe('Not found')
    })
  })

  describe('updateRunner', () => {
    it('should update runner successfully', async () => {
      const existingRunner = createMockRunner()
      const updatedRunner = { ...existingRunner, description: 'Updated description' }

      useRunnerStore.setState({
        runners: [existingRunner],
        availableRunners: [existingRunner],
        currentRunner: existingRunner,
      })
      vi.mocked(runnerApi.update).mockResolvedValue({ runner: updatedRunner })

      const result = await useRunnerStore.getState().updateRunner(1, { description: 'Updated description' })

      expect(result).toEqual(updatedRunner)
      expect(useRunnerStore.getState().runners[0].description).toBe('Updated description')
      expect(useRunnerStore.getState().currentRunner?.description).toBe('Updated description')
    })

    it('should handle update error', async () => {
      vi.mocked(runnerApi.update).mockRejectedValue(new Error('Update failed'))

      await expect(useRunnerStore.getState().updateRunner(1, { description: 'test' })).rejects.toThrow()
      expect(useRunnerStore.getState().error).toBe('Update failed')
    })
  })

  describe('deleteRunner', () => {
    it('should delete runner successfully', async () => {
      const runner = createMockRunner()
      useRunnerStore.setState({
        runners: [runner],
        availableRunners: [runner],
        currentRunner: runner,
      })
      vi.mocked(runnerApi.delete).mockResolvedValue({ message: 'Deleted' })

      await useRunnerStore.getState().deleteRunner(1)

      expect(useRunnerStore.getState().runners).toHaveLength(0)
      expect(useRunnerStore.getState().availableRunners).toHaveLength(0)
      expect(useRunnerStore.getState().currentRunner).toBeNull()
    })

    it('should handle delete error', async () => {
      vi.mocked(runnerApi.delete).mockRejectedValue(new Error('Delete failed'))

      await expect(useRunnerStore.getState().deleteRunner(1)).rejects.toThrow()
    })
  })

  describe('regenerateAuthToken', () => {
    it('should regenerate auth token successfully', async () => {
      vi.mocked(runnerApi.regenerateAuthToken).mockResolvedValue({ auth_token: 'new-token', message: 'Token regenerated' })

      const token = await useRunnerStore.getState().regenerateAuthToken(1)

      expect(token).toBe('new-token')
    })

    it('should handle error', async () => {
      vi.mocked(runnerApi.regenerateAuthToken).mockRejectedValue(new Error('Failed'))

      await expect(useRunnerStore.getState().regenerateAuthToken(1)).rejects.toThrow()
    })
  })

  describe('createToken', () => {
    it('should create token successfully', async () => {
      vi.mocked(runnerApi.createToken).mockResolvedValue({ token: 'new-token-123', message: 'Token created' })

      const token = await useRunnerStore.getState().createToken()

      expect(token).toBe('new-token-123')
      expect(runnerApi.createToken).toHaveBeenCalled()
    })

    it('should handle create token error', async () => {
      vi.mocked(runnerApi.createToken).mockRejectedValue(new Error('Failed to create token'))

      await expect(useRunnerStore.getState().createToken()).rejects.toThrow()
      expect(useRunnerStore.getState().error).toBe('Failed to create token')
    })
  })

  describe('setCurrentRunner and updateRunnerStatus', () => {
    it('should set current runner', () => {
      const runner = createMockRunner()
      useRunnerStore.getState().setCurrentRunner(runner)

      expect(useRunnerStore.getState().currentRunner).toEqual(runner)
    })

    it('should clear current runner', () => {
      useRunnerStore.setState({ currentRunner: createMockRunner() })
      useRunnerStore.getState().setCurrentRunner(null)

      expect(useRunnerStore.getState().currentRunner).toBeNull()
    })

    it('should update runner status to offline', () => {
      const runner = createMockRunner({ status: 'online' })
      useRunnerStore.setState({
        runners: [runner],
        availableRunners: [runner],
        currentRunner: runner,
      })

      useRunnerStore.getState().updateRunnerStatus(1, 'offline')

      expect(useRunnerStore.getState().runners[0].status).toBe('offline')
      expect(useRunnerStore.getState().availableRunners).toHaveLength(0)
      expect(useRunnerStore.getState().currentRunner?.status).toBe('offline')
    })

    it('should keep available runners when status is online', () => {
      const runner = createMockRunner()
      useRunnerStore.setState({
        runners: [runner],
        availableRunners: [runner],
      })

      useRunnerStore.getState().updateRunnerStatus(1, 'online')

      expect(useRunnerStore.getState().availableRunners).toHaveLength(1)
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

  describe('canAcceptPods', () => {
    const createRunner = (overrides: Partial<Runner> = {}): Runner => ({
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

    it('should return true for online runner with available slots', () => {
      const runner = createRunner({ status: 'online', current_pods: 2, max_concurrent_pods: 5 })
      expect(canAcceptPods(runner)).toBe(true)
    })

    it('should return false for offline runner', () => {
      const runner = createRunner({ status: 'offline' })
      expect(canAcceptPods(runner)).toBe(false)
    })

    it('should return false for maintenance runner', () => {
      const runner = createRunner({ status: 'maintenance' })
      expect(canAcceptPods(runner)).toBe(false)
    })

    it('should return false for busy runner', () => {
      const runner = createRunner({ status: 'busy' })
      expect(canAcceptPods(runner)).toBe(false)
    })

    it('should return false when at max capacity', () => {
      const runner = createRunner({ status: 'online', current_pods: 5, max_concurrent_pods: 5 })
      expect(canAcceptPods(runner)).toBe(false)
    })

    it('should return false when over max capacity', () => {
      const runner = createRunner({ status: 'online', current_pods: 6, max_concurrent_pods: 5 })
      expect(canAcceptPods(runner)).toBe(false)
    })

    it('should return true with zero current pods', () => {
      const runner = createRunner({ status: 'online', current_pods: 0, max_concurrent_pods: 5 })
      expect(canAcceptPods(runner)).toBe(true)
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
      // 16GB in bytes
      const result = formatHostInfo({ memory: 17179869184 })
      expect(result).toBe('16.0GB RAM')
    })

    it('should format all fields', () => {
      const result = formatHostInfo({
        os: 'linux',
        arch: 'amd64',
        cpu_cores: 8,
        memory: 17179869184, // 16GB
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
      // 1GB in bytes
      const result = formatHostInfo({ memory: 1073741824 })
      expect(result).toBe('1.0GB RAM')
    })

    it('should handle large memory values', () => {
      // 128GB in bytes
      const result = formatHostInfo({ memory: 137438953472 })
      expect(result).toBe('128.0GB RAM')
    })
  })
})

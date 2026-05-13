import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getRunnerStatusInfo, canAcceptPods, formatHostInfo, Runner } from '../runner'

describe('Runner Store Helper Functions', () => {
  beforeEach(() => { vi.clearAllMocks() })

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

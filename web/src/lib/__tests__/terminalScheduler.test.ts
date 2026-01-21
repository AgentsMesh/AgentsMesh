import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { TerminalWriteScheduler } from '../terminalScheduler'

// Mock requestAnimationFrame and cancelAnimationFrame
let rafCallback: FrameRequestCallback | null = null
let rafId = 0

beforeEach(() => {
  rafCallback = null
  rafId = 0

  vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
    rafCallback = callback
    rafId++
    return rafId
  })

  vi.stubGlobal('cancelAnimationFrame', vi.fn())
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// Helper to trigger RAF callback
function flushRAF() {
  if (rafCallback) {
    const cb = rafCallback
    rafCallback = null
    cb(performance.now())
  }
}

// Mock Terminal
function createMockTerminal() {
  return {
    write: vi.fn(),
  }
}

describe('TerminalWriteScheduler', () => {
  describe('constructor', () => {
    it('should create a new instance', () => {
      const scheduler = new TerminalWriteScheduler()
      expect(scheduler).toBeInstanceOf(TerminalWriteScheduler)
    })
  })

  describe('attach', () => {
    it('should attach a terminal', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()

      // Should not throw
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])
    })

    it('should allow reattaching a different terminal', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal1 = createMockTerminal()
      const mockTerminal2 = createMockTerminal()

      scheduler.attach(mockTerminal1 as unknown as Parameters<typeof scheduler.attach>[0])
      scheduler.attach(mockTerminal2 as unknown as Parameters<typeof scheduler.attach>[0])

      // Schedule and flush
      scheduler.schedule(new Uint8Array([1, 2, 3]))
      flushRAF()

      // Should write to the second terminal
      expect(mockTerminal1.write).not.toHaveBeenCalled()
      expect(mockTerminal2.write).toHaveBeenCalled()
    })
  })

  describe('schedule', () => {
    it('should schedule data for writing', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      const data = new Uint8Array([1, 2, 3])
      scheduler.schedule(data)

      // Should request animation frame
      expect(rafCallback).not.toBeNull()
    })

    it('should only request one animation frame for multiple schedules', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1]))
      const firstRafId = rafId

      scheduler.schedule(new Uint8Array([2]))
      scheduler.schedule(new Uint8Array([3]))

      // Should still be the same RAF id (only one request)
      expect(rafId).toBe(firstRafId)
    })

    it('should combine multiple scheduled data into one write', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1, 2]))
      scheduler.schedule(new Uint8Array([3, 4]))
      scheduler.schedule(new Uint8Array([5]))

      flushRAF()

      expect(mockTerminal.write).toHaveBeenCalledTimes(1)
      const writtenData = mockTerminal.write.mock.calls[0][0] as Uint8Array
      expect(Array.from(writtenData)).toEqual([1, 2, 3, 4, 5])
    })

    it('should handle empty data', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([]))
      scheduler.schedule(new Uint8Array([1, 2]))
      scheduler.schedule(new Uint8Array([]))

      flushRAF()

      expect(mockTerminal.write).toHaveBeenCalledTimes(1)
      const writtenData = mockTerminal.write.mock.calls[0][0] as Uint8Array
      expect(Array.from(writtenData)).toEqual([1, 2])
    })

    it('should work without attached terminal', () => {
      const scheduler = new TerminalWriteScheduler()

      // Should not throw
      scheduler.schedule(new Uint8Array([1, 2, 3]))
      flushRAF()
    })

    it('should allow scheduling after flush', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      // First batch
      scheduler.schedule(new Uint8Array([1]))
      flushRAF()

      // Second batch
      scheduler.schedule(new Uint8Array([2]))
      flushRAF()

      expect(mockTerminal.write).toHaveBeenCalledTimes(2)
    })
  })

  describe('flush (internal)', () => {
    it('should not write if no data is pending', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      // Manually trigger flush without scheduling
      flushRAF()

      expect(mockTerminal.write).not.toHaveBeenCalled()
    })

    it('should clear pending data after flush', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1, 2, 3]))
      flushRAF()

      // Schedule new data
      scheduler.schedule(new Uint8Array([4, 5]))
      flushRAF()

      // Second write should only contain the new data
      expect(mockTerminal.write).toHaveBeenCalledTimes(2)
      const secondWriteData = mockTerminal.write.mock.calls[1][0] as Uint8Array
      expect(Array.from(secondWriteData)).toEqual([4, 5])
    })

    it('should reset rafId after flush', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1]))
      const firstRafId = rafId

      flushRAF()

      // Schedule again - should get a new RAF
      scheduler.schedule(new Uint8Array([2]))

      expect(rafId).toBe(firstRafId + 1)
    })
  })

  describe('dispose', () => {
    it('should cancel pending animation frame', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1, 2, 3]))
      scheduler.dispose()

      expect(cancelAnimationFrame).toHaveBeenCalledWith(rafId)
    })

    it('should not call cancelAnimationFrame if no RAF is pending', () => {
      const scheduler = new TerminalWriteScheduler()
      scheduler.dispose()

      expect(cancelAnimationFrame).not.toHaveBeenCalled()
    })

    it('should clear pending data', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1, 2, 3]))
      scheduler.dispose()

      // Re-attach and schedule new data
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])
      scheduler.schedule(new Uint8Array([4, 5]))
      flushRAF()

      // Should only contain the new data
      const writtenData = mockTerminal.write.mock.calls[0][0] as Uint8Array
      expect(Array.from(writtenData)).toEqual([4, 5])
    })

    it('should clear terminal reference', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1]))
      scheduler.dispose()

      // Schedule after dispose
      scheduler.schedule(new Uint8Array([2]))
      flushRAF()

      // Terminal should not receive any writes after dispose
      expect(mockTerminal.write).not.toHaveBeenCalled()
    })

    it('should be safe to call multiple times', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      scheduler.schedule(new Uint8Array([1]))

      // Should not throw
      scheduler.dispose()
      scheduler.dispose()
      scheduler.dispose()
    })
  })

  describe('data integrity', () => {
    it('should preserve byte order', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      // Schedule data in specific order
      for (let i = 0; i < 10; i++) {
        scheduler.schedule(new Uint8Array([i]))
      }

      flushRAF()

      const writtenData = mockTerminal.write.mock.calls[0][0] as Uint8Array
      expect(Array.from(writtenData)).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8, 9])
    })

    it('should handle large data', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      // Create large data chunks
      const chunk1 = new Uint8Array(1000).fill(1)
      const chunk2 = new Uint8Array(2000).fill(2)
      const chunk3 = new Uint8Array(3000).fill(3)

      scheduler.schedule(chunk1)
      scheduler.schedule(chunk2)
      scheduler.schedule(chunk3)

      flushRAF()

      const writtenData = mockTerminal.write.mock.calls[0][0] as Uint8Array
      expect(writtenData.length).toBe(6000)

      // Verify data integrity
      expect(writtenData.slice(0, 1000).every(b => b === 1)).toBe(true)
      expect(writtenData.slice(1000, 3000).every(b => b === 2)).toBe(true)
      expect(writtenData.slice(3000, 6000).every(b => b === 3)).toBe(true)
    })

    it('should handle binary data correctly', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      // Include all byte values
      const allBytes = new Uint8Array(256)
      for (let i = 0; i < 256; i++) {
        allBytes[i] = i
      }

      scheduler.schedule(allBytes)
      flushRAF()

      const writtenData = mockTerminal.write.mock.calls[0][0] as Uint8Array
      expect(writtenData.length).toBe(256)
      for (let i = 0; i < 256; i++) {
        expect(writtenData[i]).toBe(i)
      }
    })
  })

  describe('edge cases', () => {
    it('should handle rapid schedule and dispose cycles', () => {
      const scheduler = new TerminalWriteScheduler()
      const mockTerminal = createMockTerminal()

      for (let i = 0; i < 10; i++) {
        scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])
        scheduler.schedule(new Uint8Array([i]))
        scheduler.dispose()
      }

      // Should not throw and terminal should not receive writes
      expect(mockTerminal.write).not.toHaveBeenCalled()
    })

    it('should handle schedule before attach', () => {
      const scheduler = new TerminalWriteScheduler()

      scheduler.schedule(new Uint8Array([1, 2, 3]))

      const mockTerminal = createMockTerminal()
      scheduler.attach(mockTerminal as unknown as Parameters<typeof scheduler.attach>[0])

      flushRAF()

      // Data scheduled before attach should still be written
      expect(mockTerminal.write).toHaveBeenCalled()
    })
  })
})

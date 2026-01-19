import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getStatusInfo, getPriorityInfo, getTypeInfo, useTicketStore } from '../ticket'

// Mock the API client
vi.mock('@/lib/api', () => ({
  ticketApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    updateStatus: vi.fn(),
    listLabels: vi.fn(),
    createLabel: vi.fn(),
    deleteLabel: vi.fn(),
  },
}))

import { ticketApi } from '@/lib/api'

// Reset store before each test
beforeEach(() => {
  useTicketStore.setState({
    tickets: [],
    currentTicket: null,
    labels: [],
    filters: {},
    loading: false,
    error: null,
    totalCount: 0,
  })
  vi.clearAllMocks()
})

// Helper to create a mock ticket with required fields
const createMockTicket = (overrides: Partial<{
  id: number;
  number: number;
  identifier: string;
  title: string;
  status: 'backlog' | 'todo' | 'in_progress' | 'in_review' | 'done' | 'cancelled';
  type: 'task' | 'bug' | 'feature' | 'improvement' | 'epic' | 'subtask' | 'story';
  priority: 'none' | 'low' | 'medium' | 'high' | 'urgent';
  created_at: string;
  updated_at: string;
}> = {}) => ({
  id: 1,
  number: 1,
  identifier: 'TKT-1',
  title: 'Test Ticket',
  status: 'todo' as const,
  type: 'task' as const,
  priority: 'medium' as const,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
})

describe('Ticket Store Actions', () => {
  describe('fetchTickets', () => {
    it('should fetch tickets successfully', async () => {
      const mockTickets = [
        createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Ticket 1', status: 'todo' }),
        createMockTicket({ id: 2, number: 2, identifier: 'TKT-2', title: 'Ticket 2', status: 'in_progress' }),
      ]
      vi.mocked(ticketApi.list).mockResolvedValue({ tickets: mockTickets, total: 2 })

      await useTicketStore.getState().fetchTickets()

      expect(ticketApi.list).toHaveBeenCalled()
      expect(useTicketStore.getState().tickets).toEqual(mockTickets)
      expect(useTicketStore.getState().totalCount).toBe(2)
      expect(useTicketStore.getState().loading).toBe(false)
    })

    it('should handle fetch error', async () => {
      vi.mocked(ticketApi.list).mockRejectedValue(new Error('Network error'))

      await useTicketStore.getState().fetchTickets()

      expect(useTicketStore.getState().error).toBe('Network error')
      expect(useTicketStore.getState().loading).toBe(false)
    })

    it('should merge filters when fetching', async () => {
      vi.mocked(ticketApi.list).mockResolvedValue({ tickets: [], total: 0 })

      useTicketStore.setState({ filters: { status: 'todo' } })
      await useTicketStore.getState().fetchTickets({ priority: 'high' })

      expect(ticketApi.list).toHaveBeenCalledWith({ status: 'todo', priority: 'high' })
    })
  })

  describe('fetchTicket', () => {
    it('should fetch single ticket successfully', async () => {
      const mockTicket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Ticket 1', status: 'todo' })
      vi.mocked(ticketApi.get).mockResolvedValue(mockTicket)

      await useTicketStore.getState().fetchTicket('TKT-1')

      expect(ticketApi.get).toHaveBeenCalledWith('TKT-1')
      expect(useTicketStore.getState().currentTicket).toEqual(mockTicket)
    })

    it('should handle fetch ticket error', async () => {
      vi.mocked(ticketApi.get).mockRejectedValue(new Error('Not found'))

      await useTicketStore.getState().fetchTicket('TKT-1')

      expect(useTicketStore.getState().error).toBe('Not found')
    })
  })

  describe('createTicket', () => {
    it('should create ticket successfully', async () => {
      const mockTicket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'New Ticket', status: 'todo' })
      vi.mocked(ticketApi.create).mockResolvedValue(mockTicket)

      const result = await useTicketStore.getState().createTicket({
        repositoryId: 1,
        type: 'task',
        title: 'New Ticket',
      })

      expect(result).toEqual(mockTicket)
      expect(useTicketStore.getState().tickets).toContainEqual(mockTicket)
      expect(useTicketStore.getState().totalCount).toBe(1)
    })

    it('should handle create error', async () => {
      vi.mocked(ticketApi.create).mockRejectedValue(new Error('Creation failed'))

      await expect(useTicketStore.getState().createTicket({
        repositoryId: 1,
        type: 'task',
        title: 'New Ticket',
      })).rejects.toThrow('Creation failed')
    })
  })

  describe('updateTicket', () => {
    it('should update ticket successfully', async () => {
      const existingTicket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Original', status: 'todo' })
      const updatedTicket = { ...existingTicket, title: 'Updated' }

      useTicketStore.setState({ tickets: [existingTicket], currentTicket: existingTicket })
      vi.mocked(ticketApi.update).mockResolvedValue(updatedTicket)

      await useTicketStore.getState().updateTicket('TKT-1', { title: 'Updated' })

      expect(useTicketStore.getState().tickets[0].title).toBe('Updated')
      expect(useTicketStore.getState().currentTicket?.title).toBe('Updated')
    })

    it('should handle update error', async () => {
      vi.mocked(ticketApi.update).mockRejectedValue(new Error('Update failed'))

      await expect(useTicketStore.getState().updateTicket('TKT-1', { title: 'Updated' })).rejects.toThrow()
      expect(useTicketStore.getState().error).toBe('Update failed')
    })
  })

  describe('deleteTicket', () => {
    it('should delete ticket successfully', async () => {
      const ticket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Ticket', status: 'todo' })
      useTicketStore.setState({ tickets: [ticket], totalCount: 1, currentTicket: ticket })
      vi.mocked(ticketApi.delete).mockResolvedValue({ message: 'Deleted' })

      await useTicketStore.getState().deleteTicket('TKT-1')

      expect(useTicketStore.getState().tickets).toHaveLength(0)
      expect(useTicketStore.getState().totalCount).toBe(0)
      expect(useTicketStore.getState().currentTicket).toBeNull()
    })

    it('should handle delete error', async () => {
      vi.mocked(ticketApi.delete).mockRejectedValue(new Error('Delete failed'))

      await expect(useTicketStore.getState().deleteTicket('TKT-1')).rejects.toThrow()
    })
  })

  describe('updateTicketStatus', () => {
    it('should update ticket status successfully', async () => {
      const ticket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Ticket', status: 'todo' })
      useTicketStore.setState({ tickets: [ticket], currentTicket: ticket })
      vi.mocked(ticketApi.updateStatus).mockResolvedValue({ ...ticket, status: 'done' })

      await useTicketStore.getState().updateTicketStatus('TKT-1', 'done')

      expect(useTicketStore.getState().tickets[0].status).toBe('done')
      expect(useTicketStore.getState().currentTicket?.status).toBe('done')
    })

    it('should handle status update error', async () => {
      vi.mocked(ticketApi.updateStatus).mockRejectedValue(new Error('Status update failed'))

      await expect(useTicketStore.getState().updateTicketStatus('TKT-1', 'done')).rejects.toThrow()
    })
  })

  describe('Label operations', () => {
    it('should fetch labels successfully', async () => {
      const mockLabels = [{ id: 1, name: 'bug', color: 'red' }]
      vi.mocked(ticketApi.listLabels).mockResolvedValue({ labels: mockLabels })

      await useTicketStore.getState().fetchLabels()

      expect(useTicketStore.getState().labels).toEqual(mockLabels)
    })

    it('should create label successfully', async () => {
      const newLabel = { id: 1, name: 'feature', color: 'green' }
      vi.mocked(ticketApi.createLabel).mockResolvedValue(newLabel)

      const result = await useTicketStore.getState().createLabel('feature', 'green')

      expect(result).toEqual(newLabel)
      expect(useTicketStore.getState().labels).toContainEqual(newLabel)
    })

    it('should delete label successfully', async () => {
      const label = { id: 1, name: 'bug', color: 'red' }
      useTicketStore.setState({ labels: [label] })
      vi.mocked(ticketApi.deleteLabel).mockResolvedValue({ message: 'Deleted' })

      await useTicketStore.getState().deleteLabel(1)

      expect(useTicketStore.getState().labels).toHaveLength(0)
    })
  })

  describe('setFilters and setCurrentTicket', () => {
    it('should set filters', () => {
      useTicketStore.getState().setFilters({ status: 'in_progress', priority: 'high' })

      expect(useTicketStore.getState().filters).toEqual({ status: 'in_progress', priority: 'high' })
    })

    it('should set current ticket', () => {
      const ticket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Test', status: 'todo' })
      useTicketStore.getState().setCurrentTicket(ticket)

      expect(useTicketStore.getState().currentTicket).toEqual(ticket)
    })

    it('should clear current ticket', () => {
      const ticket = createMockTicket({ id: 1, identifier: 'TKT-1', title: 'Test', status: 'todo' })
      useTicketStore.setState({ currentTicket: ticket })
      useTicketStore.getState().setCurrentTicket(null)

      expect(useTicketStore.getState().currentTicket).toBeNull()
    })
  })

  describe('clearError', () => {
    it('should clear error', () => {
      useTicketStore.setState({ error: 'Some error' })
      useTicketStore.getState().clearError()

      expect(useTicketStore.getState().error).toBeNull()
    })
  })
})

describe('Ticket Store Helper Functions', () => {
  describe('getStatusInfo', () => {
    it('should return correct info for backlog status', () => {
      const info = getStatusInfo('backlog')
      expect(info).toEqual({
        label: 'Backlog',
        color: 'text-gray-600 dark:text-gray-400',
        bgColor: 'bg-gray-100 dark:bg-gray-800',
      })
    })

    it('should return correct info for todo status', () => {
      const info = getStatusInfo('todo')
      expect(info).toEqual({
        label: 'To Do',
        color: 'text-blue-600 dark:text-blue-400',
        bgColor: 'bg-blue-100 dark:bg-blue-900/30',
      })
    })

    it('should return correct info for in_progress status', () => {
      const info = getStatusInfo('in_progress')
      expect(info).toEqual({
        label: 'In Progress',
        color: 'text-yellow-600 dark:text-yellow-400',
        bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
      })
    })

    it('should return correct info for in_review status', () => {
      const info = getStatusInfo('in_review')
      expect(info).toEqual({
        label: 'In Review',
        color: 'text-purple-600 dark:text-purple-400',
        bgColor: 'bg-purple-100 dark:bg-purple-900/30',
      })
    })

    it('should return correct info for done status', () => {
      const info = getStatusInfo('done')
      expect(info).toEqual({
        label: 'Done',
        color: 'text-green-600 dark:text-green-400',
        bgColor: 'bg-green-100 dark:bg-green-900/30',
      })
    })

    it('should return correct info for cancelled status', () => {
      const info = getStatusInfo('cancelled')
      expect(info).toEqual({
        label: 'Cancelled',
        color: 'text-red-600 dark:text-red-400',
        bgColor: 'bg-red-100 dark:bg-red-900/30',
      })
    })
  })

  describe('getPriorityInfo', () => {
    it('should return correct info for none priority', () => {
      const info = getPriorityInfo('none')
      expect(info).toEqual({
        label: 'None',
        color: 'text-gray-400 dark:text-gray-500',
        icon: '—',
      })
    })

    it('should return correct info for low priority', () => {
      const info = getPriorityInfo('low')
      expect(info).toEqual({
        label: 'Low',
        color: 'text-green-500 dark:text-green-400',
        icon: '↓',
      })
    })

    it('should return correct info for medium priority', () => {
      const info = getPriorityInfo('medium')
      expect(info).toEqual({
        label: 'Medium',
        color: 'text-yellow-500 dark:text-yellow-400',
        icon: '→',
      })
    })

    it('should return correct info for high priority', () => {
      const info = getPriorityInfo('high')
      expect(info).toEqual({
        label: 'High',
        color: 'text-orange-500 dark:text-orange-400',
        icon: '↑',
      })
    })

    it('should return correct info for urgent priority', () => {
      const info = getPriorityInfo('urgent')
      expect(info).toEqual({
        label: 'Urgent',
        color: 'text-red-500 dark:text-red-400',
        icon: '⚡',
      })
    })
  })

  describe('getTypeInfo', () => {
    it('should return correct info for task type', () => {
      const info = getTypeInfo('task')
      expect(info).toEqual({
        label: 'Task',
        color: 'text-blue-500 dark:text-blue-400',
        icon: '✓',
      })
    })

    it('should return correct info for bug type', () => {
      const info = getTypeInfo('bug')
      expect(info).toEqual({
        label: 'Bug',
        color: 'text-red-500 dark:text-red-400',
        icon: '🐛',
      })
    })

    it('should return correct info for feature type', () => {
      const info = getTypeInfo('feature')
      expect(info).toEqual({
        label: 'Feature',
        color: 'text-green-500 dark:text-green-400',
        icon: '✨',
      })
    })

    it('should return correct info for epic type', () => {
      const info = getTypeInfo('epic')
      expect(info).toEqual({
        label: 'Epic',
        color: 'text-purple-500 dark:text-purple-400',
        icon: '⚡',
      })
    })

    it('should return correct info for subtask type', () => {
      const info = getTypeInfo('subtask')
      expect(info).toEqual({
        label: 'Subtask',
        color: 'text-gray-500 dark:text-gray-400',
        icon: '◦',
      })
    })

    it('should return correct info for story type', () => {
      const info = getTypeInfo('story')
      expect(info).toEqual({
        label: 'Story',
        color: 'text-teal-500 dark:text-teal-400',
        icon: '📖',
      })
    })
  })
})

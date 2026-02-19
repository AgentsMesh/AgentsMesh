import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import { TicketDetail } from '../TicketDetail'
import { useTicketStore } from '@/stores/ticket'
import { ticketApi } from '@/lib/api'

// Mock next/navigation
const mockRouterBack = vi.fn()
const mockRouterPush = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    back: mockRouterBack,
    push: mockRouterPush,
  }),
}))

// Mock useAuthStore to provide currentOrg
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    currentOrg: { slug: 'test-org' },
  }),
}))

// Mock ticket store
vi.mock('@/stores/ticket', () => ({
  useTicketStore: vi.fn(),
  getStatusInfo: (status: string) => ({
    label: status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase()),
    color: 'text-gray-700',
    bgColor: 'bg-gray-100',
  }),
  getPriorityInfo: (priority: string) => ({
    label: priority.charAt(0).toUpperCase() + priority.slice(1),
    color: 'text-gray-500',
    icon: '→',
  }),
  getTypeInfo: (type: string) => ({
    label: type.charAt(0).toUpperCase() + type.slice(1),
    color: 'text-blue-500',
    icon: '✓',
  }),
}))

// Mock ticket API
vi.mock('@/lib/api', () => ({
  ticketApi: {
    getSubTickets: vi.fn(),
    listRelations: vi.fn(),
    listCommits: vi.fn(),
  },
}))

// Mock TicketPodPanel
vi.mock('../TicketPodPanel', () => ({
  default: ({ ticketIdentifier, ticketTitle }: { ticketIdentifier: string; ticketTitle: string }) => (
    <div data-testid="pod-panel">
      Pod Panel for {ticketIdentifier}: {ticketTitle}
    </div>
  ),
}))

describe('TicketDetail Component', () => {
  const mockTicket = {
    id: 1,
    number: 42,
    identifier: 'PROJ-42',
    type: 'task' as const,
    title: 'Implement new feature',
    description: 'This is the ticket description',
    status: 'in_progress' as const,
    priority: 'high' as const,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-15T12:00:00Z',
    assignees: [
      { id: 1, username: 'john', name: 'John Doe' },
    ],
    labels: [
      { id: 1, name: 'frontend', color: '#3b82f6' },
    ],
    due_date: '2024-02-01T00:00:00Z',
    repository: { id: 1, name: 'my-repo' },
  }

  const mockFetchTicket = vi.fn()
  const mockUpdateTicket = vi.fn()
  const mockUpdateTicketStatus = vi.fn()
  const mockDeleteTicket = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup default mock implementation
    ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
      currentTicket: mockTicket,
      fetchTicket: mockFetchTicket,
      updateTicket: mockUpdateTicket,
      updateTicketStatus: mockUpdateTicketStatus,
      deleteTicket: mockDeleteTicket,
      loading: false,
      error: null,
    })

    // Setup API mocks
    ;(ticketApi.getSubTickets as ReturnType<typeof vi.fn>).mockResolvedValue({ tickets: [] })
    ;(ticketApi.listRelations as ReturnType<typeof vi.fn>).mockResolvedValue({ relations: [] })
    ;(ticketApi.listCommits as ReturnType<typeof vi.fn>).mockResolvedValue({ commits: [] })
  })

  describe('rendering', () => {
    it('should render ticket identifier', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('PROJ-42')).toBeInTheDocument()
      })
    })

    it('should render ticket title', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Implement new feature')).toBeInTheDocument()
      })
    })

    it('should render ticket description', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('This is the ticket description')).toBeInTheDocument()
      })
    })

    it('should render status badge', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        // Status badge should be in the header section with specific styling
        const statusBadge = screen.getByText('In Progress', { selector: 'span.rounded.text-xs' })
        expect(statusBadge).toBeInTheDocument()
      })
    })

    it('should call fetchTicket on mount', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(mockFetchTicket).toHaveBeenCalledWith('PROJ-42')
      })
    })
  })

  describe('loading state', () => {
    it('should render skeleton when loading', () => {
      ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
        currentTicket: null,
        fetchTicket: mockFetchTicket,
        updateTicket: mockUpdateTicket,
        updateTicketStatus: mockUpdateTicketStatus,
        deleteTicket: mockDeleteTicket,
        loading: true,
        error: null,
      })

      render(<TicketDetail identifier="PROJ-42" />)
      expect(screen.getByTestId('ticket-detail-skeleton')).toBeInTheDocument()
    })
  })

  describe('error state', () => {
    it('should render error message', () => {
      ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
        currentTicket: null,
        fetchTicket: mockFetchTicket,
        updateTicket: mockUpdateTicket,
        updateTicketStatus: mockUpdateTicketStatus,
        deleteTicket: mockDeleteTicket,
        loading: false,
        error: 'Failed to load ticket',
      })

      render(<TicketDetail identifier="PROJ-42" />)
      expect(screen.getByText('Failed to load ticket')).toBeInTheDocument()
    })

    it('should render retry button on error', () => {
      ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
        currentTicket: null,
        fetchTicket: mockFetchTicket,
        updateTicket: mockUpdateTicket,
        updateTicketStatus: mockUpdateTicketStatus,
        deleteTicket: mockDeleteTicket,
        loading: false,
        error: 'Failed to load ticket',
      })

      render(<TicketDetail identifier="PROJ-42" />)
      const retryButton = screen.getByText('Retry')
      expect(retryButton).toBeInTheDocument()
    })

    it('should call fetchTicket when retry is clicked', () => {
      ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
        currentTicket: null,
        fetchTicket: mockFetchTicket,
        updateTicket: mockUpdateTicket,
        updateTicketStatus: mockUpdateTicketStatus,
        deleteTicket: mockDeleteTicket,
        loading: false,
        error: 'Failed to load ticket',
      })

      render(<TicketDetail identifier="PROJ-42" />)
      const retryButton = screen.getByText('Retry')
      fireEvent.click(retryButton)

      // Once on mount, once on retry click
      expect(mockFetchTicket).toHaveBeenCalledTimes(2)
    })
  })

  describe('not found state', () => {
    it('should render not found message when ticket is null', () => {
      ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
        currentTicket: null,
        fetchTicket: mockFetchTicket,
        updateTicket: mockUpdateTicket,
        updateTicketStatus: mockUpdateTicketStatus,
        deleteTicket: mockDeleteTicket,
        loading: false,
        error: null,
      })

      render(<TicketDetail identifier="PROJ-42" />)
      expect(screen.getByText('Ticket not found')).toBeInTheDocument()
    })
  })

  describe('labels', () => {
    it('should render labels when provided', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('frontend')).toBeInTheDocument()
      })
    })

    it('should apply label colors', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const label = screen.getByText('frontend')
        expect(label).toHaveStyle({ color: '#3b82f6' })
      })
    })
  })

  describe('assignees', () => {
    it('should render assignees', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument()
      })
    })

    it('should show no assignees message when empty', async () => {
      ;(useTicketStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
        currentTicket: { ...mockTicket, assignees: [] },
        fetchTicket: mockFetchTicket,
        updateTicket: mockUpdateTicket,
        updateTicketStatus: mockUpdateTicketStatus,
        deleteTicket: mockDeleteTicket,
        loading: false,
        error: null,
      })

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('No assignees')).toBeInTheDocument()
      })
    })
  })

  describe('metadata sidebar', () => {
    it('should display ticket type', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Type')).toBeInTheDocument()
      })
    })

    it('should display ticket priority', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Priority')).toBeInTheDocument()
      })
    })

    it('should display due date when provided', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Due Date')).toBeInTheDocument()
      })
    })

    it('should display repository when provided', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Repository')).toBeInTheDocument()
        expect(screen.getByText('my-repo')).toBeInTheDocument()
      })
    })

    it('should display timestamps', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Created')).toBeInTheDocument()
        expect(screen.getByText('Updated')).toBeInTheDocument()
      })
    })
  })

  describe('sub-tickets', () => {
    it('should display sub-tickets when available', async () => {
      ;(ticketApi.getSubTickets as ReturnType<typeof vi.fn>).mockResolvedValue({
        tickets: [
          {
            id: 2,
            identifier: 'PROJ-43',
            title: 'Sub-task 1',
            status: 'todo',
            type: 'task',
          },
        ],
      })

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Sub-tickets (1)')).toBeInTheDocument()
        expect(screen.getByText('PROJ-43')).toBeInTheDocument()
        expect(screen.getByText('Sub-task 1')).toBeInTheDocument()
      })
    })

    it('should not render sub-tickets section when empty', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.queryByText(/Sub-tickets/)).not.toBeInTheDocument()
      })
    })

    it('should navigate to sub-ticket on click', async () => {
      ;(ticketApi.getSubTickets as ReturnType<typeof vi.fn>).mockResolvedValue({
        tickets: [
          {
            id: 2,
            identifier: 'PROJ-43',
            title: 'Sub-task 1',
            status: 'todo',
            type: 'task',
          },
        ],
      })

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const subTicket = screen.getByText('Sub-task 1')
        fireEvent.click(subTicket)
        expect(mockRouterPush).toHaveBeenCalledWith('/test-org/tickets/PROJ-43')
      })
    })
  })

  describe('relations', () => {
    it('should display relations when available', async () => {
      ;(ticketApi.listRelations as ReturnType<typeof vi.fn>).mockResolvedValue({
        relations: [
          {
            id: 1,
            source_ticket_id: 42,
            target_ticket_id: 3,
            relation_type: 'blocks',
            target_ticket: {
              id: 3,
              identifier: 'PROJ-44',
              title: 'Related ticket',
            },
            created_at: '2024-01-10T10:00:00Z',
          },
        ],
      })

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Related (1)')).toBeInTheDocument()
        expect(screen.getByText('PROJ-44')).toBeInTheDocument()
        expect(screen.getByText('Related ticket')).toBeInTheDocument()
      })
    })

    it('should navigate to related ticket on click', async () => {
      ;(ticketApi.listRelations as ReturnType<typeof vi.fn>).mockResolvedValue({
        relations: [
          {
            id: 1,
            source_ticket_id: 42,
            target_ticket_id: 3,
            relation_type: 'blocks',
            target_ticket: {
              id: 3,
              identifier: 'PROJ-44',
              title: 'Related ticket',
            },
            created_at: '2024-01-10T10:00:00Z',
          },
        ],
      })

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const relatedTicket = screen.getByText('Related ticket')
        fireEvent.click(relatedTicket)
        expect(mockRouterPush).toHaveBeenCalledWith('/test-org/tickets/PROJ-44')
      })
    })
  })

  describe('commits', () => {
    it('should display commits when available', async () => {
      ;(ticketApi.listCommits as ReturnType<typeof vi.fn>).mockResolvedValue({
        commits: [
          {
            id: 1,
            ticket_id: 42,
            commit_sha: 'abc123def456',
            commit_message: 'Fix bug in authentication',
            author_name: 'John Doe',
            committed_at: '2024-01-10T10:00:00Z',
            created_at: '2024-01-10T10:00:00Z',
          },
        ],
      })

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Commits (1)')).toBeInTheDocument()
        expect(screen.getByText('abc123d')).toBeInTheDocument() // Short SHA
        expect(screen.getByText('Fix bug in authentication')).toBeInTheDocument()
      })
    })
  })

  describe('pod panel', () => {
    it('should render TicketPodPanel', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByTestId('pod-panel')).toBeInTheDocument()
        expect(screen.getByText(/Pod Panel for PROJ-42/)).toBeInTheDocument()
      })
    })
  })

  describe('editing', () => {
    it('should show edit button', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Edit')).toBeInTheDocument()
      })
    })

    it('should enter edit mode when edit button is clicked', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const editButton = screen.getByText('Edit')
        fireEvent.click(editButton)
      })

      // Should show input for title
      expect(screen.getByDisplayValue('Implement new feature')).toBeInTheDocument()
      // Should show Save and Cancel buttons
      expect(screen.getByText('Save')).toBeInTheDocument()
      expect(screen.getByText('Cancel')).toBeInTheDocument()
    })

    it('should call updateTicket when save is clicked', async () => {
      mockUpdateTicket.mockResolvedValue({})

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const editButton = screen.getByText('Edit')
        fireEvent.click(editButton)
      })

      const titleInput = screen.getByDisplayValue('Implement new feature')
      fireEvent.change(titleInput, { target: { value: 'Updated title' } })

      const saveButton = screen.getByText('Save')
      fireEvent.click(saveButton)

      await waitFor(() => {
        expect(mockUpdateTicket).toHaveBeenCalledWith('PROJ-42', {
          title: 'Updated title',
          description: 'This is the ticket description',
          content: '', // content field is now included
        })
      })
    })

    it('should exit edit mode when cancel is clicked', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const editButton = screen.getByText('Edit')
        fireEvent.click(editButton)
      })

      const cancelButton = screen.getByText('Cancel')
      fireEvent.click(cancelButton)

      // Should show Edit button again
      await waitFor(() => {
        expect(screen.getByText('Edit')).toBeInTheDocument()
      })
    })
  })

  describe('status change', () => {
    it('should show status dropdown', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const statusDropdown = screen.getByRole('combobox')
        expect(statusDropdown).toBeInTheDocument()
        expect(statusDropdown).toHaveValue('in_progress')
      })
    })

    it('should call updateTicketStatus when status is changed', async () => {
      mockUpdateTicketStatus.mockResolvedValue({})

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const statusDropdown = screen.getByRole('combobox')
        fireEvent.change(statusDropdown, { target: { value: 'done' } })
      })

      await waitFor(() => {
        expect(mockUpdateTicketStatus).toHaveBeenCalledWith('PROJ-42', 'done')
      })
    })
  })

  describe('delete action', () => {
    it('should show delete button', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Delete')).toBeInTheDocument()
      })
    })

    it('should show confirmation modal when delete is clicked', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const deleteButton = screen.getByText('Delete')
        fireEvent.click(deleteButton)
      })

      expect(screen.getByText('Delete Ticket')).toBeInTheDocument()
      expect(screen.getByText(/Are you sure you want to delete ticket/)).toBeInTheDocument()
    })

    it('should call deleteTicket and navigate to tickets list when confirmed', async () => {
      mockDeleteTicket.mockResolvedValue({})

      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const deleteButton = screen.getByText('Delete')
        fireEvent.click(deleteButton)
      })

      // Click the confirm delete button in the modal
      const confirmButtons = screen.getAllByText('Delete')
      const confirmDeleteButton = confirmButtons[confirmButtons.length - 1]
      fireEvent.click(confirmDeleteButton)

      await waitFor(() => {
        expect(mockDeleteTicket).toHaveBeenCalledWith('PROJ-42')
        expect(mockRouterPush).toHaveBeenCalledWith('/test-org/tickets')
      })
    })

    it('should close modal when cancel is clicked', async () => {
      render(<TicketDetail identifier="PROJ-42" />)
      await waitFor(() => {
        const deleteButton = screen.getByText('Delete')
        fireEvent.click(deleteButton)
      })

      const cancelButton = screen.getAllByText('Cancel')[0]
      fireEvent.click(cancelButton)

      await waitFor(() => {
        expect(screen.queryByText('Delete Ticket')).not.toBeInTheDocument()
      })
    })
  })
})

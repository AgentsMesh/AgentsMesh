import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import { TicketDetail } from '../TicketDetail'
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

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    currentOrg: { slug: 'test-org' },
    user: { id: 1, username: 'john' },
  }),
}))

const mockTicketStoreState: Record<string, unknown> = {}
vi.mock('@/stores/ticket', () => ({
  useTicketStore: vi.fn((selector?: (state: Record<string, unknown>) => unknown) =>
    selector ? selector(mockTicketStoreState) : mockTicketStoreState
  ),
  getStatusInfo: (status: string) => ({
    label: status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase()),
    color: 'text-gray-700',
    bgColor: 'bg-gray-100',
  }),
  getPriorityInfo: (priority: string) => ({
    label: priority.charAt(0).toUpperCase() + priority.slice(1),
    color: 'text-gray-500',
    icon: '->',
  }),
}))

vi.mock('@/lib/api', () => ({
  ticketApi: {
    getSubTickets: vi.fn(),
    listRelations: vi.fn(),
    listCommits: vi.fn(),
    listComments: vi.fn(),
    getPods: vi.fn(),
  },
  organizationApi: {
    listMembers: vi.fn().mockResolvedValue({ members: [] }),
  },
}))

vi.mock('@/components/common/RepositorySelect', () => ({
  RepositorySelect: ({ value, onChange, placeholder }: { value: number | null; onChange: (v: number | null) => void; placeholder?: string }) => (
    <select data-testid="repository-select" value={value ?? ''} onChange={(e) => onChange(e.target.value ? Number(e.target.value) : null)}>
      <option value="">{placeholder || 'Select...'}</option>
      <option value="1">my-repo</option>
    </select>
  ),
}))

vi.mock('@/components/ui/block-editor', () => ({
  BlockViewer: ({ content }: { content: string }) => <div data-testid="block-viewer">{content}</div>,
  default: ({ initialContent, onChange }: { initialContent: string; onChange?: (v: string) => void; editable?: boolean }) => (
    <div data-testid="block-editor" onClick={() => onChange?.('updated content')}>{initialContent}</div>
  ),
}))

vi.mock('@/stores/workspace', () => ({
  useWorkspaceStore: () => ({
    addPane: vi.fn(),
  }),
}))

vi.mock('@/components/ide/CreatePodModal', () => ({
  CreatePodModal: () => null,
}))

vi.mock('@/lib/pod-utils', () => ({
  getPodDisplayName: (pod: { pod_key: string }) => pod.pod_key,
}))
vi.mock('@/components/shared/AgentStatusBadge', () => ({
  AgentStatusBadge: () => null,
}))

describe('TicketDetail - Editing, Status & Delete', () => {
  const mockTicket = {
    id: 1,
    number: 42,
    slug: 'PROJ-42',
    title: 'Implement new feature',
    content: 'This is the ticket description',
    status: 'in_progress' as const,
    priority: 'high' as const,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-15T12:00:00Z',
    assignees: [
      { ticket_id: 1, user_id: 1, user: { id: 1, username: 'john', name: 'John Doe' } },
    ],
    labels: [
      { id: 1, name: 'frontend', color: '#3b82f6' },
    ],
    due_date: '2024-02-01T00:00:00Z',
    repository_id: 1,
    repository: { id: 1, name: 'my-repo' },
  }

  const mockFetchTicket = vi.fn().mockResolvedValue(undefined)
  const mockDeleteTicket = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()

    Object.assign(mockTicketStoreState, {
      currentTicket: mockTicket,
      fetchTicket: mockFetchTicket,
      setCurrentTicket: vi.fn(),
      updateTicket: vi.fn(),
      updateTicketStatus: vi.fn(),
      deleteTicket: mockDeleteTicket,
      loading: false,
      error: null,
    })

    ;(ticketApi.getSubTickets as ReturnType<typeof vi.fn>).mockResolvedValue({ sub_tickets: [] })
    ;(ticketApi.listRelations as ReturnType<typeof vi.fn>).mockResolvedValue({ relations: [] })
    ;(ticketApi.listCommits as ReturnType<typeof vi.fn>).mockResolvedValue({ commits: [] })
    ;(ticketApi.listComments as ReturnType<typeof vi.fn>).mockResolvedValue({ comments: [], total: 0 })
    ;(ticketApi.getPods as ReturnType<typeof vi.fn>).mockResolvedValue({ pods: [] })
  })

  describe('pod panel', () => {
    it('should show execute button in sidebar', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Execute')).toBeInTheDocument()
      })
    })

    it('should show no pods message when empty', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('No pods for this ticket yet')).toBeInTheDocument()
      })
    })
  })

  describe('inline editing (Linear-style)', () => {
    it('should render inline editable title', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Implement new feature')).toBeInTheDocument()
      })
    })

    it('should render block editor for content (always editable)', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByTestId('block-editor')).toBeInTheDocument()
      })
    })

    it('should not show a separate Edit button', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
      })
    })
  })

  describe('status change', () => {
    it('should render StatusSelect in sidebar', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Status')).toBeInTheDocument()
      })
    })
  })

  describe('delete action', () => {
    it('should show delete button', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Delete')).toBeInTheDocument()
      })
    })

    it('should show confirmation modal when delete is clicked', async () => {
      render(<TicketDetail slug="PROJ-42" />)

      await waitFor(() => {
        const deleteButton = screen.getByText('Delete')
        fireEvent.click(deleteButton)
      })

      expect(screen.getByText('Delete Ticket')).toBeInTheDocument()
      expect(screen.getByText(/Are you sure you want to delete ticket/)).toBeInTheDocument()
    })

    it('should call deleteTicket and navigate when confirmed', async () => {
      mockDeleteTicket.mockResolvedValue({})

      render(<TicketDetail slug="PROJ-42" />)

      await waitFor(() => {
        const deleteButton = screen.getByText('Delete')
        fireEvent.click(deleteButton)
      })

      const confirmButtons = screen.getAllByText('Delete')
      const confirmDeleteButton = confirmButtons[confirmButtons.length - 1]
      fireEvent.click(confirmDeleteButton)

      await waitFor(() => {
        expect(mockDeleteTicket).toHaveBeenCalledWith('PROJ-42')
        expect(mockRouterPush).toHaveBeenCalledWith('/test-org/tickets')
      })
    })

    it('should close modal when cancel is clicked', async () => {
      render(<TicketDetail slug="PROJ-42" />)

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

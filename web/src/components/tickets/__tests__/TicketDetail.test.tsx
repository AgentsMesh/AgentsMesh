import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import { TicketDetail } from '../TicketDetail'
import { getTicketRelationsService, getApiClient, getOrgApiService } from '@/lib/wasm-core'

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
  useCurrentUser: () => ({ id: 1, email: 'u@e.com', username: 'john' }),
  useCurrentOrg: () => ({ id: 1, name: 'TestOrg', slug: 'test-org' }),
  useAuthOrganizations: () => [],
  readCurrentUser: () => ({ id: 1, email: 'u@e.com', username: 'john' }),
  readCurrentOrg: () => ({ id: 1, name: 'TestOrg', slug: 'test-org' }),
  readOrganizations: () => [],
}))

const mockTicketStoreState: Record<string, unknown> = {}
vi.mock('@/stores/ticket', () => ({
  useTicketStore: vi.fn((selector?: (state: Record<string, unknown>) => unknown) =>
    selector ? selector(mockTicketStoreState) : mockTicketStoreState
  ),
  useCurrentTicket: vi.fn(() => mockTicketStoreState.currentTicket ?? null),
  useTickets: vi.fn(() => []),
  useBoardColumns: vi.fn(() => []),
  useLabels: vi.fn(() => []),
  getStatusInfo: (status: string) => ({
    label: status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase()),
    color: 'text-gray-700',
    bgColor: 'bg-gray-100',
  }),
  getPriorityInfo: (priority: string) => ({
    label: priority.charAt(0).toUpperCase() + priority.slice(1),
    color: 'text-gray-500',
    icon: '\u2192',
  }),
}))

vi.mock('@/lib/api', () => ({}))

vi.mock('@/lib/wasm-getters', async () => {
  const wasmCore = await vi.importMock<typeof import('@/lib/wasm-core')>('@/lib/wasm-core')
  return { ...wasmCore }
})

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

// Mock pod display name helpers and AgentStatusBadge
vi.mock('@/lib/pod-display-name', () => ({
  getPodDisplayName: (pod: { pod_key: string }) => pod.pod_key,
}))
vi.mock('@/components/shared/AgentStatusBadge', () => ({
  AgentStatusBadge: () => null,
}))

describe('TicketDetail Component', () => {
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
  const mockSetCurrentTicket = vi.fn()
  const mockUpdateTicket = vi.fn()
  const mockUpdateTicketStatus = vi.fn()
  const mockDeleteTicket = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()

    Object.assign(mockTicketStoreState, {
      currentTicket: mockTicket,
      fetchTicket: mockFetchTicket,
      setCurrentTicket: mockSetCurrentTicket,
      updateTicket: mockUpdateTicket,
      updateTicketStatus: mockUpdateTicketStatus,
      deleteTicket: mockDeleteTicket,
      loading: false,
      error: null,
    })

    const client = getApiClient()
    vi.mocked(client.get).mockResolvedValue(JSON.stringify({ sub_tickets: [], pods: [] }))
    vi.mocked(getTicketRelationsService().list_relations).mockResolvedValue(JSON.stringify({ relations: [] }))
    vi.mocked(getTicketRelationsService().list_commits).mockResolvedValue(JSON.stringify({ commits: [] }))
    vi.mocked(getTicketRelationsService().list_comments).mockResolvedValue(JSON.stringify({ comments: [], total: 0 }))

    vi.mocked(getOrgApiService().list_members).mockResolvedValue(JSON.stringify({ members: [] }))
  })

  describe('rendering', () => {
    it('should render ticket title', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Implement new feature')).toBeInTheDocument()
      })
    })

    it('should render ticket description in block editor', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        const editor = screen.getByTestId('block-editor')
        expect(editor).toBeInTheDocument()
        expect(editor).toHaveTextContent('This is the ticket description')
      })
    })

    it('should render status badge', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        const badges = screen.getAllByText('In Progress')
        expect(badges.length).toBeGreaterThanOrEqual(1)
      })
    })

    it('should call fetchTicket on mount', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(mockFetchTicket).toHaveBeenCalledWith('PROJ-42')
      })
    })
  })

  describe('loading state', () => {
    it('should render skeleton when loading', () => {
      Object.assign(mockTicketStoreState, {
        currentTicket: null,
        loading: true,
        error: null,
      })

      render(<TicketDetail slug="PROJ-42" />)
      expect(screen.getByTestId('ticket-detail-skeleton')).toBeInTheDocument()
    })
  })

  describe('error state', () => {
    it('should render error message', async () => {
      Object.assign(mockTicketStoreState, {
        currentTicket: null,
        loading: false,
        error: 'Failed to load ticket',
      })

      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Failed to load ticket')).toBeInTheDocument()
      })
    })

    it('should render retry button on error', async () => {
      Object.assign(mockTicketStoreState, {
        currentTicket: null,
        loading: false,
        error: 'Failed to load ticket',
      })

      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        const retryButton = screen.getByText('Retry')
        expect(retryButton).toBeInTheDocument()
      })
    })

    it('should call fetchTicket when retry is clicked', async () => {
      Object.assign(mockTicketStoreState, {
        currentTicket: null,
        loading: false,
        error: 'Failed to load ticket',
      })

      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        const retryButton = screen.getByText('Retry')
        fireEvent.click(retryButton)
      })

      expect(mockFetchTicket).toHaveBeenCalledTimes(2)
    })
  })

  describe('not found state', () => {
    it('should render not found message when ticket is null', async () => {
      Object.assign(mockTicketStoreState, {
        currentTicket: null,
        loading: false,
        error: null,
      })

      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('Ticket not found')).toBeInTheDocument()
      })
    })
  })

  describe('labels', () => {
    it('should render labels when provided', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        expect(screen.getByText('frontend')).toBeInTheDocument()
      })
    })

    it('should apply label colors', async () => {
      render(<TicketDetail slug="PROJ-42" />)
      await waitFor(() => {
        const label = screen.getByText('frontend')
        expect(label).toHaveStyle({ color: '#3b82f6' })
      })
    })
  })

  // NOTE: assignees / repository / due date / timestamps now live in
  // TicketDetailSidebar (right rail) and should be covered by its own tests.

})

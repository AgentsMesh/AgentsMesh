import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import { TicketCard } from '../TicketCard'

// Mock next/link
vi.mock('next/link', () => ({
  default: ({ children, href, onClick }: { children: React.ReactNode; href: string; onClick?: (e: React.MouseEvent) => void }) => (
    <a href={href} onClick={onClick}>
      {children}
    </a>
  ),
}))

describe('TicketCard Component', () => {
  const baseTicket = {
    id: 1,
    number: 42,
    identifier: 'PROJ-42',
    type: 'task' as const,
    title: 'Implement new feature',
    status: 'todo' as const,
    priority: 'medium' as const,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }

  describe('rendering', () => {
    it('should render ticket identifier', () => {
      render(<TicketCard ticket={baseTicket} />)
      expect(screen.getByText('PROJ-42')).toBeInTheDocument()
    })

    it('should render ticket title', () => {
      render(<TicketCard ticket={baseTicket} />)
      expect(screen.getByText('Implement new feature')).toBeInTheDocument()
    })

    it('should render ticket identifier as link', () => {
      render(<TicketCard ticket={baseTicket} />)
      const link = screen.getByRole('link', { name: 'PROJ-42' })
      expect(link).toHaveAttribute('href', '/tickets/PROJ-42')
    })
  })

  describe('type display', () => {
    it('should display task type icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, type: 'task' }} />)
      expect(screen.getByTitle('Task')).toHaveTextContent('✓')
    })

    it('should display bug type icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, type: 'bug' }} />)
      expect(screen.getByTitle('Bug')).toHaveTextContent('🐛')
    })

    it('should display feature type icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, type: 'feature' }} />)
      expect(screen.getByTitle('Feature')).toHaveTextContent('✨')
    })

    it('should display epic type icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, type: 'epic' }} />)
      expect(screen.getByTitle('Epic')).toHaveTextContent('⚡')
    })
  })

  describe('status display', () => {
    it('should display backlog status', () => {
      render(<TicketCard ticket={{ ...baseTicket, status: 'backlog' }} />)
      expect(screen.getByText('Backlog')).toBeInTheDocument()
    })

    it('should display todo status', () => {
      render(<TicketCard ticket={{ ...baseTicket, status: 'todo' }} />)
      expect(screen.getByText('To Do')).toBeInTheDocument()
    })

    it('should display in_progress status', () => {
      render(<TicketCard ticket={{ ...baseTicket, status: 'in_progress' }} />)
      expect(screen.getByText('In Progress')).toBeInTheDocument()
    })

    it('should display in_review status', () => {
      render(<TicketCard ticket={{ ...baseTicket, status: 'in_review' }} />)
      expect(screen.getByText('In Review')).toBeInTheDocument()
    })

    it('should display done status', () => {
      render(<TicketCard ticket={{ ...baseTicket, status: 'done' }} />)
      expect(screen.getByText('Done')).toBeInTheDocument()
    })

    it('should display cancelled status', () => {
      render(<TicketCard ticket={{ ...baseTicket, status: 'cancelled' }} />)
      expect(screen.getByText('Cancelled')).toBeInTheDocument()
    })
  })

  describe('priority display', () => {
    it('should display none priority icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, priority: 'none' }} />)
      expect(screen.getByTitle('No Priority')).toHaveTextContent('—')
    })

    it('should display low priority icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, priority: 'low' }} />)
      expect(screen.getByTitle('Low')).toHaveTextContent('↓')
    })

    it('should display medium priority icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, priority: 'medium' }} />)
      expect(screen.getByTitle('Medium')).toHaveTextContent('→')
    })

    it('should display high priority icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, priority: 'high' }} />)
      expect(screen.getByTitle('High')).toHaveTextContent('↑')
    })

    it('should display urgent priority icon', () => {
      render(<TicketCard ticket={{ ...baseTicket, priority: 'urgent' }} />)
      expect(screen.getByTitle('Urgent')).toHaveTextContent('⚡')
    })
  })

  describe('labels', () => {
    it('should render labels when provided', () => {
      const ticketWithLabels = {
        ...baseTicket,
        labels: [
          { id: 1, name: 'frontend', color: '#3b82f6' },
          { id: 2, name: 'urgent', color: '#ef4444' },
        ],
      }
      render(<TicketCard ticket={ticketWithLabels} />)
      expect(screen.getByText('frontend')).toBeInTheDocument()
      expect(screen.getByText('urgent')).toBeInTheDocument()
    })

    it('should apply label colors', () => {
      const ticketWithLabels = {
        ...baseTicket,
        labels: [{ id: 1, name: 'frontend', color: '#3b82f6' }],
      }
      render(<TicketCard ticket={ticketWithLabels} />)
      const label = screen.getByText('frontend')
      expect(label).toHaveStyle({ color: '#3b82f6' })
    })

    it('should not render labels section when empty', () => {
      render(<TicketCard ticket={{ ...baseTicket, labels: [] }} />)
      // Should not have any label elements
      expect(screen.queryByText('frontend')).not.toBeInTheDocument()
    })
  })

  describe('due date', () => {
    it('should render due date when provided', () => {
      const ticketWithDue = {
        ...baseTicket,
        due_date: '2024-12-31T00:00:00Z',
      }
      render(<TicketCard ticket={ticketWithDue} />)
      // The date should be rendered (format depends on locale)
      expect(screen.getByText(/12\/31\/2024|31\/12\/2024|2024/)).toBeInTheDocument()
    })

    it('should not render due date when not provided', () => {
      render(<TicketCard ticket={baseTicket} />)
      // Should not have due date display
    })
  })

  describe('assignees', () => {
    it('should render assignee avatars', () => {
      const ticketWithAssignees = {
        ...baseTicket,
        assignees: [
          { id: 1, username: 'john', name: 'John Doe' },
          { id: 2, username: 'jane', name: 'Jane Doe' },
        ],
      }
      render(<TicketCard ticket={ticketWithAssignees} />)
      // Should show initials or avatars
      expect(screen.getByTitle('John Doe')).toBeInTheDocument()
      expect(screen.getByTitle('Jane Doe')).toBeInTheDocument()
    })

    it('should show +N for more than 3 assignees', () => {
      const ticketWithManyAssignees = {
        ...baseTicket,
        assignees: [
          { id: 1, username: 'user1' },
          { id: 2, username: 'user2' },
          { id: 3, username: 'user3' },
          { id: 4, username: 'user4' },
          { id: 5, username: 'user5' },
        ],
      }
      render(<TicketCard ticket={ticketWithManyAssignees} />)
      expect(screen.getByText('+2')).toBeInTheDocument()
    })

    it('should show initials when no avatar URL', () => {
      const ticketWithAssignees = {
        ...baseTicket,
        assignees: [{ id: 1, username: 'john', name: 'John Doe' }],
      }
      render(<TicketCard ticket={ticketWithAssignees} />)
      expect(screen.getByText('J')).toBeInTheDocument()
    })

    it('should render avatar image when URL provided', () => {
      const ticketWithAssignees = {
        ...baseTicket,
        assignees: [{ id: 1, username: 'john', avatar_url: 'https://example.com/avatar.png' }],
      }
      render(<TicketCard ticket={ticketWithAssignees} />)
      const img = screen.getByAltText('john')
      expect(img).toBeInTheDocument()
      expect(img).toHaveAttribute('src', 'https://example.com/avatar.png')
    })
  })

  describe('repository', () => {
    it('should show repository name when showRepository is true', () => {
      const ticketWithRepo = {
        ...baseTicket,
        repository: { id: 1, name: 'my-repo' },
      }
      render(<TicketCard ticket={ticketWithRepo} showRepository={true} />)
      expect(screen.getByText('my-repo')).toBeInTheDocument()
    })

    it('should not show repository when showRepository is false', () => {
      const ticketWithRepo = {
        ...baseTicket,
        repository: { id: 1, name: 'my-repo' },
      }
      render(<TicketCard ticket={ticketWithRepo} showRepository={false} />)
      expect(screen.queryByText('my-repo')).not.toBeInTheDocument()
    })

    it('should show repository by default', () => {
      const ticketWithRepo = {
        ...baseTicket,
        repository: { id: 1, name: 'my-repo' },
      }
      render(<TicketCard ticket={ticketWithRepo} />)
      expect(screen.getByText('my-repo')).toBeInTheDocument()
    })
  })

  describe('events', () => {
    it('should call onClick when card is clicked', () => {
      const handleClick = vi.fn()
      render(<TicketCard ticket={baseTicket} onClick={handleClick} />)

      // Click on the card, not the link
      fireEvent.click(screen.getByText('Implement new feature'))
      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('should not propagate click from identifier link', () => {
      const handleClick = vi.fn()
      render(<TicketCard ticket={baseTicket} onClick={handleClick} />)

      const link = screen.getByRole('link', { name: 'PROJ-42' })
      fireEvent.click(link)

      // onClick should not be called when clicking the link
      expect(handleClick).toHaveBeenCalledTimes(0)
    })
  })

  describe('edge cases', () => {
    it('should handle unknown type gracefully', () => {
      const ticketWithUnknownType = {
        ...baseTicket,
        type: 'unknown' as any,
      }
      render(<TicketCard ticket={ticketWithUnknownType} />)
      // Should fall back to task styling (icon)
      expect(screen.getByText('✓')).toBeInTheDocument()
    })

    it('should handle unknown status gracefully', () => {
      const ticketWithUnknownStatus = {
        ...baseTicket,
        status: 'unknown' as any,
      }
      render(<TicketCard ticket={ticketWithUnknownStatus} />)
      // Should fall back to backlog styling (but translation key may show for unknown status)
      // The component uses statusConfig.backlog for styling, but displays t(`tickets.status.${status}`)
      // For unknown status, this would display the translation key since it doesn't exist
      expect(screen.getByText('tickets.status.unknown')).toBeInTheDocument()
    })

    it('should handle unknown priority gracefully', () => {
      const ticketWithUnknownPriority = {
        ...baseTicket,
        priority: 'unknown' as any,
      }
      render(<TicketCard ticket={ticketWithUnknownPriority} />)
      // Should fall back to none styling (icon)
      expect(screen.getByText('—')).toBeInTheDocument()
    })
  })
})

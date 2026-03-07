package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// TicketListFilter represents filters for listing tickets.
type TicketListFilter struct {
	OrganizationID int64
	RepositoryID   *int64
	Status         string
	Priority       string
	AssigneeID     *int64
	ReporterID     *int64
	LabelIDs       []int64
	ParentTicketID *int64
	Query          string
	UserRole       string // Kept for future use
	Limit          int
	Offset         int
}

// CreateTicketParams holds the parameters for creating a ticket atomically.
type CreateTicketParams struct {
	Ticket      *Ticket
	Prefix      string   // e.g. "AM", used to generate slug like "AM-123"
	AssigneeIDs []int64
	LabelIDs    []int64
	LabelNames  []string // label names resolved inside transaction
}

// TicketRepository defines the data-access contract for the ticket aggregate.
type TicketRepository interface {
	// --- Ticket CRUD ---
	GetByID(ctx context.Context, ticketID int64) (*Ticket, error)
	GetByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*Ticket, error)
	List(ctx context.Context, filter *TicketListFilter) ([]*Ticket, int64, error)
	UpdateFields(ctx context.Context, ticketID int64, updates map[string]interface{}) error
	CreateTicketAtomic(ctx context.Context, params *CreateTicketParams) error
	DeleteTicketAtomic(ctx context.Context, ticketID int64) error
	GetRepoTicketPrefix(ctx context.Context, repoID int64) (string, error)

	// --- Assignees ---
	ReplaceAssignees(ctx context.Context, ticketID int64, userIDs []int64) error
	AddAssignee(ctx context.Context, ticketID, userID int64) error
	RemoveAssignee(ctx context.Context, ticketID, userID int64) error
	GetAssigneeUsers(ctx context.Context, ticketID int64) ([]*user.User, error)

	// --- Comments ---
	GetCommentByIDAndTicket(ctx context.Context, commentID, ticketID int64) (*Comment, error)
	CreateComment(ctx context.Context, comment *Comment) error
	GetCommentWithUser(ctx context.Context, commentID int64) (*Comment, error)
	ListComments(ctx context.Context, ticketID int64, limit, offset int) ([]*Comment, int64, error)
	GetComment(ctx context.Context, commentID int64) (*Comment, error)
	UpdateComment(ctx context.Context, comment *Comment) error
	DeleteCommentAtomic(ctx context.Context, commentID int64) error
	DeleteCommentsByTicket(ctx context.Context, ticketID int64) error

	// --- Labels ---
	GetLabelByOrgNameRepo(ctx context.Context, orgID int64, name string, repoID *int64) (*Label, error)
	CreateLabel(ctx context.Context, label *Label) error
	GetLabel(ctx context.Context, labelID int64) (*Label, error)
	ListLabels(ctx context.Context, orgID int64, repoID *int64) ([]*Label, error)
	UpdateLabelFields(ctx context.Context, orgID, labelID int64, updates map[string]interface{}) error
	DeleteLabelAtomic(ctx context.Context, orgID, labelID int64) error
	GetTicketLabels(ctx context.Context, ticketID int64) ([]*Label, error)
	AddTicketLabel(ctx context.Context, ticketID, labelID int64) error
	RemoveTicketLabel(ctx context.Context, ticketID, labelID int64) error

	// --- Relations ---
	GetRelation(ctx context.Context, relationID int64) (*Relation, error)
	CreateRelationPair(ctx context.Context, relation, reverse *Relation) error
	DeleteRelationPair(ctx context.Context, relation *Relation, reverseType string) error
	ListRelations(ctx context.Context, ticketID int64) ([]*Relation, error)

	// --- Commits ---
	CreateCommit(ctx context.Context, commit *Commit) error
	DeleteCommit(ctx context.Context, commitID int64) error
	ListCommitsByTicket(ctx context.Context, ticketID int64) ([]*Commit, error)
	GetCommitBySHA(ctx context.Context, repoID int64, sha string) (*Commit, error)

	// --- Merge Requests ---
	CreateMR(ctx context.Context, mr *MergeRequest) error
	UpdateMRState(ctx context.Context, mrID int64, state string) error
	GetMRByURL(ctx context.Context, mrURL string) (*MergeRequest, error)
	ListMRsByTicket(ctx context.Context, ticketID int64) ([]*MergeRequest, error)

	// --- Board / Statistics ---
	GetActiveTickets(ctx context.Context, orgID int64, repoID *int64, limit int) ([]*Ticket, error)
	GetChildTickets(ctx context.Context, parentTicketID int64) ([]*Ticket, error)
	GetSubTicketCounts(ctx context.Context, parentIDs []int64) (map[int64]map[string]int64, error)
	GetTicketStats(ctx context.Context, orgID int64, repoID *int64) (map[string]int64, error)
}

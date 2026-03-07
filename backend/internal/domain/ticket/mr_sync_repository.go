package ticket

import "context"

// PodForMRSync is a lightweight projection of a pod record used by MR synchronisation.
// It avoids an import cycle between domain/ticket and domain/agentpod.
type PodForMRSync struct {
	ID             int64
	OrganizationID int64
	BranchName     *string
	TicketID       *int64
}

// MRSyncRepository defines the data-access contract for MR synchronisation.
type MRSyncRepository interface {
	// --- MR CRUD ---
	GetMRByURL(ctx context.Context, mrURL string) (*MergeRequest, error)
	GetMRByURLWithTicket(ctx context.Context, mrURL string) (*MergeRequest, error)
	SaveMR(ctx context.Context, mr *MergeRequest) error
	CreateMR(ctx context.Context, mr *MergeRequest) error
	ListMRsByTicket(ctx context.Context, ticketID int64) ([]*MergeRequest, error)
	ListMRsByPod(ctx context.Context, podID int64) ([]*MergeRequest, error)

	// --- Ticket look-ups ---
	FindTicketByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*Ticket, error)
	GetTicketByID(ctx context.Context, ticketID int64) (*Ticket, error)

	// --- Cross-domain helpers ---
	GetRepoExternalID(ctx context.Context, repoID int64) (string, error)
	FindPodsWithoutMR(ctx context.Context) ([]*PodForMRSync, error)
	ListOpenMRsWithTicket(ctx context.Context) ([]*MergeRequest, error)
}

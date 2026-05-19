package ticket

import "context"

type PodForMRSync struct {
	ID             int64
	OrganizationID int64
	BranchName     *string
	TicketID       *int64
}

type MRSyncRepository interface {
	GetMRByURL(ctx context.Context, mrURL string) (*MergeRequest, error)
	GetMRByURLWithTicket(ctx context.Context, mrURL string) (*MergeRequest, error)
	SaveMR(ctx context.Context, mr *MergeRequest) error
	CreateMR(ctx context.Context, mr *MergeRequest) error
	ListMRsByTicket(ctx context.Context, ticketID int64) ([]*MergeRequest, error)
	ListMRsByPod(ctx context.Context, podID int64) ([]*MergeRequest, error)

	FindTicketByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*Ticket, error)
	GetTicketByID(ctx context.Context, ticketID int64) (*Ticket, error)

	GetRepoExternalID(ctx context.Context, repoID int64) (string, error)
	FindPodsWithoutMR(ctx context.Context) ([]*PodForMRSync, error)
	ListOpenMRsWithTicket(ctx context.Context) ([]*MergeRequest, error)
}

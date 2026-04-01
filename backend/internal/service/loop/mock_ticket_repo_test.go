package loop

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// mockTicketRepoForCrossUnit captures CreateComment calls for assertion
// while stubbing all other TicketRepository methods.
type mockTicketRepoForCrossUnit struct {
	mu       sync.Mutex
	comments []capturedComment
}

type capturedComment struct {
	TicketID int64
	UserID   int64
	Content  string
}

func (m *mockTicketRepoForCrossUnit) CreateComment(_ context.Context, c *ticket.Comment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.ID = int64(len(m.comments) + 1)
	m.comments = append(m.comments, capturedComment{
		TicketID: c.TicketID,
		UserID:   c.UserID,
		Content:  c.Content,
	})
	return nil
}

func (m *mockTicketRepoForCrossUnit) getComments() []capturedComment {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]capturedComment, len(m.comments))
	copy(cp, m.comments)
	return cp
}

// --- Stubs to satisfy ticket.TicketRepository (only CreateComment is used) ---

func (m *mockTicketRepoForCrossUnit) GetCommentWithUser(_ context.Context, commentID int64) (*ticket.Comment, error) {
	return &ticket.Comment{ID: commentID}, nil
}
func (m *mockTicketRepoForCrossUnit) GetByID(context.Context, int64) (*ticket.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetByOrgAndSlug(context.Context, int64, string) (*ticket.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) List(context.Context, *ticket.TicketListFilter) ([]*ticket.Ticket, int64, error) {
	return nil, 0, nil
}
func (m *mockTicketRepoForCrossUnit) UpdateFields(context.Context, int64, map[string]interface{}) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) CreateTicketAtomic(context.Context, *ticket.CreateTicketParams) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) DeleteTicketAtomic(context.Context, int64) error { return nil }
func (m *mockTicketRepoForCrossUnit) GetRepoTicketPrefix(context.Context, int64) (string, error) {
	return "", nil
}
func (m *mockTicketRepoForCrossUnit) ReplaceAssignees(context.Context, int64, []int64) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) AddAssignee(context.Context, int64, int64) error { return nil }
func (m *mockTicketRepoForCrossUnit) RemoveAssignee(context.Context, int64, int64) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) GetAssigneeUsers(context.Context, int64) ([]*user.User, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetCommentByIDAndTicket(context.Context, int64, int64) (*ticket.Comment, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) ListComments(context.Context, int64, int, int) ([]*ticket.Comment, int64, error) {
	return nil, 0, nil
}
func (m *mockTicketRepoForCrossUnit) GetComment(context.Context, int64) (*ticket.Comment, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) UpdateComment(context.Context, *ticket.Comment) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) DeleteCommentAtomic(context.Context, int64) error { return nil }
func (m *mockTicketRepoForCrossUnit) DeleteCommentsByTicket(context.Context, int64) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) GetLabelByOrgNameRepo(context.Context, int64, string, *int64) (*ticket.Label, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) CreateLabel(context.Context, *ticket.Label) error { return nil }
func (m *mockTicketRepoForCrossUnit) GetLabel(context.Context, int64) (*ticket.Label, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) ListLabels(context.Context, int64, *int64) ([]*ticket.Label, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) UpdateLabelFields(context.Context, int64, int64, map[string]interface{}) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) DeleteLabelAtomic(context.Context, int64, int64) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) GetTicketLabels(context.Context, int64) ([]*ticket.Label, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) AddTicketLabel(context.Context, int64, int64) error { return nil }
func (m *mockTicketRepoForCrossUnit) RemoveTicketLabel(context.Context, int64, int64) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) GetRelation(context.Context, int64) (*ticket.Relation, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) CreateRelationPair(context.Context, *ticket.Relation, *ticket.Relation) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) DeleteRelationPair(context.Context, *ticket.Relation, string) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) ListRelations(context.Context, int64) ([]*ticket.Relation, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) CreateCommit(context.Context, *ticket.Commit) error { return nil }
func (m *mockTicketRepoForCrossUnit) DeleteCommit(context.Context, int64) error         { return nil }
func (m *mockTicketRepoForCrossUnit) ListCommitsByTicket(context.Context, int64) ([]*ticket.Commit, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetCommitBySHA(context.Context, int64, string) (*ticket.Commit, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) CreateMR(context.Context, *ticket.MergeRequest) error {
	return nil
}
func (m *mockTicketRepoForCrossUnit) UpdateMRState(context.Context, int64, string) error { return nil }
func (m *mockTicketRepoForCrossUnit) GetMRByURL(context.Context, string) (*ticket.MergeRequest, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) ListMRsByTicket(context.Context, int64) ([]*ticket.MergeRequest, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetActiveTickets(context.Context, int64, *int64, int) ([]*ticket.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetChildTickets(context.Context, int64) ([]*ticket.Ticket, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetSubTicketCounts(context.Context, []int64) (map[int64]map[string]int64, error) {
	return nil, nil
}
func (m *mockTicketRepoForCrossUnit) GetTicketStats(context.Context, int64, *int64) (map[string]int64, error) {
	return nil, nil
}

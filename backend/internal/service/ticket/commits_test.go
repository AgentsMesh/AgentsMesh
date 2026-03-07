package ticket

import (
	"context"
	"testing"
	"time"
)

func setupCommitsTestDB(t *testing.T) (*Service, context.Context) {
	db := setupTestDB(t)

	// Create commits table
	db.Exec(`CREATE TABLE IF NOT EXISTS ticket_commits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		ticket_id INTEGER NOT NULL,
		repository_id INTEGER NOT NULL,
		pod_id INTEGER,
		commit_sha TEXT NOT NULL,
		commit_message TEXT NOT NULL,
		commit_url TEXT,
		author_name TEXT,
		author_email TEXT,
		committed_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return newTestService(db), context.Background()
}

func TestLinkCommit(t *testing.T) {
	service, ctx := setupCommitsTestDB(t)

	// Create ticket first
	tkt, _ := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test",
		Priority:       "medium",
	})

	t.Run("link commit to ticket", func(t *testing.T) {
		url := "https://github.com/org/repo/commit/abc123"
		author := "John Doe"
		email := "john@example.com"
		now := time.Now()

		commit, err := service.LinkCommit(ctx, 1, tkt.ID, 1, nil, "abc123", "Fix bug", &url, &author, &email, &now)
		if err != nil {
			t.Fatalf("LinkCommit() error = %v", err)
		}
		if commit.CommitSHA != "abc123" {
			t.Errorf("CommitSHA = %s, want abc123", commit.CommitSHA)
		}
	})

	t.Run("link commit with pod", func(t *testing.T) {
		podID := int64(100)
		commit, err := service.LinkCommit(ctx, 1, tkt.ID, 1, &podID, "def456", "Add feature", nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("LinkCommit() error = %v", err)
		}
		if commit.PodID == nil || *commit.PodID != podID {
			t.Errorf("PodID mismatch")
		}
	})
}

func TestLinkCommit_NoTicketValidation(t *testing.T) {
	service, ctx := setupCommitsTestDB(t)

	// LinkCommit does not validate ticket existence (relies on DB constraints)
	// In SQLite without foreign key constraints, this will succeed
	commit, err := service.LinkCommit(ctx, 1, 99999, 1, nil, "abc", "msg", nil, nil, nil, nil)
	if err != nil {
		t.Logf("LinkCommit() returned error (expected if DB has foreign key): %v", err)
	} else if commit == nil {
		t.Error("expected commit to be created")
	}
}

func TestGetCommitBySHA(t *testing.T) {
	service, ctx := setupCommitsTestDB(t)

	tkt, _ := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test",
		Priority:       "medium",
	})

	service.LinkCommit(ctx, 1, tkt.ID, 1, nil, "unique-sha", "Commit msg", nil, nil, nil, nil)

	t.Run("found", func(t *testing.T) {
		commit, err := service.GetCommitBySHA(ctx, 1, "unique-sha")
		if err != nil {
			t.Fatalf("GetCommitBySHA() error = %v", err)
		}
		if commit.CommitSHA != "unique-sha" {
			t.Errorf("CommitSHA = %s, want unique-sha", commit.CommitSHA)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.GetCommitBySHA(ctx, 1, "nonexistent")
		if err != ErrCommitNotFound {
			t.Errorf("expected ErrCommitNotFound, got %v", err)
		}
	})
}

func TestListCommits(t *testing.T) {
	service, ctx := setupCommitsTestDB(t)

	tkt, _ := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test",
		Priority:       "medium",
	})

	// Link multiple commits
	for i, sha := range []string{"sha1", "sha2", "sha3"} {
		service.LinkCommit(ctx, 1, tkt.ID, 1, nil, sha, "Commit "+string(rune('A'+i)), nil, nil, nil, nil)
	}

	commits, err := service.ListCommits(ctx, tkt.ID)
	if err != nil {
		t.Fatalf("ListCommits() error = %v", err)
	}
	if len(commits) != 3 {
		t.Errorf("len = %d, want 3", len(commits))
	}
}

func TestUnlinkCommit(t *testing.T) {
	service, ctx := setupCommitsTestDB(t)

	tkt, _ := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Test",
		Priority:       "medium",
	})

	commit, _ := service.LinkCommit(ctx, 1, tkt.ID, 1, nil, "to-unlink", "Will be removed", nil, nil, nil, nil)

	err := service.UnlinkCommit(ctx, commit.ID)
	if err != nil {
		t.Fatalf("UnlinkCommit() error = %v", err)
	}

	commits, _ := service.ListCommits(ctx, tkt.ID)
	if len(commits) != 0 {
		t.Errorf("expected 0 commits after unlink, got %d", len(commits))
	}
}

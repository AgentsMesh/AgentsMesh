package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRelationsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			slug TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT,
			status TEXT NOT NULL DEFAULT 'backlog',
			priority TEXT NOT NULL DEFAULT 'none',
			severity TEXT,
			estimate INTEGER,
			due_date DATETIME,
			started_at DATETIME,
			completed_at DATETIME,
			repository_id INTEGER,
			reporter_id INTEGER NOT NULL,
			parent_ticket_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			source_ticket_id INTEGER NOT NULL,
			target_ticket_id INTEGER NOT NULL,
			relation_type TEXT NOT NULL DEFAULT 'relates',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestCreateRelation(t *testing.T) {
	ctx := context.Background()
	db := setupRelationsTestDB(t)
	service := newTestService(db)

	// Create two tickets
	ticket1 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "REL-1",
		Title:          "Ticket 1",
		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	ticket2 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "REL-2",
		Title:          "Ticket 2",
		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(ticket1)
	db.Create(ticket2)

	t.Run("creates relation and reverse relation", func(t *testing.T) {
		relation, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket2.ID, ticket.RelationTypeBlocks)
		require.NoError(t, err)
		assert.NotNil(t, relation)
		assert.Equal(t, ticket1.ID, relation.SourceTicketID)
		assert.Equal(t, ticket2.ID, relation.TargetTicketID)
		assert.Equal(t, ticket.RelationTypeBlocks, relation.RelationType)

		// Verify reverse relation was created
		var reverseRelation ticket.Relation
		err = db.Where("source_ticket_id = ? AND target_ticket_id = ?", ticket2.ID, ticket1.ID).First(&reverseRelation).Error
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeBlockedBy, reverseRelation.RelationType)
	})

	t.Run("prevents self-relation", func(t *testing.T) {
		_, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket1.ID, ticket.RelationTypeRelates)
		assert.Error(t, err)
		assert.Equal(t, ErrSelfRelation, err)
	})

	t.Run("creates relates relation", func(t *testing.T) {
		ticket3 := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "REL-3",
			Title:          "Ticket 3",
	
			Status:         ticket.TicketStatusTodo,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(ticket3)

		relation, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket3.ID, ticket.RelationTypeRelates)
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeRelates, relation.RelationType)

		// Verify reverse relation
		var reverseRelation ticket.Relation
		err = db.Where("source_ticket_id = ? AND target_ticket_id = ?", ticket3.ID, ticket1.ID).First(&reverseRelation).Error
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeRelates, reverseRelation.RelationType)
	})

	t.Run("creates blocked_by relation", func(t *testing.T) {
		ticket4 := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "REL-4",
			Title:          "Ticket 4",
	
			Status:         ticket.TicketStatusTodo,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(ticket4)

		relation, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket4.ID, ticket.RelationTypeBlockedBy)
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeBlockedBy, relation.RelationType)

		// Verify reverse relation is blocks
		var reverseRelation ticket.Relation
		err = db.Where("source_ticket_id = ? AND target_ticket_id = ?", ticket4.ID, ticket1.ID).First(&reverseRelation).Error
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeBlocks, reverseRelation.RelationType)
	})

	t.Run("creates duplicate relation", func(t *testing.T) {
		ticket5 := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "REL-5",
			Title:          "Ticket 5",
	
			Status:         ticket.TicketStatusTodo,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(ticket5)

		relation, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket5.ID, ticket.RelationTypeDuplicate)
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeDuplicate, relation.RelationType)

		// Verify reverse relation is also duplicate
		var reverseRelation ticket.Relation
		err = db.Where("source_ticket_id = ? AND target_ticket_id = ?", ticket5.ID, ticket1.ID).First(&reverseRelation).Error
		require.NoError(t, err)
		assert.Equal(t, ticket.RelationTypeDuplicate, reverseRelation.RelationType)
	})
}

func TestDeleteRelation(t *testing.T) {
	ctx := context.Background()
	db := setupRelationsTestDB(t)
	service := newTestService(db)

	// Create two tickets
	ticket1 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "DEL-1",
		Title:          "Ticket 1",

		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	ticket2 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "DEL-2",
		Title:          "Ticket 2",

		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(ticket1)
	db.Create(ticket2)

	t.Run("deletes relation and reverse relation", func(t *testing.T) {
		// Create relation
		relation, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket2.ID, ticket.RelationTypeBlocks)
		require.NoError(t, err)

		// Delete relation
		err = service.DeleteRelation(ctx, relation.ID)
		require.NoError(t, err)

		// Verify both relations are deleted
		var count int64
		db.Model(&ticket.Relation{}).Where("source_ticket_id = ? OR target_ticket_id = ?", ticket1.ID, ticket1.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("returns error for non-existent relation", func(t *testing.T) {
		err := service.DeleteRelation(ctx, 99999)
		assert.Error(t, err)
		assert.Equal(t, ErrRelationNotFound, err)
	})
}

func TestListRelations(t *testing.T) {
	ctx := context.Background()
	db := setupRelationsTestDB(t)
	service := newTestService(db)

	// Create tickets
	ticket1 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "LST-1",
		Title:          "Ticket 1",

		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	ticket2 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "LST-2",
		Title:          "Ticket 2",

		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	ticket3 := &ticket.Ticket{
		OrganizationID: 1,
		Slug:     "LST-3",
		Title:          "Ticket 3",

		Status:         ticket.TicketStatusTodo,
		Priority:       ticket.TicketPriorityMedium,
	}
	db.Create(ticket1)
	db.Create(ticket2)
	db.Create(ticket3)

	t.Run("lists relations for ticket", func(t *testing.T) {
		// Create relations
		_, err := service.CreateRelation(ctx, 1, ticket1.ID, ticket2.ID, ticket.RelationTypeBlocks)
		require.NoError(t, err)
		_, err = service.CreateRelation(ctx, 1, ticket1.ID, ticket3.ID, ticket.RelationTypeRelates)
		require.NoError(t, err)

		// List relations for ticket1
		relations, err := service.ListRelations(ctx, ticket1.ID)
		require.NoError(t, err)
		assert.Len(t, relations, 2)
	})

	t.Run("returns empty list for ticket without relations", func(t *testing.T) {
		ticket4 := &ticket.Ticket{
			OrganizationID: 1,
			Slug:     "LST-4",
			Title:          "Ticket 4",
	
			Status:         ticket.TicketStatusTodo,
			Priority:       ticket.TicketPriorityMedium,
		}
		db.Create(ticket4)

		relations, err := service.ListRelations(ctx, ticket4.ID)
		require.NoError(t, err)
		assert.Empty(t, relations)
	})
}

// Note: TestGetReverseRelationType is defined in service_extended_test.go

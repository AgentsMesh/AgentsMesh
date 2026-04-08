package mesh

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

func setupTestRepo(t *testing.T) (mesh.MeshRepository, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewMeshRepository(db)
	return repo, db
}

// ChannelPod for testing (local type to avoid import cycle)
type ChannelPod struct {
	ID        int64  `gorm:"primaryKey"`
	ChannelID int64  `gorm:"not null"`
	PodKey    string `gorm:"not null"`
}

func (ChannelPod) TableName() string {
	return "channel_pods"
}

// ChannelAccess for testing
type ChannelAccess struct {
	ID        int64   `gorm:"primaryKey"`
	ChannelID int64   `gorm:"not null"`
	PodKey    *string
	UserID    *int64
}

func (ChannelAccess) TableName() string {
	return "channel_access"
}

// Mock the channel.Message for count query
func init() {
	// Register table name mapping if needed
}

package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/file"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ file.FileRepository = (*fileRepository)(nil)

type fileRepository struct{ db *gorm.DB }

// NewFileRepository creates a new GORM-backed FileRepository.
func NewFileRepository(db *gorm.DB) file.FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(ctx context.Context, f *file.File) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *fileRepository) GetByID(ctx context.Context, id, orgID int64) (*file.File, error) {
	var f file.File
	err := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, orgID).
		First(&f).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *fileRepository) Delete(ctx context.Context, f *file.File) error {
	return r.db.WithContext(ctx).Delete(f).Error
}

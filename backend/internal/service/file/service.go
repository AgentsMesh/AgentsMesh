package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/file"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrFileNotFound     = errors.New("file not found")
	ErrFileTooLarge     = errors.New("file exceeds maximum size")
	ErrInvalidFileType  = errors.New("file type not allowed")
	ErrStorageError     = errors.New("storage operation failed")
)

// Service handles file operations
type Service struct {
	db      *gorm.DB
	storage storage.Storage
	config  config.StorageConfig
}

// NewService creates a new file service
func NewService(db *gorm.DB, storage storage.Storage, cfg config.StorageConfig) *Service {
	return &Service{
		db:      db,
		storage: storage,
		config:  cfg,
	}
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	OrganizationID int64
	UploaderID     int64
	FileName       string
	ContentType    string
	Size           int64
	Reader         io.Reader
}

// UploadResponse represents a file upload response
type UploadResponse struct {
	File *file.File `json:"file"`
	URL  string     `json:"url"`
}

// Upload uploads a file to storage and records metadata
func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	// Validate file size
	maxSize := s.config.MaxFileSize * 1024 * 1024 // Convert MB to bytes
	if req.Size > maxSize {
		return nil, fmt.Errorf("%w: max size is %d MB", ErrFileTooLarge, s.config.MaxFileSize)
	}

	// Validate file type
	if !s.isAllowedType(req.ContentType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidFileType, req.ContentType)
	}

	// Generate storage key
	storageKey := s.generateStorageKey(req.OrganizationID, req.FileName)

	// Upload to storage
	_, err := s.storage.Upload(ctx, storageKey, req.Reader, req.Size, req.ContentType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageError, err)
	}

	// Create file record
	f := &file.File{
		OrganizationID: req.OrganizationID,
		UploaderID:     req.UploaderID,
		OriginalName:   req.FileName,
		StorageKey:     storageKey,
		MimeType:       req.ContentType,
		Size:           req.Size,
	}

	if err := s.db.WithContext(ctx).Create(f).Error; err != nil {
		// Try to clean up uploaded file on database error
		_ = s.storage.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// Get presigned URL
	url, err := s.storage.GetURL(ctx, storageKey, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate URL: %w", err)
	}

	return &UploadResponse{
		File: f,
		URL:  url,
	}, nil
}

// GetByID retrieves a file by ID
func (s *Service) GetByID(ctx context.Context, id int64, orgID int64) (*file.File, error) {
	var f file.File
	err := s.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, orgID).
		First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return &f, nil
}

// GetURL returns a presigned URL for accessing a file
func (s *Service) GetURL(ctx context.Context, id int64, orgID int64, expiry time.Duration) (string, error) {
	f, err := s.GetByID(ctx, id, orgID)
	if err != nil {
		return "", err
	}

	return s.storage.GetURL(ctx, f.StorageKey, expiry)
}

// Delete removes a file from storage and database
func (s *Service) Delete(ctx context.Context, id int64, orgID int64) error {
	f, err := s.GetByID(ctx, id, orgID)
	if err != nil {
		return err
	}

	// Delete from storage first
	if err := s.storage.Delete(ctx, f.StorageKey); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageError, err)
	}

	// Delete database record
	if err := s.db.WithContext(ctx).Delete(f).Error; err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	return nil
}

// generateStorageKey generates a unique storage key for a file
func (s *Service) generateStorageKey(orgID int64, fileName string) string {
	// Extract file extension
	ext := path.Ext(fileName)
	if ext == "" {
		ext = ".bin"
	}

	// Generate unique ID
	id := uuid.New().String()

	// Format: orgs/{org_id}/files/{year}/{month}/{uuid}{ext}
	now := time.Now()
	return fmt.Sprintf("orgs/%d/files/%d/%02d/%s%s",
		orgID,
		now.Year(),
		now.Month(),
		id,
		ext,
	)
}

// isAllowedType checks if the content type is in the allowed list
func (s *Service) isAllowedType(contentType string) bool {
	// Normalize content type (remove parameters like charset)
	ct := strings.Split(contentType, ";")[0]
	ct = strings.TrimSpace(ct)

	for _, allowed := range s.config.AllowedTypes {
		if strings.EqualFold(ct, allowed) {
			return true
		}
	}
	return false
}

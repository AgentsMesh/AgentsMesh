package file

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/google/uuid"
)

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum size")
	ErrInvalidFileType = errors.New("file type not allowed")
	ErrStorageError    = errors.New("storage operation failed")
)

// Service handles file operations
type Service struct {
	storage storage.Storage
	config  config.StorageConfig
}

// NewService creates a new file service
func NewService(storage storage.Storage, cfg config.StorageConfig) *Service {
	return &Service{
		storage: storage,
		config:  cfg,
	}
}

// PresignUploadRequest represents a presigned upload request
type PresignUploadRequest struct {
	OrganizationID int64
	FileName       string
	ContentType    string
	Size           int64
}

// PresignUploadResponse represents a presigned upload response
type PresignUploadResponse struct {
	PutURL string `json:"put_url"`
	GetURL string `json:"get_url"`
}

// RequestPresignedUpload validates the request and returns presigned URLs for direct S3 upload
func (s *Service) RequestPresignedUpload(ctx context.Context, req *PresignUploadRequest) (*PresignUploadResponse, error) {
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

	// Get presigned PUT URL (15 min expiry)
	putURL, err := s.storage.PresignPutURL(ctx, storageKey, req.ContentType, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageError, err)
	}

	// Get presigned GET URL (24 hour expiry)
	getURL, err := s.storage.GetURL(ctx, storageKey, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GET URL: %w", err)
	}

	return &PresignUploadResponse{
		PutURL: putURL,
		GetURL: getURL,
	}, nil
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

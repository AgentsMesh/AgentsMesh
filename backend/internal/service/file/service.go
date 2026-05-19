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

type Service struct {
	storage storage.Storage
	config  config.StorageConfig
}

func NewService(storage storage.Storage, cfg config.StorageConfig) *Service {
	return &Service{
		storage: storage,
		config:  cfg,
	}
}

type PresignUploadRequest struct {
	OrganizationID int64
	FileName       string
	ContentType    string
	Size           int64
}

type PresignUploadResponse struct {
	PutURL string `json:"put_url"`
	GetURL string `json:"get_url"`
}

func (s *Service) RequestPresignedUpload(ctx context.Context, req *PresignUploadRequest) (*PresignUploadResponse, error) {
	maxSize := s.config.MaxFileSize * 1024 * 1024 // Convert MB to bytes
	if req.Size > maxSize {
		return nil, fmt.Errorf("%w: max size is %d MB", ErrFileTooLarge, s.config.MaxFileSize)
	}

	if !s.isAllowedType(req.ContentType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidFileType, req.ContentType)
	}

	storageKey := s.generateStorageKey(req.OrganizationID, req.FileName)

	putURL, err := s.storage.PresignPutURL(ctx, storageKey, req.ContentType, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageError, err)
	}

	getURL, err := s.storage.GetURL(ctx, storageKey, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GET URL: %w", err)
	}

	return &PresignUploadResponse{
		PutURL: putURL,
		GetURL: getURL,
	}, nil
}

func (s *Service) generateStorageKey(orgID int64, fileName string) string {
	ext := path.Ext(fileName)
	if ext == "" {
		ext = ".bin"
	}

	id := uuid.New().String()

	now := time.Now()
	return fmt.Sprintf("orgs/%d/files/%d/%02d/%s%s",
		orgID,
		now.Year(),
		now.Month(),
		id,
		ext,
	)
}

func (s *Service) isAllowedType(contentType string) bool {
	ct := strings.Split(contentType, ";")[0]
	ct = strings.TrimSpace(ct)

	for _, allowed := range s.config.AllowedTypes {
		if strings.EqualFold(ct, allowed) {
			return true
		}
	}
	return false
}

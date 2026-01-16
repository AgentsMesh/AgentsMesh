package file

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/file"
)

// MockService implements a mock file service for testing
type MockService struct {
	mu    sync.RWMutex
	files map[int64]*file.File
	nextID int64

	// Error injection
	UploadErr  error
	GetByIDErr error
	GetURLErr  error
	DeleteErr  error
}

// NewMockService creates a new mock file service
func NewMockService() *MockService {
	return &MockService{
		files:  make(map[int64]*file.File),
		nextID: 1,
	}
}

// Upload mocks file upload
func (m *MockService) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	if m.UploadErr != nil {
		return nil, m.UploadErr
	}

	// Read and discard data
	_, _ = io.Copy(io.Discard, req.Reader)

	m.mu.Lock()
	defer m.mu.Unlock()

	f := &file.File{
		ID:             m.nextID,
		OrganizationID: req.OrganizationID,
		UploaderID:     req.UploaderID,
		OriginalName:   req.FileName,
		StorageKey:     "mock/storage/key/" + req.FileName,
		MimeType:       req.ContentType,
		Size:           req.Size,
		CreatedAt:      time.Now(),
	}
	m.files[m.nextID] = f
	m.nextID++

	return &UploadResponse{
		File: f,
		URL:  "https://mock-storage.example.com/" + f.StorageKey,
	}, nil
}

// GetByID mocks getting a file by ID
func (m *MockService) GetByID(ctx context.Context, id int64, orgID int64) (*file.File, error) {
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	f, exists := m.files[id]
	if !exists || f.OrganizationID != orgID {
		return nil, ErrFileNotFound
	}

	return f, nil
}

// GetURL mocks getting a presigned URL
func (m *MockService) GetURL(ctx context.Context, id int64, orgID int64, expiry time.Duration) (string, error) {
	if m.GetURLErr != nil {
		return "", m.GetURLErr
	}

	f, err := m.GetByID(ctx, id, orgID)
	if err != nil {
		return "", err
	}

	return "https://mock-storage.example.com/" + f.StorageKey + "?expires=" + time.Now().Add(expiry).Format(time.RFC3339), nil
}

// Delete mocks file deletion
func (m *MockService) Delete(ctx context.Context, id int64, orgID int64) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	f, exists := m.files[id]
	if !exists || f.OrganizationID != orgID {
		return ErrFileNotFound
	}

	delete(m.files, id)
	return nil
}

// AddFile adds a file directly for testing
func (m *MockService) AddFile(f *file.File) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if f.ID == 0 {
		f.ID = m.nextID
		m.nextID++
	}
	m.files[f.ID] = f
}

// FileCount returns the number of files
func (m *MockService) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.files)
}

// Reset clears all files and errors
func (m *MockService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = make(map[int64]*file.File)
	m.nextID = 1
	m.UploadErr = nil
	m.GetByIDErr = nil
	m.GetURLErr = nil
	m.DeleteErr = nil
}

// SetUploadErr sets the upload error for testing
func (m *MockService) SetUploadErr(err error) {
	m.UploadErr = err
}

// SetGetByIDErr sets the GetByID error for testing
func (m *MockService) SetGetByIDErr(err error) {
	m.GetByIDErr = err
}

// SetDeleteErr sets the delete error for testing
func (m *MockService) SetDeleteErr(err error) {
	m.DeleteErr = err
}

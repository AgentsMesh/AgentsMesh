package file

import (
	"context"
	"sync"
)

// MockService implements a mock file service for testing
type MockService struct {
	mu         sync.RWMutex
	callCount  int
	PresignErr error
}

// NewMockService creates a new mock file service
func NewMockService() *MockService {
	return &MockService{}
}

// RequestPresignedUpload mocks presigned upload request
func (m *MockService) RequestPresignedUpload(ctx context.Context, req *PresignUploadRequest) (*PresignUploadResponse, error) {
	if m.PresignErr != nil {
		return nil, m.PresignErr
	}

	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	return &PresignUploadResponse{
		PutURL: "https://mock-storage.example.com/put/mock-key/" + req.FileName,
		GetURL: "https://mock-storage.example.com/mock-key/" + req.FileName,
	}, nil
}

// CallCount returns the number of presign requests
func (m *MockService) CallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// Reset clears state and errors
func (m *MockService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.PresignErr = nil
}

// SetPresignErr sets the presign error for testing
func (m *MockService) SetPresignErr(err error) {
	m.PresignErr = err
}

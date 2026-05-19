package file

import (
	"context"
	"sync"
)

type MockService struct {
	mu         sync.RWMutex
	callCount  int
	PresignErr error
}

func NewMockService() *MockService {
	return &MockService{}
}

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

func (m *MockService) CallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *MockService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.PresignErr = nil
}

func (m *MockService) SetPresignErr(err error) {
	m.PresignErr = err
}

package storage

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// MockStorage implements Storage interface for testing
type MockStorage struct {
	mu    sync.RWMutex
	files map[string]*mockFile

	// Error injection for testing
	UploadErr  error
	DeleteErr  error
	GetURLErr  error
	ExistsErr  error
}

type mockFile struct {
	key         string
	data        []byte
	contentType string
	size        int64
}

// NewMockStorage creates a new mock storage for testing
func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string]*mockFile),
	}
}

// Upload stores a file in memory
func (m *MockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*FileInfo, error) {
	if m.UploadErr != nil {
		return nil, m.UploadErr
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	m.mu.Lock()
	m.files[key] = &mockFile{
		key:         key,
		data:        data,
		contentType: contentType,
		size:        int64(len(data)),
	}
	m.mu.Unlock()

	return &FileInfo{
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		ETag:        fmt.Sprintf("mock-etag-%s", key),
	}, nil
}

// Delete removes a file from memory
func (m *MockStorage) Delete(ctx context.Context, key string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	m.mu.Lock()
	delete(m.files, key)
	m.mu.Unlock()

	return nil
}

// GetURL returns a mock presigned URL
func (m *MockStorage) GetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.GetURLErr != nil {
		return "", m.GetURLErr
	}

	m.mu.RLock()
	_, exists := m.files[key]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("file not found: %s", key)
	}

	return fmt.Sprintf("https://mock-storage.example.com/%s?expires=%d", key, time.Now().Add(expiry).Unix()), nil
}

// GetInternalURL returns a mock internal presigned URL (same as GetURL for mock)
func (m *MockStorage) GetInternalURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return m.GetURL(ctx, key, expiry)
}

// Exists checks if a file exists in memory
func (m *MockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsErr != nil {
		return false, m.ExistsErr
	}

	m.mu.RLock()
	_, exists := m.files[key]
	m.mu.RUnlock()

	return exists, nil
}

// GetFile returns file data for testing (not part of Storage interface)
func (m *MockStorage) GetFile(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	f, exists := m.files[key]
	if !exists {
		return nil, false
	}
	return f.data, true
}

// FileCount returns the number of stored files for testing
func (m *MockStorage) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.files)
}

// Clear removes all files for testing
func (m *MockStorage) Clear() {
	m.mu.Lock()
	m.files = make(map[string]*mockFile)
	m.mu.Unlock()
}

// Reset clears all files and errors
func (m *MockStorage) Reset() {
	m.mu.Lock()
	m.files = make(map[string]*mockFile)
	m.UploadErr = nil
	m.DeleteErr = nil
	m.GetURLErr = nil
	m.ExistsErr = nil
	m.mu.Unlock()
}

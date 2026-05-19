package storage

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type MockStorage struct {
	mu    sync.RWMutex
	files map[string]*mockFile

	UploadErr        error
	DeleteErr        error
	GetURLErr        error
	ExistsErr        error
	PresignPutURLErr error
}

type mockFile struct {
	key         string
	data        []byte
	contentType string
	size        int64
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string]*mockFile),
	}
}

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

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	m.mu.Lock()
	delete(m.files, key)
	m.mu.Unlock()

	return nil
}

func (m *MockStorage) GetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.GetURLErr != nil {
		return "", m.GetURLErr
	}

	return fmt.Sprintf("https://mock-storage.example.com/%s?expires=%d", key, time.Now().Add(expiry).Unix()), nil
}

func (m *MockStorage) GetInternalURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return m.GetURL(ctx, key, expiry)
}

func (m *MockStorage) PresignPutURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	if m.PresignPutURLErr != nil {
		return "", m.PresignPutURLErr
	}

	return fmt.Sprintf("https://mock-storage.example.com/%s?upload=true&expires=%d", key, time.Now().Add(expiry).Unix()), nil
}

func (m *MockStorage) InternalPresignPutURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	return m.PresignPutURL(ctx, key, contentType, expiry)
}

func (m *MockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsErr != nil {
		return false, m.ExistsErr
	}

	m.mu.RLock()
	_, exists := m.files[key]
	m.mu.RUnlock()

	return exists, nil
}

func (m *MockStorage) GetFile(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	f, exists := m.files[key]
	if !exists {
		return nil, false
	}
	return f.data, true
}

func (m *MockStorage) PutFile(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key] = &mockFile{key: key}
}

func (m *MockStorage) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.files)
}

func (m *MockStorage) Clear() {
	m.mu.Lock()
	m.files = make(map[string]*mockFile)
	m.mu.Unlock()
}

func (m *MockStorage) Reset() {
	m.mu.Lock()
	m.files = make(map[string]*mockFile)
	m.UploadErr = nil
	m.DeleteErr = nil
	m.GetURLErr = nil
	m.ExistsErr = nil
	m.PresignPutURLErr = nil
	m.mu.Unlock()
}

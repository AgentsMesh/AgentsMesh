package storage

import (
	"testing"
)

func TestS3ConfigStruct(t *testing.T) {
	cfg := S3Config{
		Endpoint:     "localhost:9000",
		Region:       "us-east-1",
		Bucket:       "test-bucket",
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		UsePathStyle: true,
	}

	if cfg.Endpoint != "localhost:9000" {
		t.Errorf("expected Endpoint 'localhost:9000', got %s", cfg.Endpoint)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("expected Region 'us-east-1', got %s", cfg.Region)
	}
	if cfg.Bucket != "test-bucket" {
		t.Errorf("expected Bucket 'test-bucket', got %s", cfg.Bucket)
	}
	if cfg.AccessKey != "minioadmin" {
		t.Errorf("expected AccessKey 'minioadmin', got %s", cfg.AccessKey)
	}
	if cfg.SecretKey != "minioadmin" {
		t.Errorf("expected SecretKey 'minioadmin', got %s", cfg.SecretKey)
	}
	if cfg.UseSSL != false {
		t.Errorf("expected UseSSL false, got %v", cfg.UseSSL)
	}
	if cfg.UsePathStyle != true {
		t.Errorf("expected UsePathStyle true, got %v", cfg.UsePathStyle)
	}
}

func TestS3ConfigAWSDefaults(t *testing.T) {
	// AWS S3 configuration (endpoint empty)
	cfg := S3Config{
		Endpoint:     "", // Empty for AWS default
		Region:       "us-west-2",
		Bucket:       "prod-bucket",
		AccessKey:    "aws-key",
		SecretKey:    "aws-secret",
		UseSSL:       true,
		UsePathStyle: false, // AWS uses virtual-hosted style
	}

	if cfg.Endpoint != "" {
		t.Errorf("expected empty Endpoint for AWS, got %s", cfg.Endpoint)
	}
	if cfg.UseSSL != true {
		t.Errorf("expected UseSSL true for AWS, got %v", cfg.UseSSL)
	}
	if cfg.UsePathStyle != false {
		t.Errorf("expected UsePathStyle false for AWS, got %v", cfg.UsePathStyle)
	}
}

func TestS3ConfigOSSCompatible(t *testing.T) {
	// Aliyun OSS S3-compatible configuration
	cfg := S3Config{
		Endpoint:     "oss-cn-hangzhou.aliyuncs.com",
		Region:       "oss-cn-hangzhou",
		Bucket:       "oss-bucket",
		AccessKey:    "oss-key",
		SecretKey:    "oss-secret",
		UseSSL:       true,
		UsePathStyle: false,
	}

	if cfg.Endpoint != "oss-cn-hangzhou.aliyuncs.com" {
		t.Errorf("expected Endpoint 'oss-cn-hangzhou.aliyuncs.com', got %s", cfg.Endpoint)
	}
	if cfg.Region != "oss-cn-hangzhou" {
		t.Errorf("expected Region 'oss-cn-hangzhou', got %s", cfg.Region)
	}
}

func TestFileInfoStruct(t *testing.T) {
	info := FileInfo{
		Key:         "orgs/1/files/2024/01/test.png",
		Size:        102400,
		ContentType: "image/png",
		ETag:        "abc123def456",
	}

	if info.Key != "orgs/1/files/2024/01/test.png" {
		t.Errorf("expected Key 'orgs/1/files/2024/01/test.png', got %s", info.Key)
	}
	if info.Size != 102400 {
		t.Errorf("expected Size 102400, got %d", info.Size)
	}
	if info.ContentType != "image/png" {
		t.Errorf("expected ContentType 'image/png', got %s", info.ContentType)
	}
	if info.ETag != "abc123def456" {
		t.Errorf("expected ETag 'abc123def456', got %s", info.ETag)
	}
}

// Note: Integration tests with actual S3/MinIO should be in a separate _integration_test.go file
// and run with build tags like: go test -tags=integration ./...

// TestNewS3Storage_InvalidConfig tests that invalid configs return errors
func TestNewS3Storage_EmptyCredentials(t *testing.T) {
	// Test with completely empty credentials - should still create client
	// (connection errors happen on first operation)
	cfg := S3Config{
		Endpoint:     "localhost:9000",
		Region:       "us-east-1",
		Bucket:       "test",
		AccessKey:    "",
		SecretKey:    "",
		UseSSL:       false,
		UsePathStyle: true,
	}

	storage, err := NewS3Storage(cfg)
	// SDK doesn't error on empty credentials during init
	// It will fail when actually trying to make requests
	if err != nil {
		t.Errorf("unexpected error on init: %v", err)
	}
	if storage == nil {
		t.Error("expected storage to be created even with empty credentials")
	}
}

func TestNewS3Storage_ValidConfig(t *testing.T) {
	cfg := S3Config{
		Endpoint:     "localhost:9000",
		Region:       "us-east-1",
		Bucket:       "test-bucket",
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		UsePathStyle: true,
	}

	storage, err := NewS3Storage(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if storage == nil {
		t.Error("expected storage to be created")
	}
	if storage.bucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got %s", storage.bucket)
	}
}

func TestNewS3Storage_HTTPSEndpoint(t *testing.T) {
	cfg := S3Config{
		Endpoint:     "storage.example.com",
		Region:       "us-east-1",
		Bucket:       "test-bucket",
		AccessKey:    "key",
		SecretKey:    "secret",
		UseSSL:       true,
		UsePathStyle: false,
	}

	storage, err := NewS3Storage(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if storage == nil {
		t.Error("expected storage to be created")
	}
	if storage.endpoint != "https://storage.example.com" {
		t.Errorf("expected endpoint 'https://storage.example.com', got %s", storage.endpoint)
	}
	if storage.useSSL != true {
		t.Errorf("expected useSSL true, got %v", storage.useSSL)
	}
}

func TestNewS3Storage_HTTPEndpoint(t *testing.T) {
	cfg := S3Config{
		Endpoint:     "localhost:9000",
		Region:       "us-east-1",
		Bucket:       "test-bucket",
		AccessKey:    "key",
		SecretKey:    "secret",
		UseSSL:       false,
		UsePathStyle: true,
	}

	storage, err := NewS3Storage(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if storage == nil {
		t.Error("expected storage to be created")
	}
	if storage.endpoint != "http://localhost:9000" {
		t.Errorf("expected endpoint 'http://localhost:9000', got %s", storage.endpoint)
	}
}

// Benchmark tests
func BenchmarkNewS3Storage(b *testing.B) {
	cfg := S3Config{
		Endpoint:     "localhost:9000",
		Region:       "us-east-1",
		Bucket:       "test-bucket",
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		UsePathStyle: true,
	}

	for i := 0; i < b.N; i++ {
		_, _ = NewS3Storage(cfg)
	}
}

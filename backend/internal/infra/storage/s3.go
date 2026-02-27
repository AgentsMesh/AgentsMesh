package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds configuration for S3-compatible storage.
type S3Config struct {
	Endpoint       string // S3 endpoint (empty for AWS, set for MinIO/OSS)
	PublicEndpoint string // Public endpoint for browser access (if different from Endpoint)
	Region         string // AWS region or equivalent
	Bucket         string // Bucket name
	AccessKey      string // Access key ID
	SecretKey      string // Secret access key
	UseSSL         bool   // Use HTTPS
	UsePathStyle   bool   // Use path-style URLs (required for MinIO)
}

// S3Storage implements Storage interface for S3-compatible services.
type S3Storage struct {
	client             *s3.Client
	presign            *s3.PresignClient
	bucket             string
	endpoint           string
	publicEndpoint     string // Full URL with scheme (e.g., https://oss-cn-beijing.aliyuncs.com)
	publicEndpointHost string // Host only without scheme (e.g., oss-cn-beijing.aliyuncs.com)
	useSSL             bool
	usePathStyle       bool // Use path-style URLs (required for MinIO)
}

// NewS3Storage creates a new S3-compatible storage client.
func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	// Build endpoint URL
	var endpointURL string
	if cfg.Endpoint != "" {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		endpointURL = fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)
	}

	// Create custom endpoint resolver if endpoint is specified
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.Endpoint != "" {
			return aws.Endpoint{
				URL:               endpointURL,
				HostnameImmutable: cfg.UsePathStyle,
			}, nil
		}
		// Return EndpointNotFoundError to fallback to default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with path-style option
	// Disable automatic checksum calculation to avoid aws-chunked encoding
	// which is not supported by Aliyun OSS
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	// Build public endpoint URL if specified
	publicEndpointURL := endpointURL
	publicEndpointHost := ""
	if cfg.PublicEndpoint != "" {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		publicEndpointURL = fmt.Sprintf("%s://%s", scheme, cfg.PublicEndpoint)
		publicEndpointHost = cfg.PublicEndpoint
	}

	return &S3Storage{
		client:             client,
		presign:            s3.NewPresignClient(client),
		bucket:             cfg.Bucket,
		endpoint:           endpointURL,
		publicEndpoint:     publicEndpointURL,
		publicEndpointHost: publicEndpointHost,
		useSSL:             cfg.UseSSL,
		usePathStyle:       cfg.UsePathStyle,
	}, nil
}

// Upload stores a file in S3.
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*FileInfo, error) {
	// Read all content into memory to avoid chunked encoding
	// This is required for Aliyun OSS compatibility as it doesn't support aws-chunked encoding
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(data))),
	}

	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	etag := ""
	if result.ETag != nil {
		etag = *result.ETag
	}

	return &FileInfo{
		Key:         key,
		Size:        size,
		ContentType: contentType,
		ETag:        etag,
	}, nil
}

// Delete removes a file from S3.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetURL returns a URL for accessing the file.
// If publicEndpoint differs from internal endpoint, returns a simple public URL (for public buckets).
// Otherwise, returns a pre-signed URL.
func (s *S3Storage) GetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// If public endpoint differs from internal endpoint, return simple public URL
	// This is used when bucket is configured for public/anonymous read access
	if s.publicEndpoint != "" && s.endpoint != "" && s.publicEndpoint != s.endpoint {
		scheme := "http"
		if s.useSSL {
			scheme = "https"
		}
		// Use path-style URL for MinIO: http://endpoint/bucket/key
		// Use virtual-hosted-style URL for Aliyun OSS: http://bucket.endpoint/key
		if s.usePathStyle {
			return fmt.Sprintf("%s://%s/%s/%s", scheme, s.publicEndpointHost, s.bucket, key), nil
		}
		return fmt.Sprintf("%s://%s.%s/%s", scheme, s.bucket, s.publicEndpointHost, key), nil
	}

	// Otherwise, generate pre-signed URL
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// GetInternalURL returns a pre-signed URL using the internal endpoint.
// This bypasses publicEndpoint and always generates a presigned URL with the internal endpoint,
// suitable for service-to-service downloads (e.g., Runner downloading skill packages within Docker network).
func (s *S3Storage) GetInternalURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate internal presigned URL: %w", err)
	}

	return request.URL, nil
}

// Exists checks if a file exists in S3.
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		// AWS SDK v2 doesn't have a typed error, so we check the error message
		return false, nil
	}
	return true, nil
}

// EnsureBucket creates the bucket if it doesn't exist.
func (s *S3Storage) EnsureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		return nil // Bucket exists
	}

	// Create bucket
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

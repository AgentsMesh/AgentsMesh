package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*FileInfo, error) {
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

func (s *S3Storage) GetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s.presign
	if s.publicPresign != nil {
		presigner = s.publicPresign
	}
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return request.URL, nil
}

func (s *S3Storage) presignGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return request.URL, nil
}

func (s *S3Storage) GetInternalURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return s.presignGetURL(ctx, key, expiry)
}

func (s *S3Storage) PresignPutURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	presigner := s.presign
	if s.publicPresign != nil {
		presigner = s.publicPresign
	}
	return s.presignPutURL(ctx, presigner, key, contentType, expiry)
}

func (s *S3Storage) InternalPresignPutURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	return s.presignPutURL(ctx, s.presign, key, contentType, expiry)
}

func (s *S3Storage) presignPutURL(ctx context.Context, presigner *s3.PresignClient, key string, contentType string, expiry time.Duration) (string, error) {
	request, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}
	return request.URL, nil
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *S3Storage) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		return nil
	}

	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}
	return nil
}

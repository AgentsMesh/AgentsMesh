package storage

import (
	"context"
	"fmt"

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
	publicPresign      *s3.PresignClient // Presign client using public endpoint (nil when same as internal)
	bucket             string
	endpoint           string
	publicEndpoint     string // Full URL with scheme
	publicEndpointHost string // Host only without scheme
	useSSL             bool
	usePathStyle       bool
}

// NewS3Storage creates a new S3-compatible storage client.
func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	endpointURL := buildEndpointURL(cfg.Endpoint, cfg.UseSSL)

	awsCfg, err := loadAWSConfig(cfg, endpointURL)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	publicEndpointURL, publicEndpointHost, publicPresignClient, err := buildPublicPresign(cfg, endpointURL)
	if err != nil {
		return nil, err
	}

	return &S3Storage{
		client:             client,
		presign:            s3.NewPresignClient(client),
		publicPresign:      publicPresignClient,
		bucket:             cfg.Bucket,
		endpoint:           endpointURL,
		publicEndpoint:     publicEndpointURL,
		publicEndpointHost: publicEndpointHost,
		useSSL:             cfg.UseSSL,
		usePathStyle:       cfg.UsePathStyle,
	}, nil
}

// buildEndpointURL constructs the endpoint URL from host and SSL flag.
func buildEndpointURL(endpoint string, useSSL bool) string {
	if endpoint == "" {
		return ""
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, endpoint)
}

// loadAWSConfig creates an AWS SDK config with the given S3 configuration.
func loadAWSConfig(cfg S3Config, endpointURL string) (aws.Config, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.Endpoint != "" {
			return aws.Endpoint{
				URL:               endpointURL,
				HostnameImmutable: cfg.UsePathStyle,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		)),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return awsCfg, nil
}

// buildPublicPresign creates a dedicated presign client for the public endpoint if needed.
func buildPublicPresign(cfg S3Config, endpointURL string) (string, string, *s3.PresignClient, error) {
	publicEndpointURL := endpointURL
	publicEndpointHost := ""

	if cfg.PublicEndpoint == "" {
		return publicEndpointURL, publicEndpointHost, nil, nil
	}

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	publicEndpointURL = fmt.Sprintf("%s://%s", scheme, cfg.PublicEndpoint)
	publicEndpointHost = cfg.PublicEndpoint

	if publicEndpointURL == endpointURL {
		return publicEndpointURL, publicEndpointHost, nil, nil
	}

	publicResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               publicEndpointURL,
			HostnameImmutable: cfg.UsePathStyle,
		}, nil
	})
	publicCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		)),
		config.WithEndpointResolverWithOptions(publicResolver),
	)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load public endpoint AWS config: %w", err)
	}

	publicClient := s3.NewFromConfig(publicCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	return publicEndpointURL, publicEndpointHost, s3.NewPresignClient(publicClient), nil
}

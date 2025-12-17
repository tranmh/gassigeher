package services

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Service handles S3-compatible object storage operations (Hetzner Object Storage)
type S3Service struct {
	client     *minio.Client
	bucketName string
	publicURL  string
}

// S3Config holds configuration for S3 service
type S3Config struct {
	Endpoint   string // e.g., "fsn1.your-objectstorage.com"
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
	PublicURL  string // e.g., "https://gassigeher-uploads.fsn1.your-objectstorage.com"
	UseSSL     bool
}

// NewS3Service creates a new S3 service instance
func NewS3Service(cfg *S3Config) (*S3Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("S3 config is nil")
	}

	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.BucketName == "" {
		return nil, fmt.Errorf("S3 config is incomplete: endpoint, access key, secret key, and bucket name are required")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &S3Service{
		client:     client,
		bucketName: cfg.BucketName,
		publicURL:  cfg.PublicURL,
	}, nil
}

// Upload uploads data to S3 and returns the public URL
// Path format: tenants/{slug}/{path}
func (s *S3Service) Upload(ctx context.Context, tenantSlug, path string, data []byte, contentType string) (string, error) {
	// Organize by tenant: tenants/{slug}/{path}
	objectKey := fmt.Sprintf("tenants/%s/%s", tenantSlug, path)

	_, err := s.client.PutObject(ctx, s.bucketName, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return public URL
	return fmt.Sprintf("%s/%s", s.publicURL, objectKey), nil
}

// Delete removes an object from S3
func (s *S3Service) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{})
}

// DeleteByPath deletes an object by tenant slug and path
func (s *S3Service) DeleteByPath(ctx context.Context, tenantSlug, path string) error {
	objectKey := fmt.Sprintf("tenants/%s/%s", tenantSlug, path)
	return s.Delete(ctx, objectKey)
}

// GetPresignedURL generates a presigned URL for temporary access
func (s *S3Service) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

// BucketExists checks if the configured bucket exists
func (s *S3Service) BucketExists(ctx context.Context) (bool, error) {
	return s.client.BucketExists(ctx, s.bucketName)
}

// GetObjectKey returns the full object key for a tenant path
func (s *S3Service) GetObjectKey(tenantSlug, path string) string {
	return fmt.Sprintf("tenants/%s/%s", tenantSlug, path)
}

// GetPublicURL returns the public URL for an object key
func (s *S3Service) GetPublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s", s.publicURL, objectKey)
}

package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSProvider implements storage.Provider for Aliyun OSS
type OSSProvider struct {
	client   *oss.Client
	bucket   *oss.Bucket
	endpoint string
	bucketName string
	domain   string // Custom domain or CDN domain
}

// NewOSSProvider creates a new OSS storage provider
// Endpoint: oss-cn-hangzhou.aliyuncs.com
func NewOSSProvider(endpoint, accessKeyID, accessKeySecret, bucketName, domain string) (*OSSProvider, error) {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket %s: %w", bucketName, err)
	}

	// Use bucket domain if custom domain is not provided
	if domain == "" {
		domain = fmt.Sprintf("https://%s.%s", bucketName, endpoint)
	} else {
		// Ensure protocol scheme
		if !strings.HasPrefix(domain, "http") {
			domain = "https://" + domain
		}
	}

	return &OSSProvider{
		client:     client,
		bucket:     bucket,
		endpoint:   endpoint,
		bucketName: bucketName,
		domain:     domain,
	}, nil
}

// Upload saves a file to OSS
func (p *OSSProvider) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
	// Remove leading slash if present to avoid empty folder
	objectKey := strings.TrimPrefix(path, "/")

	// OSS SDK doesn't support context directly in PutObject unless using latest version or wrapper
	// Basic implementation:
	err := p.bucket.PutObject(objectKey, file)
	if err != nil {
		return "", fmt.Errorf("failed to upload to OSS: %w", err)
	}

	// Return public URL
	return fmt.Sprintf("%s/%s", p.domain, objectKey), nil
}

// Delete removes a file from OSS
func (p *OSSProvider) Delete(ctx context.Context, path string) error {
	objectKey := strings.TrimPrefix(path, "/")
	err := p.bucket.DeleteObject(objectKey)
	if err != nil {
		return fmt.Errorf("failed to delete from OSS: %w", err)
	}
	return nil
}

// GetSignedURL generates a signed URL for private object access
func (p *OSSProvider) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	objectKey := strings.TrimPrefix(path, "/")

	// Calculate expiry in seconds
	expirySec := int64(expiry.Seconds())
	if expirySec <= 0 {
		expirySec = 3600 // Default 1 hour
	}

	url, err := p.bucket.SignURL(objectKey, oss.HTTPGet, expirySec)
	if err != nil {
		return "", fmt.Errorf("failed to sign URL: %w", err)
	}

	// If custom domain is set, replace the standard OSS domain in the signed URL
	if p.domain != "" && !strings.Contains(url, p.domain) {
		// This part can be tricky because SignURL returns a full URL with params
		// A simple string replace might work if standard domain structure is predictable
		// For now, returning the standard signed URL is safer
	}

	return url, nil
}

// Exists checks if a file exists in OSS
func (p *OSSProvider) Exists(ctx context.Context, path string) (bool, error) {
	objectKey := strings.TrimPrefix(path, "/")
	return p.bucket.IsObjectExist(objectKey)
}

func (p *OSSProvider) Name() string {
	return "oss"
}

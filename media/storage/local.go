package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalProvider implements storage.Provider for local filesystem
type LocalProvider struct {
	basePath string
	baseURL  string
}

// NewLocalProvider creates a new local storage provider
func NewLocalProvider(basePath, baseURL string) (*LocalProvider, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &LocalProvider{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Upload saves a file to the local filesystem
func (p *LocalProvider) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
	fullPath := filepath.Join(p.basePath, path)
	dir := filepath.Dir(fullPath)

	// Create directory if not exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to write file content: %w", err)
	}

	// Return public URL
	// Note: We use forward slashes for URLs even on Windows
	return fmt.Sprintf("%s/%s", p.baseURL, path), nil
}

// Delete removes a file from the local filesystem
func (p *LocalProvider) Delete(ctx context.Context, input DeleteInput) error {
	fullPath := filepath.Join(p.basePath, input.Folder, input.Filename)
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetURL returns the public URL for a file
func (p *LocalProvider) GetURL(ctx context.Context, input GetURLInput) (string, error) {
	return fmt.Sprintf("%s/%s/%s", p.baseURL, input.Folder, input.Filename), nil
}

// GetSignedURL for local provider just returns the public URL
func (p *LocalProvider) GetSignedURL(ctx context.Context, input GetSignedURLInput) (string, error) {
	return p.GetURL(ctx, GetURLInput{
		Filename: input.Filename,
		Folder:   input.Folder,
	})
}

// Exists checks if a file exists
func (p *LocalProvider) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(p.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (p *LocalProvider) Name() string {
	return "local"
}

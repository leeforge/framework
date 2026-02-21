package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// StorageProvider 存储提供者接口
type StorageProvider interface {
	Upload(ctx context.Context, input UploadInput) (UploadOutput, error)
	Delete(ctx context.Context, input DeleteInput) error
	GetURL(ctx context.Context, input GetURLInput) (string, error)
	GetSignedURL(ctx context.Context, input GetSignedURLInput) (string, error)
	Move(ctx context.Context, input MoveInput) error
	List(ctx context.Context, input ListInput) ([]FileInfo, error)
}

// UploadInput 上传输入
type UploadInput struct {
	File      io.Reader
	Filename  string
	Folder    string
	IsPrivate bool
	Metadata  map[string]interface{}
	Size      int64
}

// UploadOutput 上传输出
type UploadOutput struct {
	URL      string
	Filename string
	Size     int64
	Metadata map[string]interface{}
}

// DeleteInput 删除输入
type DeleteInput struct {
	Filename string
	Folder   string
}

// GetURLInput 获取 URL 输入
type GetURLInput struct {
	Filename string
	Folder   string
}

// GetSignedURLInput 获取签名 URL 输入
type GetSignedURLInput struct {
	Filename string
	Folder   string
	Expires  int64 // 过期时间（秒）
}

// MoveInput 移动输入
type MoveInput struct {
	Source      string
	Destination string
}

// ListInput 列表输入
type ListInput struct {
	Folder string
	Prefix string
	Limit  int
}

// FileInfo 文件信息
type FileInfo struct {
	Name      string
	Size      int64
	IsDir     bool
	UpdatedAt int64
	Metadata  map[string]interface{}
}

// ProviderConfig 提供者配置
type ProviderConfig struct {
	Type     string                 `json:"type"`     // local, s3, cloudinary
	Settings map[string]interface{} `json:"settings"` // 各提供者特定配置
}

// LocalStorageProvider 本地存储提供者（使用新接口）
type LocalStorageProvider struct {
	basePath string
	baseURL  string
}

// NewLocalStorageProvider 创建本地存储提供者
func NewLocalStorageProvider(basePath, baseURL string) (*LocalStorageProvider, error) {
	// 创建基础目录
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalStorageProvider{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Upload 上传文件
func (p *LocalStorageProvider) Upload(ctx context.Context, input UploadInput) (UploadOutput, error) {
	// 构建完整路径
	fullPath := filepath.Join(p.basePath, input.Folder, input.Filename)

	// 创建目录
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return UploadOutput{}, fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(fullPath)
	if err != nil {
		return UploadOutput{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 复制内容
	size, err := io.Copy(file, input.File)
	if err != nil {
		return UploadOutput{}, fmt.Errorf("failed to copy file: %w", err)
	}

	// 构建 URL
	url := fmt.Sprintf("%s/%s/%s", p.baseURL, input.Folder, input.Filename)

	return UploadOutput{
		URL:      url,
		Filename: input.Filename,
		Size:     size,
		Metadata: input.Metadata,
	}, nil
}

// Delete 删除文件
func (p *LocalStorageProvider) Delete(ctx context.Context, input DeleteInput) error {
	fullPath := filepath.Join(p.basePath, input.Folder, input.Filename)
	return os.Remove(fullPath)
}

// GetURL 获取文件 URL
func (p *LocalStorageProvider) GetURL(ctx context.Context, input GetURLInput) (string, error) {
	return fmt.Sprintf("%s/%s/%s", p.baseURL, input.Folder, input.Filename), nil
}

// GetSignedURL 获取签名 URL（本地存储不需要签名）
func (p *LocalStorageProvider) GetSignedURL(ctx context.Context, input GetSignedURLInput) (string, error) {
	return p.GetURL(ctx, GetURLInput{
		Filename: input.Filename,
		Folder:   input.Folder,
	})
}

// Move 移动文件
func (p *LocalStorageProvider) Move(ctx context.Context, input MoveInput) error {
	sourcePath := filepath.Join(p.basePath, input.Source)
	destPath := filepath.Join(p.basePath, input.Destination)

	// 创建目标目录
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return os.Rename(sourcePath, destPath)
}

// List 列出文件
func (p *LocalStorageProvider) List(ctx context.Context, input ListInput) ([]FileInfo, error) {
	folder := filepath.Join(p.basePath, input.Folder, input.Prefix)

	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileInfo
	limit := input.Limit
	if limit == 0 {
		limit = len(entries)
	}

	for i, entry := range entries {
		if i >= limit {
			break
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, FileInfo{
			Name:      entry.Name(),
			Size:      info.Size(),
			IsDir:     entry.IsDir(),
			UpdatedAt: info.ModTime().Unix(),
		})
	}

	return files, nil
}

// S3Provider S3 存储提供者（接口定义，具体实现需要 AWS SDK）
type S3Provider struct {
	// 实际实现需要 AWS SDK
	// 这里只保留接口定义
}

// CloudinaryProvider Cloudinary 存储提供者（接口定义）
type CloudinaryProvider struct {
	// 实际实现需要 Cloudinary SDK
	// 这里只保留接口定义
}

// ProviderFactory 提供者工厂
type ProviderFactory struct {
	providers map[string]StorageProvider
}

// NewProviderFactory 创建工厂
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: make(map[string]StorageProvider),
	}
}

// Register 注册提供者
func (f *ProviderFactory) Register(name string, provider StorageProvider) {
	f.providers[name] = provider
}

// Get 获取提供者
func (f *ProviderFactory) Get(name string) (StorageProvider, error) {
	provider, exists := f.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// CreateFromConfig 从配置创建提供者
func (f *ProviderFactory) CreateFromConfig(config ProviderConfig) (StorageProvider, error) {
	switch config.Type {
	case "local":
		basePath, ok := config.Settings["base_path"].(string)
		if !ok {
			return nil, fmt.Errorf("local provider requires base_path")
		}
		baseURL, ok := config.Settings["base_url"].(string)
		if !ok {
			baseURL = "/media" // 默认值
		}
		return NewLocalStorageProvider(basePath, baseURL)

	case "s3":
		return nil, fmt.Errorf("s3 provider requires aws-sdk-go, not implemented in this version")

	case "cloudinary":
		return nil, fmt.Errorf("cloudinary provider not implemented in this version")

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

package processor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	config ImageConfig
}

// ImageConfig 图片配置
type ImageConfig struct {
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
	Quality   int    `json:"quality"`   // 1-100
	Format    string `json:"format"`    // jpeg, png, webp
	Watermark string `json:"watermark"` // 水印文字
	Compress  bool   `json:"compress"`  // 是否压缩
	Grayscale bool   `json:"grayscale"` // 是否灰度
}

// ProcessingPipeline 处理管道
type ProcessingPipeline struct {
	steps []ProcessingStep
}

// ProcessingStep 处理步骤接口
type ProcessingStep interface {
	Process(ctx context.Context, image []byte) ([]byte, error)
}

// ValidateStep 验证步骤
type ValidateStep struct{}

// ExtractMetadataStep 提取元数据步骤
type ExtractMetadataStep struct{}

// ResizeStep 调整大小步骤
type ResizeStep struct {
	width  int
	height int
}

// CompressStep 压缩步骤
type CompressStep struct {
	quality int
}

// FormatStep 格式转换步骤
type FormatStep struct {
	format string
}

// WatermarkStep 水印步骤
type WatermarkStep struct {
	text string
}

// GrayscaleStep 灰度转换步骤
type GrayscaleStep struct{}

// NewImageProcessor 创建图片处理器
func NewImageProcessor(config ImageConfig) *ImageProcessor {
	return &ImageProcessor{
		config: config,
	}
}

// NewProcessingPipeline 创建处理管道
func NewProcessingPipeline(config ImageConfig) *ProcessingPipeline {
	steps := []ProcessingStep{}

	// 添加步骤
	steps = append(steps, ValidateStep{})
	steps = append(steps, ExtractMetadataStep{})

	if config.MaxWidth > 0 || config.MaxHeight > 0 {
		steps = append(steps, ResizeStep{
			width:  config.MaxWidth,
			height: config.MaxHeight,
		})
	}

	if config.Grayscale {
		steps = append(steps, GrayscaleStep{})
	}

	if config.Compress || config.Quality > 0 {
		quality := config.Quality
		if quality == 0 {
			quality = 85
		}
		steps = append(steps, CompressStep{quality: quality})
	}

	if config.Format != "" {
		steps = append(steps, FormatStep{format: config.Format})
	}

	if config.Watermark != "" {
		steps = append(steps, WatermarkStep{text: config.Watermark})
	}

	return &ProcessingPipeline{
		steps: steps,
	}
}

// Process 处理图片
func (p *ProcessingPipeline) Process(ctx context.Context, input []byte) ([]byte, error) {
	data := input
	for _, step := range p.steps {
		var err error
		data, err = step.Process(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("step failed: %w", err)
		}
	}
	return data, nil
}

// Process 处理单个图片
func (p *ImageProcessor) Process(ctx context.Context, input []byte) ([]byte, error) {
	pipeline := NewProcessingPipeline(p.config)
	return pipeline.Process(ctx, input)
}

// ProcessFromFile 从文件处理
func (p *ImageProcessor) ProcessFromFile(ctx context.Context, filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return p.Process(ctx, data)
}

// ProcessFromReader 从 Reader 处理
func (p *ImageProcessor) ProcessFromReader(ctx context.Context, reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return p.Process(ctx, data)
}

// GetDimensions 获取图片尺寸（实现 Resizer 接口）
func (p *ImageProcessor) GetDimensions(reader io.Reader) (int, int, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, 0, err
	}
	width, height, _, err := GetImageInfo(data)
	return width, height, err
}

// Resize 调整图片大小（实现 Resizer 接口）
func (p *ImageProcessor) Resize(reader io.Reader, width, height uint) (io.Reader, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	config := p.config
	config.MaxWidth = int(width)
	config.MaxHeight = int(height)

	tempProcessor := NewImageProcessor(config)
	result, err := tempProcessor.Process(context.Background(), data)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(result), nil
}

// ValidateStep 实现
func (s ValidateStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	// 检测图片格式
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	// 支持的格式
	supported := map[string]bool{
		"jpeg": true,
		"jpg":  true,
		"png":  true,
		"webp": true,
		"bmp":  true,
		"tiff": true,
	}

	if !supported[format] {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return data, nil
}

// ExtractMetadataStep 实现
func (s ExtractMetadataStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	// 简化实现：只验证图片
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// 可以在这里提取更多元数据
	// 例如：EXIF 信息等

	// 打印元数据（调试用）
	fmt.Printf("Image metadata: %dx%d\n", config.Width, config.Height)

	return data, nil
}

// ResizeStep 实现
func (s ResizeStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 计算新尺寸
	newWidth, newHeight := s.calculateNewSize(width, height)

	// 如果不需要调整
	if newWidth == width && newHeight == height {
		return data, nil
	}

	// 创建新图片
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// 缩放
	// 简化实现：使用最近邻插值
	// 实际可以使用更高质量的插值算法
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := x * width / newWidth
			srcY := y * height / newHeight
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	// 编码回字节
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, newImg, nil); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(&buf, newImg); err != nil {
			return nil, err
		}
	default:
		// 默认使用 JPEG
		if err := jpeg.Encode(&buf, newImg, nil); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// calculateNewSize 计算新尺寸
func (s ResizeStep) calculateNewSize(width, height int) (int, int) {
	newWidth := width
	newHeight := height

	if s.width > 0 && width > s.width {
		newWidth = s.width
		newHeight = height * s.width / width
	}

	if s.height > 0 && newHeight > s.height {
		newHeight = s.height
		newWidth = width * s.height / height
	}

	return newWidth, newHeight
}

// CompressStep 实现
func (s CompressStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	quality := s.quality
	if quality == 0 {
		quality = 85
	}

	switch format {
	case "jpeg", "jpg":
		opt := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(&buf, img, opt); err != nil {
			return nil, err
		}
	case "png":
		// PNG 压缩通过减少颜色深度
		// 简化实现：直接编码
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	default:
		// 默认使用 JPEG
		opt := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(&buf, img, opt); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// FormatStep 实现
func (s FormatStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	switch strings.ToLower(s.format) {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case "webp":
		// WebP 需要额外库，这里使用 PNG 作为占位
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported output format: %s", s.format)
	}

	return buf.Bytes(), nil
}

// WatermarkStep 实现
func (s WatermarkStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()

	// 创建新图片（支持透明度）
	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, bounds, img, bounds.Min, draw.Src)

	// 添加水印
	// 简化实现：在右下角添加文字
	// 实际可以使用 font.DrawString
	watermarkText := s.text
	textLen := len(watermarkText)

	// 计算位置
	x := bounds.Dx() - textLen*10 - 10
	y := bounds.Dy() - 20

	// 绘制简单文字（使用像素点模拟）
	// 实际应该使用字体库
	for i, ch := range watermarkText {
		if x+i*10+5 >= bounds.Dx() {
			break
		}
		// 绘制一个简单的矩形作为文字占位
		for dy := 0; dy < 10; dy++ {
			for dx := 0; dx < 8; dx++ {
				if dx < 6 && dy < 8 {
					newImg.Set(x+i*10+dx, y+dy, color.RGBA{255, 255, 255, 128})
				}
			}
		}
		_ = ch // 使用变量
	}

	// 编码回字节
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, newImg, nil); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(&buf, newImg); err != nil {
			return nil, err
		}
	default:
		if err := jpeg.Encode(&buf, newImg, nil); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// GrayscaleStep 实现
func (s GrayscaleStep) Process(ctx context.Context, data []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	newImg := image.NewGray(bounds)

	// 转换为灰度
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(x, y, img.At(x, y))
		}
	}

	// 编码回字节
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, newImg, nil); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(&buf, newImg); err != nil {
			return nil, err
		}
	default:
		if err := jpeg.Encode(&buf, newImg, nil); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// GetImageInfo 获取图片信息
func GetImageInfo(data []byte) (width, height int, format string, err error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, "", err
	}
	return config.Width, config.Height, format, nil
}

// GenerateThumbnail 生成缩略图
func GenerateThumbnail(data []byte, maxSize int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 计算缩略图尺寸
	var thumbWidth, thumbHeight int
	if width > height {
		thumbWidth = maxSize
		thumbHeight = height * maxSize / width
	} else {
		thumbHeight = maxSize
		thumbWidth = width * maxSize / height
	}

	// 使用 ResizeStep
	resize := ResizeStep{width: thumbWidth, height: thumbHeight}
	return resize.Process(context.Background(), data)
}

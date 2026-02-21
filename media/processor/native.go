package processor

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/nfnt/resize"
)

// Resizer defines interface for simple image resizing
type Resizer interface {
	// Resize creates a thumbnail with specified dimensions
	Resize(reader io.Reader, width, height uint) (io.Reader, error)
	// GetDimensions returns image width and height
	GetDimensions(reader io.Reader) (int, int, error)
}

// NativeProcessor implements Resizer using pure Go libraries
// This avoids CGO dependency (libvips) for easier deployment
type NativeProcessor struct{}

func NewNativeProcessor() *NativeProcessor {
	return &NativeProcessor{}
}

func (p *NativeProcessor) Resize(reader io.Reader, width, height uint) (io.Reader, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	// Resize using Lanczos3 resampling for high quality
	thumbnail := resize.Thumbnail(width, height, img, resize.Lanczos3)

	var buf bytes.Buffer
	// Encode as JPEG by default for thumbnails to save space
	// In a real system, we might want to preserve the original format
	if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return &buf, nil
}

func (p *NativeProcessor) GetDimensions(reader io.Reader) (int, int, error) {
	// We only need the config, not the whole image
	config, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}

// init registers format decoders
func init() {
	// Ensure formats are registered
	_ = jpeg.Decode
	_ = png.Decode
}

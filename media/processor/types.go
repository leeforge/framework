package processor

// PipelineImageProcessor defines the interface for pipeline-based image processing
type PipelineImageProcessor interface {
	Process(ctx interface{}, input []byte) ([]byte, error)
	ProcessFromFile(ctx interface{}, filePath string) ([]byte, error)
}

// ResizeOptions defines options for image resizing
type ResizeOptions struct {
	Width   uint
	Height  uint
	Quality int
	Format  string
}

// Standard sizes
var (
	Thumbnail = ResizeOptions{Width: 245, Height: 156, Quality: 80}
	Small     = ResizeOptions{Width: 500, Height: 500, Quality: 85}
	Medium    = ResizeOptions{Width: 750, Height: 750, Quality: 85}
	Large     = ResizeOptions{Width: 1000, Height: 1000, Quality: 90}
)

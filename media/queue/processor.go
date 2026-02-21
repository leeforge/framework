package queue

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/leeforge/framework/media/processor"
	"github.com/leeforge/framework/media/storage"
)

// AsyncProcessor 异步处理器
type AsyncProcessor struct {
	workerCount int
	jobQueue    chan ProcessingJob
	storage     storage.StorageProvider
	processor   *processor.ImageProcessor
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// ProcessingJob 处理任务
type ProcessingJob struct {
	FileID     int
	InputPath  string
	OutputPath string
	Config     processor.ImageConfig
	Callback   func(result JobResult)
	RetryCount int
}

// JobResult 任务结果
type JobResult struct {
	Success bool
	Error   error
	Formats map[string]FormatInfo
	FileID  int
}

// FormatInfo 格式信息
type FormatInfo struct {
	URL  string
	Size int64
}

// NewAsyncProcessor 创建异步处理器
func NewAsyncProcessor(workerCount int, storage storage.StorageProvider, proc *processor.ImageProcessor) *AsyncProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncProcessor{
		workerCount: workerCount,
		jobQueue:    make(chan ProcessingJob, 100),
		storage:     storage,
		processor:   proc,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动处理器
func (p *AsyncProcessor) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker 工作协程
func (p *AsyncProcessor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				return
			}
			p.processJob(job)
		}
	}
}

// processJob 处理单个任务
func (p *AsyncProcessor) processJob(job ProcessingJob) {
	var result JobResult
	result.FileID = job.FileID

	// 重试逻辑
	for attempt := 0; attempt <= job.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// 读取输入文件
		inputData, err := p.readInput(job.InputPath)
		if err != nil {
			result.Error = fmt.Errorf("failed to read input: %w", err)
			continue
		}

		// 处理图片
		outputData, err := p.processor.Process(p.ctx, inputData)
		if err != nil {
			result.Error = fmt.Errorf("processing failed: %w", err)
			continue
		}

		// 上传到存储
		url, err := p.uploadOutput(job.OutputPath, outputData)
		if err != nil {
			result.Error = fmt.Errorf("upload failed: %w", err)
			continue
		}

		// 成功
		result.Success = true
		result.Formats = map[string]FormatInfo{
			"original": {
				URL:  url,
				Size: int64(len(outputData)),
			},
		}

		break
	}

	// 调用回调
	if job.Callback != nil {
		job.Callback(result)
	}
}

// readInput 读取输入
func (p *AsyncProcessor) readInput(path string) ([]byte, error) {
	// 如果是文件路径
	if path != "" {
		return os.ReadFile(path)
	}
	return nil, fmt.Errorf("no input path provided")
}

// uploadOutput 上传输出
func (p *AsyncProcessor) uploadOutput(path string, data []byte) (string, error) {
	// 上传到存储
	reader := bytes.NewReader(data)

	output, err := p.storage.Upload(p.ctx, storage.UploadInput{
		File:     reader,
		Filename: path,
		Folder:   "processed",
		Size:     int64(len(data)),
	})
	if err != nil {
		return "", err
	}

	return output.URL, nil
}

// Submit 提交任务
func (p *AsyncProcessor) Submit(job ProcessingJob) error {
	select {
	case p.jobQueue <- job:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("processor is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

// SubmitBatch 批量提交任务
func (p *AsyncProcessor) SubmitBatch(jobs []ProcessingJob) error {
	for _, job := range jobs {
		if err := p.Submit(job); err != nil {
			return err
		}
	}
	return nil
}

// Stop 停止处理器
func (p *AsyncProcessor) Stop() error {
	// 停止接收新任务
	p.cancel()

	// 关闭队列
	close(p.jobQueue)

	// 等待所有任务完成
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// 等待或超时
	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for jobs to complete")
	}
}

// GetQueueSize 获取队列大小
func (p *AsyncProcessor) GetQueueSize() int {
	return len(p.jobQueue)
}

// GetPendingCount 获取待处理任务数
func (p *AsyncProcessor) GetPendingCount() int {
	return len(p.jobQueue)
}

// ResizeJob 调整大小任务
type ResizeJob struct {
	Width  int
	Height int
}

// BatchProcessor 批量处理器
type BatchProcessor struct {
	processor *AsyncProcessor
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(processor *AsyncProcessor) *BatchProcessor {
	return &BatchProcessor{
		processor: processor,
	}
}

// ProcessBatch 批量处理
func (b *BatchProcessor) ProcessBatch(jobs []ProcessingJob) ([]JobResult, error) {
	results := make([]JobResult, len(jobs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, job := range jobs {
		wg.Add(1)

		// 为每个任务创建回调
		job.Callback = func(result JobResult) {
			defer wg.Done()
			mu.Lock()
			results[i] = result
			mu.Unlock()
		}

		if err := b.processor.Submit(job); err != nil {
			wg.Done()
			return nil, fmt.Errorf("failed to submit job %d: %w", i, err)
		}
	}

	// 等待所有任务完成
	wg.Wait()

	return results, nil
}

// ProgressTracker 进度追踪器
type ProgressTracker struct {
	total     int
	completed int
	failed    int
	mu        sync.RWMutex
}

// NewProgressTracker 创建进度追踪器
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total: total,
	}
}

// IncrementCompleted 增加完成数
func (t *ProgressTracker) IncrementCompleted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.completed++
}

// IncrementFailed 增加失败数
func (t *ProgressTracker) IncrementFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed++
}

// GetProgress 获取进度
func (t *ProgressTracker) GetProgress() (completed, failed, total int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.completed, t.failed, t.total
}

// GetPercentage 获取百分比
func (t *ProgressTracker) GetPercentage() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.total == 0 {
		return 0
	}
	return float64(t.completed+t.failed) / float64(t.total) * 100
}

// JobManager 任务管理器
type JobManager struct {
	processor *AsyncProcessor
	tracker   *ProgressTracker
}

// NewJobManager 创建任务管理器
func NewJobManager(processor *AsyncProcessor, totalJobs int) *JobManager {
	return &JobManager{
		processor: processor,
		tracker:   NewProgressTracker(totalJobs),
	}
}

// ProcessWithProgress 带进度的处理
func (jm *JobManager) ProcessWithProgress(jobs []ProcessingJob) ([]JobResult, error) {
	results := make([]JobResult, len(jobs))
	var mu sync.Mutex

	for i, job := range jobs {
		// 包装回调以追踪进度
		originalCallback := job.Callback
		job.Callback = func(result JobResult) {
			mu.Lock()
			results[i] = result
			mu.Unlock()

			if result.Success {
				jm.tracker.IncrementCompleted()
			} else {
				jm.tracker.IncrementFailed()
			}

			// 调用原始回调
			if originalCallback != nil {
				originalCallback(result)
			}
		}

		if err := jm.processor.Submit(job); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// GetProgress 获取处理进度
func (jm *JobManager) GetProgress() (completed, failed, total int, percentage float64) {
	c, f, t := jm.tracker.GetProgress()
	return c, f, t, jm.tracker.GetPercentage()
}

// Wait 等待所有任务完成
func (jm *JobManager) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			c, f, t := jm.tracker.GetProgress()
			if c+f >= t {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

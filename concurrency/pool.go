package concurrency

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkerPool Worker 池
type WorkerPool struct {
	size       int
	jobQueue   chan Job
	resultChan chan Result
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// Job 任务接口
type Job interface {
	Execute() error
}

// Result 结果接口
type Result interface {
	GetError() error
}

// Worker 工作协程
type Worker struct {
	id         int
	jobQueue   chan Job
	resultChan chan Result
	wg         *sync.WaitGroup
	ctx        context.Context
}

// NewWorkerPool 创建 Worker 池
func NewWorkerPool(size int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		size:       size,
		jobQueue:   make(chan Job, 100),
		resultChan: make(chan Result, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动 Worker 池
func (p *WorkerPool) Start() {
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		worker := &Worker{
			id:         i,
			jobQueue:   p.jobQueue,
			resultChan: p.resultChan,
			wg:         &p.wg,
			ctx:        p.ctx,
		}
		go worker.start()
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(job Job) error {
	select {
	case p.jobQueue <- job:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("pool is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

// SubmitBatch 批量提交任务
func (p *WorkerPool) SubmitBatch(jobs []Job) error {
	for _, job := range jobs {
		if err := p.Submit(job); err != nil {
			return err
		}
	}
	return nil
}

// GetResults 获取结果通道
func (p *WorkerPool) GetResults() <-chan Result {
	return p.resultChan
}

// Wait 等待所有任务完成
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Stop 停止 Worker 池
func (p *WorkerPool) Stop() error {
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

	select {
	case <-done:
		close(p.resultChan)
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for workers to finish")
	}
}

// start 启动 Worker
func (w *Worker) start() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				return
			}
			result := w.execute(job)
			w.resultChan <- result
		}
	}
}

// execute 执行任务
func (w *Worker) execute(job Job) Result {
	err := job.Execute()
	return &SimpleResult{err: err}
}

// SimpleResult 简单结果
type SimpleResult struct {
	err error
}

// GetError 获取错误
func (r *SimpleResult) GetError() error {
	return r.err
}

// JobFunc 函数式任务
type JobFunc func() error

// Execute 执行函数
func (f JobFunc) Execute() error {
	return f()
}

// PooledExecutor 池化执行器
type PooledExecutor struct {
	pool *WorkerPool
}

// NewPooledExecutor 创建池化执行器
func NewPooledExecutor(size int) *PooledExecutor {
	return &PooledExecutor{
		pool: NewWorkerPool(size),
	}
}

// Execute 执行任务
func (e *PooledExecutor) Execute(jobs []Job) ([]Result, error) {
	e.pool.Start()
	defer e.pool.Stop()

	// 提交任务
	for _, job := range jobs {
		if err := e.pool.Submit(job); err != nil {
			return nil, err
		}
	}

	// 收集结果
	results := make([]Result, 0, len(jobs))
	go func() {
		for result := range e.pool.GetResults() {
			results = append(results, result)
		}
	}()

	e.pool.Wait()
	return results, nil
}

// Semaphore 信号量
type Semaphore struct {
	capacity int
	tickets  chan struct{}
}

// NewSemaphore 创建信号量
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		capacity: capacity,
		tickets:  make(chan struct{}, capacity),
	}
}

// Acquire 获取信号量
func (s *Semaphore) Acquire() {
	s.tickets <- struct{}{}
}

// Release 释放信号量
func (s *Semaphore) Release() {
	<-s.tickets
}

// WithSemaphore 在信号量控制下执行
func (s *Semaphore) WithSemaphore(fn func()) {
	s.Acquire()
	defer s.Release()
	fn()
}

// RateLimiter 速率限制器
type RateLimiter struct {
	rate   int // 每秒请求数
	bucket chan struct{}
	ticker *time.Ticker
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(rate int) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		rate:   rate,
		bucket: make(chan struct{}, rate),
		ticker: time.NewTicker(time.Second / time.Duration(rate)),
		ctx:    ctx,
		cancel: cancel,
	}

	// 填充桶
	go rl.refill()

	return rl
}

// refill 填充令牌
func (rl *RateLimiter) refill() {
	defer rl.ticker.Stop()
	for {
		select {
		case <-rl.ticker.C:
			select {
			case rl.bucket <- struct{}{}:
			default:
				// 桶已满，丢弃令牌
			}
		case <-rl.ctx.Done():
			return
		}
	}
}

// Allow 检查是否允许
func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.bucket:
		return true
	default:
		return false
	}
}

// Wait 等待可用
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		time.Sleep(10 * time.Millisecond)
	}
}

// Stop 停止
func (rl *RateLimiter) Stop() {
	rl.cancel()
}

// ConcurrencyLimiter 并发限制器
type ConcurrencyLimiter struct {
	maxConcurrent int
	semaphore     *Semaphore
}

// NewConcurrencyLimiter 创建并发限制器
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		maxConcurrent: maxConcurrent,
		semaphore:     NewSemaphore(maxConcurrent),
	}
}

// Execute 在并发限制下执行
func (cl *ConcurrencyLimiter) Execute(fn func() error) error {
	cl.semaphore.Acquire()
	defer cl.semaphore.Release()
	return fn()
}

// ExecuteBatch 批量执行
func (cl *ConcurrencyLimiter) ExecuteBatch(fns []func() error) []error {
	results := make([]error, len(fns))
	var wg sync.WaitGroup

	for i, fn := range fns {
		wg.Add(1)
		go func(index int, f func() error) {
			defer wg.Done()
			results[index] = cl.Execute(f)
		}(i, fn)
	}

	wg.Wait()
	return results
}

// TaskQueue 任务队列
type TaskQueue struct {
	queue chan func()
	wg    sync.WaitGroup
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(bufferSize int) *TaskQueue {
	return &TaskQueue{
		queue: make(chan func(), bufferSize),
	}
}

// Start 启动队列处理
func (t *TaskQueue) Start(workers int) {
	for i := 0; i < workers; i++ {
		t.wg.Add(1)
		go t.worker()
	}
}

// worker 工作协程
func (t *TaskQueue) worker() {
	defer t.wg.Done()
	for task := range t.queue {
		task()
	}
}

// Submit 提交任务
func (t *TaskQueue) Submit(task func()) {
	t.queue <- task
}

// Stop 停止队列
func (t *TaskQueue) Stop() {
	close(t.queue)
	t.wg.Wait()
}

// ParallelParallel 并行执行器
type ParallelExecutor struct {
	maxWorkers int
}

// NewParallelExecutor 创建并行执行器
func NewParallelExecutor(maxWorkers int) *ParallelExecutor {
	return &ParallelExecutor{
		maxWorkers: maxWorkers,
	}
}

// Execute 并行执行
func (p *ParallelExecutor) Execute(fns []func() error) []error {
	if len(fns) == 0 {
		return nil
	}

	// 如果任务数少于 worker 数，调整
	workers := p.maxWorkers
	if len(fns) < workers {
		workers = len(fns)
	}

	// 创建任务队列
	queue := make(chan int, len(fns))
	results := make([]error, len(fns))
	var wg sync.WaitGroup

	// 启动 workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range queue {
				results[index] = fns[index]()
			}
		}()
	}

	// 提交任务
	for i := range fns {
		queue <- i
	}
	close(queue)

	wg.Wait()
	return results
}

// Future 异步结果
type Future struct {
	result chan Result
}

// NewFuture 创建 Future
func NewFuture() *Future {
	return &Future{
		result: make(chan Result, 1),
	}
}

// Complete 完成任务
func (f *Future) Complete(result Result) {
	f.result <- result
}

// Get 获取结果 (阻塞)
func (f *Future) Get() Result {
	return <-f.result
}

// GetWithTimeout 带超时获取结果
func (f *Future) GetWithTimeout(timeout time.Duration) (Result, error) {
	select {
	case result := <-f.result:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for result")
	}
}

// AsyncExecutor 异步执行器
type AsyncExecutor struct {
	pool *WorkerPool
}

// NewAsyncExecutor 创建异步执行器
func NewAsyncExecutor(size int) *AsyncExecutor {
	return &AsyncExecutor{
		pool: NewWorkerPool(size),
	}
}

// SubmitAsync 提交异步任务
func (a *AsyncExecutor) SubmitAsync(job Job) *Future {
	future := NewFuture()

	wrappedJob := JobFunc(func() error {
		err := job.Execute()
		future.Complete(&SimpleResult{err: err})
		return err
	})

	a.pool.Submit(wrappedJob)
	return future
}

// Start 启动
func (a *AsyncExecutor) Start() {
	a.pool.Start()
}

// Stop 停止
func (a *AsyncExecutor) Stop() {
	a.pool.Stop()
}

// ContextPool 上下文感知的 Worker 池
type ContextPool struct {
	pool   *WorkerPool
	ctx    context.Context
	cancel context.CancelFunc
}

// NewContextPool 创建上下文感知的 Worker 池
func NewContextPool(size int) *ContextPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContextPool{
		pool:   NewWorkerPool(size),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Submit 提交任务 (支持上下文)
func (p *ContextPool) Submit(job Job) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
		return p.pool.Submit(job)
	}
}

// StopWithContext 带上下文停止
func (p *ContextPool) StopWithContext(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		done <- p.pool.Stop()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Cancel 取消所有任务
func (p *ContextPool) Cancel() {
	p.cancel()
}

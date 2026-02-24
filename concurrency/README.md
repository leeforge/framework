# concurrency — 并发控制

提供 Worker Pool、信号量、速率限制器、并发限制器等并发原语，简化 Go 中的并发任务管理。

## 主要组件

### WorkerPool — Worker 池

批量并发执行任务，控制最大并发数：

```go
import "github.com/leeforge/framework/concurrency"

pool := concurrency.NewWorkerPool(10) // 10 个 worker
pool.Start()

// 提交任务
pool.Submit(concurrency.JobFunc(func() error {
    // 执行工作
    return nil
}))

// 批量提交
pool.SubmitBatch(jobs)

// 等待所有任务完成后停止
pool.Stop()
```

### Semaphore — 信号量

控制同时进入某段代码的 goroutine 数量：

```go
sem := concurrency.NewSemaphore(5) // 同时最多 5 个

sem.WithSemaphore(func() {
    // 并发安全的关键操作
    callExternalAPI()
})
```

### ConcurrencyLimiter — 并发限制器

```go
limiter := concurrency.NewConcurrencyLimiter(20)

err := limiter.Execute(func() error {
    return processItem(item)
})

// 批量并发执行（自动限制并发数）
errs := limiter.ExecuteBatch([]func() error{fn1, fn2, fn3})
```

### RateLimiter — 令牌桶速率限制器

```go
// 每秒最多 100 个请求
rl := concurrency.NewRateLimiter(100)
defer rl.Stop()

if rl.Allow() {
    // 允许执行
}
// 或阻塞等待令牌
rl.Wait()
```

### ParallelExecutor — 并行执行器

```go
executor := concurrency.NewParallelExecutor(8) // 最多 8 路并行

errs := executor.Execute([]func() error{
    func() error { return processA() },
    func() error { return processB() },
    func() error { return processC() },
})
```

### Future — 异步结果

```go
executor := concurrency.NewAsyncExecutor(4)
executor.Start()
defer executor.Stop()

future := executor.SubmitAsync(job)

// 阻塞获取结果
result := future.Get()

// 带超时获取
result, err := future.GetWithTimeout(5 * time.Second)
```

### TaskQueue — 任务队列

```go
queue := concurrency.NewTaskQueue(100) // 缓冲区 100
queue.Start(4) // 4 个消费者

queue.Submit(func() {
    sendEmail(user)
})

queue.Stop() // 等待队列排空后停止
```

## 注意事项

- `WorkerPool.Stop()` 有 30 秒超时，若任务长期阻塞请设置合理超时
- `RateLimiter.Wait()` 使用轮询，精度约 10ms，高精度场景建议使用 `time.Ticker`
- `Future` 是一次性的，Get 之后不能重复读取

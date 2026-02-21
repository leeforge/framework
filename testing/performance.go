package testing

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceTestSuite is a suite for performance testing
type PerformanceTestSuite struct {
	name       string
	benchmarks map[string]func(*PerformanceContext) *BenchmarkResult
	mu         sync.Mutex
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite(name string) *PerformanceTestSuite {
	return &PerformanceTestSuite{
		name:       name,
		benchmarks: make(map[string]func(*PerformanceContext) *BenchmarkResult),
	}
}

// Add adds a benchmark to the suite
func (ps *PerformanceTestSuite) Add(name string, fn func(*PerformanceContext) *BenchmarkResult) *PerformanceTestSuite {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.benchmarks[name] = fn
	return ps
}

// Run runs all benchmarks in the suite
func (ps *PerformanceTestSuite) Run() map[string]*BenchmarkResult {
	results := make(map[string]*BenchmarkResult)
	for name, benchmark := range ps.benchmarks {
		ctx := NewPerformanceContext()
		result := benchmark(ctx)
		results[name] = result
	}
	return results
}

// PerformanceContext provides context for performance tests
type PerformanceContext struct {
	ctx     context.Context
	start   time.Time
	metrics map[string]interface{}
	mu      sync.Mutex
}

// NewPerformanceContext creates a new performance context
func NewPerformanceContext() *PerformanceContext {
	return &PerformanceContext{
		ctx:     context.Background(),
		start:   time.Now(),
		metrics: make(map[string]interface{}),
	}
}

// Context returns the context
func (pc *PerformanceContext) Context() context.Context {
	return pc.ctx
}

// Start starts a timer
func (pc *PerformanceContext) Start() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.start = time.Now()
}

// Elapsed returns elapsed time
func (pc *PerformanceContext) Elapsed() time.Duration {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return time.Since(pc.start)
}

// SetMetric sets a metric
func (pc *PerformanceContext) SetMetric(key string, value interface{}) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.metrics[key] = value
}

// GetMetric gets a metric
func (pc *PerformanceContext) GetMetric(key string) interface{} {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.metrics[key]
}

// BenchmarkResult represents a benchmark result
type BenchmarkResult struct {
	Name        string
	Duration    time.Duration
	Operations  int64
	Bytes       int64
	Allocations int64
	Memory      uint64
	Extra       map[string]interface{}
}

// String returns a string representation
func (br *BenchmarkResult) String() string {
	return fmt.Sprintf(
		"%s: Duration=%v, Ops=%d, Bytes=%d, Allocs=%d, Memory=%d",
		br.Name, br.Duration, br.Operations, br.Bytes, br.Allocations, br.Memory,
	)
}

// Throughput returns operations per second
func (br *BenchmarkResult) Throughput() float64 {
	if br.Duration == 0 {
		return 0
	}
	return float64(br.Operations) / br.Duration.Seconds()
}

// Latency returns average latency per operation
func (br *BenchmarkResult) Latency() time.Duration {
	if br.Operations == 0 {
		return 0
	}
	return time.Duration(br.Duration.Nanoseconds() / br.Operations)
}

// MemoryPerOp returns memory allocated per operation
func (br *BenchmarkResult) MemoryPerOp() float64 {
	if br.Operations == 0 {
		return 0
	}
	return float64(br.Memory) / float64(br.Operations)
}

// LoadTestConfig represents load test configuration
type LoadTestConfig struct {
	Concurrency   int
	TotalRequests int64
	Duration      time.Duration
	RampUp        time.Duration
	TargetRPS     float64
}

// DefaultLoadTestConfig creates a default load test configuration
func DefaultLoadTestConfig() LoadTestConfig {
	return LoadTestConfig{
		Concurrency:   10,
		TotalRequests: 1000,
		Duration:      30 * time.Second,
		RampUp:        5 * time.Second,
		TargetRPS:     0, // unlimited
	}
}

// LoadTestResult represents a load test result
type LoadTestResult struct {
	Config         LoadTestConfig
	TotalRequests  int64
	Successful     int64
	Failed         int64
	TotalDuration  time.Duration
	AverageLatency time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	RequestsPerSec float64
	StatusCodes    map[int]int64
	Errors         []string
}

// String returns a string representation
func (lr *LoadTestResult) String() string {
	return fmt.Sprintf(
		"Load Test Results:\n"+
			"  Total Requests: %d\n"+
			"  Successful: %d\n"+
			"  Failed: %d\n"+
			"  Duration: %v\n"+
			"  RPS: %.2f\n"+
			"  Avg Latency: %v\n"+
			"  Min Latency: %v\n"+
			"  Max Latency: %v\n"+
			"  Status Codes: %v",
		lr.TotalRequests, lr.Successful, lr.Failed, lr.TotalDuration,
		lr.RequestsPerSec, lr.AverageLatency, lr.MinLatency, lr.MaxLatency,
		lr.StatusCodes,
	)
}

// LoadTest runs a load test
type LoadTest struct {
	config   LoadTestConfig
	workload func() error
}

// NewLoadTest creates a new load test
func NewLoadTest(config LoadTestConfig, workload func() error) *LoadTest {
	return &LoadTest{
		config:   config,
		workload: workload,
	}
}

// Run executes the load test
func (lt *LoadTest) Run() *LoadTestResult {
	start := time.Now()
	var wg sync.WaitGroup
	var successful, failed int64
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration = time.Hour, 0
	statusCodes := make(map[int]int64)
	var errorsMu sync.Mutex
	errors := make([]string, 0)

	// Rate limiter
	var rateLimiter *time.Ticker
	if lt.config.TargetRPS > 0 {
		interval := time.Duration(float64(time.Second) / lt.config.TargetRPS)
		rateLimiter = time.NewTicker(interval)
		defer rateLimiter.Stop()
	}

	// Semaphore for concurrency control
	sem := make(chan struct{}, lt.config.Concurrency)

	// Ramp up
	if lt.config.RampUp > 0 {
		rampUpInterval := lt.config.RampUp / time.Duration(lt.config.Concurrency)
		for i := 0; i < lt.config.Concurrency; i++ {
			go func() {
				time.Sleep(rampUpInterval * time.Duration(i))
				sem <- struct{}{}
			}()
		}
	} else {
		// Fill semaphore immediately
		for i := 0; i < lt.config.Concurrency; i++ {
			sem <- struct{}{}
		}
	}

	// Request counter
	var requestCount int64
	done := make(chan struct{})

	// Time-based limit
	if lt.config.Duration > 0 {
		go func() {
			time.Sleep(lt.config.Duration)
			close(done)
		}()
	}

	// Request-based limit
	requestLimit := lt.config.TotalRequests

	// Worker function
	worker := func() {
		defer wg.Done()

		for {
			// Check limits
			if requestLimit > 0 && atomic.LoadInt64(&requestCount) >= requestLimit {
				return
			}

			select {
			case <-done:
				return
			default:
			}

			// Rate limiting
			if rateLimiter != nil {
				<-rateLimiter.C
			}

			// Acquire semaphore
			select {
			case <-sem:
			case <-done:
				return
			}

			// Increment request count
			reqNum := atomic.AddInt64(&requestCount, 1)
			if requestLimit > 0 && reqNum > requestLimit {
				sem <- struct{}{}
				return
			}

			// Execute workload
			reqStart := time.Now()
			err := lt.workload()
			latency := time.Since(reqStart)

			// Update stats
			atomic.AddInt64(&successful, 1)
			totalLatency += latency

			// Update min/max latency
			for {
				if latency < minLatency {
					if atomic.CompareAndSwapInt64((*int64)(&minLatency), int64(minLatency), int64(latency)) {
						break
					}
				} else {
					break
				}
			}
			for {
				if latency > maxLatency {
					if atomic.CompareAndSwapInt64((*int64)(&maxLatency), int64(maxLatency), int64(latency)) {
						break
					}
				} else {
					break
				}
			}

			// Handle error
			if err != nil {
				atomic.AddInt64(&failed, 1)
				errorsMu.Lock()
				if len(errors) < 100 { // Limit error storage
					errors = append(errors, err.Error())
				}
				errorsMu.Unlock()
				// Simulate status code 500 for errors
				statusCodes[500]++
			} else {
				// Simulate status code 200 for success
				statusCodes[200]++
			}

			// Release semaphore
			sem <- struct{}{}
		}
	}

	// Start workers
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go worker()
	}

	// Wait for completion
	wg.Wait()
	close(done)

	duration := time.Since(start)

	// Calculate results
	totalReqs := atomic.LoadInt64(&successful) + atomic.LoadInt64(&failed)
	var avgLatency time.Duration
	if totalReqs > 0 {
		avgLatency = time.Duration(totalLatency.Nanoseconds() / totalReqs)
	}

	return &LoadTestResult{
		Config:         lt.config,
		TotalRequests:  totalReqs,
		Successful:     atomic.LoadInt64(&successful),
		Failed:         atomic.LoadInt64(&failed),
		TotalDuration:  duration,
		AverageLatency: avgLatency,
		MinLatency:     minLatency,
		MaxLatency:     maxLatency,
		RequestsPerSec: float64(totalReqs) / duration.Seconds(),
		StatusCodes:    statusCodes,
		Errors:         errors,
	}
}

// StressTestConfig represents stress test configuration
type StressTestConfig struct {
	Stages []Stage
}

// Stage represents a stress test stage
type Stage struct {
	Name        string
	Concurrency int
	Duration    time.Duration
	TargetRPS   float64
}

// StressTestResult represents a stress test result
type StressTestResult struct {
	Stages []StageResult
}

// StageResult represents a stage result
type StageResult struct {
	Stage
	LoadTestResult *LoadTestResult
}

// StressTest runs a stress test
type StressTest struct {
	config   StressTestConfig
	workload func() error
}

// NewStressTest creates a new stress test
func NewStressTest(config StressTestConfig, workload func() error) *StressTest {
	return &StressTest{
		config:   config,
		workload: workload,
	}
}

// Run executes the stress test
func (st *StressTest) Run() *StressTestResult {
	result := &StressTestResult{
		Stages: make([]StageResult, 0, len(st.config.Stages)),
	}

	for _, stage := range st.config.Stages {
		ltConfig := LoadTestConfig{
			Concurrency:   stage.Concurrency,
			TotalRequests: 0, // Use duration
			Duration:      stage.Duration,
			RampUp:        0,
			TargetRPS:     stage.TargetRPS,
		}

		lt := NewLoadTest(ltConfig, st.workload)
		ltResult := lt.Run()

		result.Stages = append(result.Stages, StageResult{
			Stage:          stage,
			LoadTestResult: ltResult,
		})
	}

	return result
}

// PerformanceMonitor monitors performance metrics
type PerformanceMonitor struct {
	metrics map[string]*MetricData
	mu      sync.RWMutex
}

// MetricData holds metric data
type MetricData struct {
	Values []float64
	Sum    float64
	Count  int64
	Min    float64
	Max    float64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*MetricData),
	}
}

// Record records a metric value
func (pm *PerformanceMonitor) Record(name string, value float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	data, exists := pm.metrics[name]
	if !exists {
		data = &MetricData{
			Values: make([]float64, 0),
			Min:    value,
			Max:    value,
		}
		pm.metrics[name] = data
	}

	data.Values = append(data.Values, value)
	data.Sum += value
	data.Count++

	if value < data.Min {
		data.Min = value
	}
	if value > data.Max {
		data.Max = value
	}
}

// GetStats gets statistics for a metric
func (pm *PerformanceMonitor) GetStats(name string) *MetricStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	data, exists := pm.metrics[name]
	if !exists {
		return nil
	}

	return &MetricStats{
		Name:    name,
		Count:   data.Count,
		Sum:     data.Sum,
		Average: data.Sum / float64(data.Count),
		Min:     data.Min,
		Max:     data.Max,
	}
}

// GetAllStats gets all statistics
func (pm *PerformanceMonitor) GetAllStats() map[string]*MetricStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*MetricStats)
	for name := range pm.metrics {
		result[name] = pm.GetStats(name)
	}
	return result
}

// Clear clears all metrics
func (pm *PerformanceMonitor) Clear() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.metrics = make(map[string]*MetricData)
}

// MetricStats represents statistics for a metric
type MetricStats struct {
	Name    string
	Count   int64
	Sum     float64
	Average float64
	Min     float64
	Max     float64
}

// String returns a string representation
func (ms *MetricStats) String() string {
	return fmt.Sprintf(
		"%s: Count=%d, Sum=%.2f, Avg=%.2f, Min=%.2f, Max=%.2f",
		ms.Name, ms.Count, ms.Sum, ms.Average, ms.Min, ms.Max,
	)
}

// MemoryTracker tracks memory usage
type MemoryTracker struct {
	before runtime.MemStats
	after  runtime.MemStats
}

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
}

// Start starts tracking
func (mt *MemoryTracker) Start() {
	runtime.GC()
	runtime.ReadMemStats(&mt.before)
}

// Stop stops tracking and returns diff
func (mt *MemoryTracker) Stop() *MemoryDiff {
	runtime.GC()
	runtime.ReadMemStats(&mt.after)

	return &MemoryDiff{
		Alloc:      mt.after.Alloc - mt.before.Alloc,
		TotalAlloc: mt.after.TotalAlloc - mt.before.TotalAlloc,
		Mallocs:    mt.after.Mallocs - mt.before.Mallocs,
		Frees:      mt.after.Frees - mt.before.Frees,
		HeapAlloc:  mt.after.HeapAlloc - mt.before.HeapAlloc,
		HeapInuse:  mt.after.HeapInuse - mt.before.HeapInuse,
		StackInuse: mt.after.StackInuse - mt.before.StackInuse,
		NumGC:      mt.after.NumGC - mt.before.NumGC,
	}
}

// MemoryDiff represents memory differences
type MemoryDiff struct {
	Alloc      uint64
	TotalAlloc uint64
	Mallocs    uint64
	Frees      uint64
	HeapAlloc  uint64
	HeapInuse  uint64
	StackInuse uint64
	NumGC      uint32
}

// String returns a string representation
func (md *MemoryDiff) String() string {
	return fmt.Sprintf(
		"Alloc: %d, TotalAlloc: %d, Mallocs: %d, Frees: %d, HeapAlloc: %d, HeapInuse: %d, StackInuse: %d, NumGC: %d",
		md.Alloc, md.TotalAlloc, md.Mallocs, md.Frees, md.HeapAlloc, md.HeapInuse, md.StackInuse, md.NumGC,
	)
}

// PerformanceBenchmarkRunner runs benchmarks
type PerformanceBenchmarkRunner struct {
	benchmarks map[string]func(*BenchmarkContext) *BenchmarkResult
	mu         sync.Mutex
}

// NewPerformanceBenchmarkRunner creates a new benchmark runner
func NewPerformanceBenchmarkRunner() *PerformanceBenchmarkRunner {
	return &PerformanceBenchmarkRunner{
		benchmarks: make(map[string]func(*BenchmarkContext) *BenchmarkResult),
	}
}

// Add adds a benchmark
func (br *PerformanceBenchmarkRunner) Add(name string, fn func(*BenchmarkContext) *BenchmarkResult) *PerformanceBenchmarkRunner {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.benchmarks[name] = fn
	return br
}

// Run runs all benchmarks
func (br *PerformanceBenchmarkRunner) Run() map[string]*BenchmarkResult {
	results := make(map[string]*BenchmarkResult)
	for name, benchmark := range br.benchmarks {
		ctx := NewBenchmarkContext()
		result := benchmark(ctx)
		results[name] = result
	}
	return results
}

// BenchmarkContext provides context for benchmarks
type BenchmarkContext struct {
	iterations int64
	mu         sync.Mutex
}

// NewBenchmarkContext creates a new benchmark context
func NewBenchmarkContext() *BenchmarkContext {
	return &BenchmarkContext{}
}

// Increment increments iteration count
func (bc *BenchmarkContext) Increment() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.iterations++
}

// Iterations returns the iteration count
func (bc *BenchmarkContext) Iterations() int64 {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.iterations
}

// BenchmarkFunc is a function that can be benchmarked
type BenchmarkFunc func() error

// Measure measures a function
func Measure(fn BenchmarkFunc, iterations int) *BenchmarkResult {
	start := time.Now()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	var successes, failures int64
	var totalBytes int64

	for i := 0; i < iterations; i++ {
		err := fn()
		if err == nil {
			atomic.AddInt64(&successes, 1)
		} else {
			atomic.AddInt64(&failures, 1)
		}
	}

	duration := time.Since(start)

	var afterMemStats runtime.MemStats
	runtime.ReadMemStats(&afterMemStats)

	return &BenchmarkResult{
		Name:        "Measured",
		Duration:    duration,
		Operations:  successes,
		Bytes:       totalBytes,
		Allocations: int64(afterMemStats.Mallocs - memStats.Mallocs),
		Memory:      afterMemStats.Alloc - memStats.Alloc,
		Extra: map[string]interface{}{
			"failures": failures,
		},
	}
}

// ThroughputTest measures throughput
type ThroughputTest struct {
	duration time.Duration
	workload func()
}

// NewThroughputTest creates a new throughput test
func NewThroughputTest(duration time.Duration, workload func()) *ThroughputTest {
	return &ThroughputTest{
		duration: duration,
		workload: workload,
	}
}

// Run executes the throughput test
func (tt *ThroughputTest) Run() *BenchmarkResult {
	var ops int64
	done := make(chan struct{})
	ticker := time.NewTicker(tt.duration)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				tt.workload()
				atomic.AddInt64(&ops, 1)
			}
		}
	}()

	<-ticker.C
	close(done)

	return &BenchmarkResult{
		Name:       "Throughput",
		Duration:   tt.duration,
		Operations: ops,
	}
}

// LatencyTest measures latency
type LatencyTest struct {
	iterations int
	workload   func() error
}

// NewLatencyTest creates a new latency test
func NewLatencyTest(iterations int, workload func() error) *LatencyTest {
	return &LatencyTest{
		iterations: iterations,
		workload:   workload,
	}
}

// Run executes the latency test
func (lt *LatencyTest) Run() *BenchmarkResult {
	var latencies []time.Duration
	var totalLatency time.Duration
	var minLatency time.Duration = time.Hour
	var maxLatency time.Duration = 0

	for i := 0; i < lt.iterations; i++ {
		start := time.Now()
		err := lt.workload()
		if err != nil {
			continue
		}
		latency := time.Since(start)
		latencies = append(latencies, latency)
		totalLatency += latency

		if latency < minLatency {
			minLatency = latency
		}
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	avgLatency := time.Duration(0)
	if len(latencies) > 0 {
		avgLatency = time.Duration(totalLatency.Nanoseconds() / int64(len(latencies)))
	}

	return &BenchmarkResult{
		Name:       "Latency",
		Duration:   totalLatency,
		Operations: int64(len(latencies)),
		Extra: map[string]interface{}{
			"avg_latency": avgLatency,
			"min_latency": minLatency,
			"max_latency": maxLatency,
		},
	}
}

// PerformanceReport generates performance reports
type PerformanceReport struct {
	results map[string]*BenchmarkResult
}

// NewPerformanceReport creates a new performance report
func NewPerformanceReport(results map[string]*BenchmarkResult) *PerformanceReport {
	return &PerformanceReport{
		results: results,
	}
}

// Generate generates a report string
func (pr *PerformanceReport) Generate() string {
	var report string
	report += "Performance Report\n"
	report += "=================\n\n"

	for name, result := range pr.results {
		report += fmt.Sprintf("%s:\n", name)
		report += fmt.Sprintf("  Duration: %v\n", result.Duration)
		report += fmt.Sprintf("  Operations: %d\n", result.Operations)
		report += fmt.Sprintf("  Throughput: %.2f ops/sec\n", result.Throughput())
		report += fmt.Sprintf("  Latency: %v\n", result.Latency())
		report += fmt.Sprintf("  Memory: %d bytes\n", result.Memory)
		report += fmt.Sprintf("  Allocations: %d\n", result.Allocations)
		if result.Extra != nil {
			report += fmt.Sprintf("  Extra: %v\n", result.Extra)
		}
		report += "\n"
	}

	return report
}

// Compare compares two sets of results
func (pr *PerformanceReport) Compare(other *PerformanceReport) string {
	var report string
	report += "Performance Comparison\n"
	report += "=====================\n\n"

	for name, result := range pr.results {
		if otherResult, exists := other.results[name]; exists {
			report += fmt.Sprintf("%s:\n", name)
			report += fmt.Sprintf("  Current:  %.2f ops/sec\n", result.Throughput())
			report += fmt.Sprintf("  Baseline: %.2f ops/sec\n", otherResult.Throughput())

			throughputDiff := result.Throughput() - otherResult.Throughput()
			throughputPct := (throughputDiff / otherResult.Throughput()) * 100
			report += fmt.Sprintf("  Diff: %.2f ops/sec (%.2f%%)\n", throughputDiff, throughputPct)

			latencyDiff := result.Latency() - otherResult.Latency()
			latencyPct := (float64(latencyDiff) / float64(otherResult.Latency())) * 100
			report += fmt.Sprintf("  Latency Diff: %v (%.2f%%)\n", latencyDiff, latencyPct)
			report += "\n"
		}
	}

	return report
}

// PerformanceSuite is a collection of performance tests
type PerformanceSuite struct {
	name  string
	tests map[string]func() *BenchmarkResult
	mu    sync.Mutex
}

// NewPerformanceSuite creates a new performance suite
func NewPerformanceSuite(name string) *PerformanceSuite {
	return &PerformanceSuite{
		name:  name,
		tests: make(map[string]func() *BenchmarkResult),
	}
}

// Add adds a test to the suite
func (ps *PerformanceSuite) Add(name string, fn func() *BenchmarkResult) *PerformanceSuite {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.tests[name] = fn
	return ps
}

// Run runs all tests in the suite
func (ps *PerformanceSuite) Run() map[string]*BenchmarkResult {
	results := make(map[string]*BenchmarkResult)
	for name, test := range ps.tests {
		results[name] = test()
	}
	return results
}

// PerformanceAnalyzer analyzes performance results
type PerformanceAnalyzer struct {
	results map[string]*BenchmarkResult
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer(results map[string]*BenchmarkResult) *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		results: results,
	}
}

// Analyze analyzes the results
func (pa *PerformanceAnalyzer) Analyze() *AnalysisReport {
	report := &AnalysisReport{
		Analyses: make(map[string]*MetricAnalysis),
	}

	for name, result := range pa.results {
		analysis := &MetricAnalysis{
			Name:        name,
			Throughput:  result.Throughput(),
			Latency:     result.Latency(),
			Operations:  result.Operations,
			Memory:      result.Memory,
			Allocations: result.Allocations,
		}

		// Determine performance level
		tp := analysis.Throughput
		if tp > 10000 {
			analysis.Level = "Excellent"
		} else if tp > 1000 {
			analysis.Level = "Good"
		} else if tp > 100 {
			analysis.Level = "Acceptable"
		} else {
			analysis.Level = "Poor"
		}

		report.Analyses[name] = analysis
	}

	return report
}

// AnalysisReport represents an analysis report
type AnalysisReport struct {
	Analyses map[string]*MetricAnalysis
}

// MetricAnalysis represents analysis for a metric
type MetricAnalysis struct {
	Name        string
	Throughput  float64
	Latency     time.Duration
	Operations  int64
	Memory      uint64
	Allocations int64
	Level       string
}

// String returns a string representation
func (ma *MetricAnalysis) String() string {
	return fmt.Sprintf(
		"%s: %s (TP: %.2f, Latency: %v, Ops: %d)",
		ma.Name, ma.Level, ma.Throughput, ma.Latency, ma.Operations,
	)
}

// PerformanceOptimization represents optimization suggestions
type PerformanceOptimization struct {
	Metric     string
	Current    string
	Suggestion string
	Priority   string
}

// GetOptimizations gets optimization suggestions
func (pa *PerformanceAnalyzer) GetOptimizations() []PerformanceOptimization {
	var optimizations []PerformanceOptimization

	for name, result := range pa.results {
		tp := result.Throughput()
		latency := result.Latency()

		if tp < 100 {
			optimizations = append(optimizations, PerformanceOptimization{
				Metric:     name,
				Current:    fmt.Sprintf("Throughput: %.2f ops/sec", tp),
				Suggestion: "Consider caching, optimizing database queries, or reducing I/O operations",
				Priority:   "High",
			})
		}

		if latency > time.Millisecond*100 {
			optimizations = append(optimizations, PerformanceOptimization{
				Metric:     name,
				Current:    fmt.Sprintf("Latency: %v", latency),
				Suggestion: "Consider async processing, reducing lock contention, or optimizing algorithms",
				Priority:   "High",
			})
		}

		if result.Allocations > 1000000 {
			optimizations = append(optimizations, PerformanceOptimization{
				Metric:     name,
				Current:    fmt.Sprintf("Allocations: %d", result.Allocations),
				Suggestion: "Consider object pooling, reducing allocations in hot paths, or using sync.Pool",
				Priority:   "Medium",
			})
		}
	}

	return optimizations
}

// BenchmarkConfig represents benchmark configuration
type BenchmarkConfig struct {
	Iterations int
	Duration   time.Duration
	Warmup     int
	CPUProfile bool
	MemProfile bool
}

// DefaultBenchmarkConfig creates a default benchmark configuration
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		Iterations: 1000,
		Duration:   5 * time.Second,
		Warmup:     100,
		CPUProfile: false,
		MemProfile: false,
	}
}

// BenchmarkExecutor executes benchmarks with configuration
type BenchmarkExecutor struct {
	config BenchmarkConfig
}

// NewBenchmarkExecutor creates a new benchmark executor
func NewBenchmarkExecutor(config BenchmarkConfig) *BenchmarkExecutor {
	return &BenchmarkExecutor{
		config: config,
	}
}

// Execute executes a benchmark
func (be *BenchmarkExecutor) Execute(name string, fn func() error) *BenchmarkResult {
	// Warmup
	for i := 0; i < be.config.Warmup; i++ {
		fn()
	}

	// Reset memory stats
	runtime.GC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	start := time.Now()
	var successes int64
	var done chan struct{}

	if be.config.Duration > 0 {
		done = make(chan struct{})
		go func() {
			time.Sleep(be.config.Duration)
			close(done)
		}()
	}

	// Execute
	for i := 0; i < be.config.Iterations || be.config.Iterations == 0; i++ {
		if be.config.Duration > 0 {
			select {
			case <-done:
				goto done
			default:
			}
		}

		if err := fn(); err == nil {
			atomic.AddInt64(&successes, 1)
		}
	}

done:
	duration := time.Since(start)

	var afterMemStats runtime.MemStats
	runtime.ReadMemStats(&afterMemStats)

	return &BenchmarkResult{
		Name:        name,
		Duration:    duration,
		Operations:  successes,
		Allocations: int64(afterMemStats.Mallocs - memStats.Mallocs),
		Memory:      afterMemStats.Alloc - memStats.Alloc,
	}
}

// PerformanceTestHelper provides helper methods for performance testing
type PerformanceTestHelper struct{}

// NewPerformanceTestHelper creates a new performance test helper
func NewPerformanceTestHelper() *PerformanceTestHelper {
	return &PerformanceTestHelper{}
}

// MeasureMemory measures memory usage of a function
func (h *PerformanceTestHelper) MeasureMemory(fn func()) *MemoryDiff {
	tracker := NewMemoryTracker()
	tracker.Start()
	fn()
	return tracker.Stop()
}

// MeasureDuration measures duration of a function
func (h *PerformanceTestHelper) MeasureDuration(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// MeasureThroughput measures throughput of a function
func (h *PerformanceTestHelper) MeasureThroughput(fn func(), duration time.Duration) float64 {
	var ops int64
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(duration)
		defer ticker.Stop()
		<-ticker.C
		close(done)
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				fn()
				atomic.AddInt64(&ops, 1)
			}
		}
	}()

	<-done
	return float64(ops) / duration.Seconds()
}

// CompareResults compares two benchmark results
func (h *PerformanceTestHelper) CompareResults(a, b *BenchmarkResult) string {
	if a == nil || b == nil {
		return "Cannot compare nil results"
	}

	var report string
	report += fmt.Sprintf("Comparing %s:\n", a.Name)

	tpA := a.Throughput()
	tpB := b.Throughput()
	tpDiff := tpA - tpB
	tpPct := (tpDiff / tpB) * 100
	report += fmt.Sprintf("  Throughput: %.2f vs %.2f (%.2f%%)\n", tpA, tpB, tpPct)

	latA := a.Latency()
	latB := b.Latency()
	latDiff := latA - latB
	latPct := (float64(latDiff) / float64(latB)) * 100
	report += fmt.Sprintf("  Latency: %v vs %v (%.2f%%)\n", latA, latB, latPct)

	memA := a.Memory
	memB := b.Memory
	memDiff := int64(memA - memB)
	memPct := (float64(memDiff) / float64(memB)) * 100
	report += fmt.Sprintf("  Memory: %d vs %d (%.2f%%)\n", memA, memB, memPct)

	return report
}

// IsRegression checks if there's a performance regression
func (h *PerformanceTestHelper) IsRegression(current, baseline *BenchmarkResult, threshold float64) bool {
	if current == nil || baseline == nil {
		return false
	}

	tpDiff := (current.Throughput() - baseline.Throughput()) / baseline.Throughput()
	latDiff := (float64(current.Latency()) - float64(baseline.Latency())) / float64(baseline.Latency())

	// Check if throughput decreased by more than threshold
	// or latency increased by more than threshold
	return tpDiff < -threshold || latDiff > threshold
}

// PerformanceReportFormat formats performance reports
type PerformanceReportFormat struct{}

// NewPerformanceReportFormat creates a new performance report formatter
func NewPerformanceReportFormat() *PerformanceReportFormat {
	return &PerformanceReportFormat{}
}

// ToJSON converts results to JSON
func (f *PerformanceReportFormat) ToJSON(results map[string]*BenchmarkResult) string {
	type jsonResult struct {
		Name        string  `json:"name"`
		Duration    float64 `json:"duration_ms"`
		Operations  int64   `json:"operations"`
		Throughput  float64 `json:"throughput"`
		Latency     float64 `json:"latency_ms"`
		Memory      uint64  `json:"memory_bytes"`
		Allocations int64   `json:"allocations"`
	}

	var jsonResults []jsonResult
	for name, result := range results {
		jsonResults = append(jsonResults, jsonResult{
			Name:        name,
			Duration:    result.Duration.Seconds() * 1000,
			Operations:  result.Operations,
			Throughput:  result.Throughput(),
			Latency:     float64(result.Latency().Nanoseconds()) / 1e6,
			Memory:      result.Memory,
			Allocations: result.Allocations,
		})
	}

	// Simple JSON formatting
	var sb strings.Builder
	sb.WriteString("[")
	for i, r := range jsonResults {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(
			`{"name":"%s","duration_ms":%.2f,"operations":%d,"throughput":%.2f,"latency_ms":%.2f,"memory_bytes":%d,"allocations":%d}`,
			r.Name, r.Duration, r.Operations, r.Throughput, r.Latency, r.Memory, r.Allocations,
		))
	}
	sb.WriteString("]")
	return sb.String()
}

// ToCSV converts results to CSV
func (f *PerformanceReportFormat) ToCSV(results map[string]*BenchmarkResult) string {
	var sb strings.Builder
	sb.WriteString("Name,Duration(ms),Operations,Throughput(ops/s),Latency(ms),Memory(bytes),Allocations\n")

	for name, result := range results {
		sb.WriteString(fmt.Sprintf(
			"%s,%.2f,%d,%.2f,%.2f,%d,%d\n",
			name,
			result.Duration.Seconds()*1000,
			result.Operations,
			result.Throughput(),
			float64(result.Latency().Nanoseconds())/1e6,
			result.Memory,
			result.Allocations,
		))
	}

	return sb.String()
}

// ToMarkdown converts results to Markdown
func (f *PerformanceReportFormat) ToMarkdown(results map[string]*BenchmarkResult) string {
	var sb strings.Builder
	sb.WriteString("# Performance Results\n\n")
	sb.WriteString("| Benchmark | Duration | Operations | Throughput | Latency | Memory | Allocations |\n")
	sb.WriteString("|-----------|----------|------------|------------|---------|--------|-------------|\n")

	for name, result := range results {
		sb.WriteString(fmt.Sprintf(
			"| %s | %.2fms | %d | %.2f ops/s | %.2fms | %d bytes | %d |\n",
			name,
			result.Duration.Seconds()*1000,
			result.Operations,
			result.Throughput(),
			float64(result.Latency().Nanoseconds())/1e6,
			result.Memory,
			result.Allocations,
		))
	}

	return sb.String()
}

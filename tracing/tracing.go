package tracing

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Span represents a single operation within a trace
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]interface{}
	Events     []SpanEvent
	Status     SpanStatus
	Kind       SpanKind
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Time       time.Time
	Name       string
	Attributes map[string]interface{}
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    SpanStatusCode
	Message string
}

// SpanStatusCode represents the status code of a span
type SpanStatusCode int

const (
	StatusCodeUnset SpanStatusCode = 0
	StatusCodeOK    SpanStatusCode = 1
	StatusCodeError SpanStatusCode = 2
)

// SpanKind represents the kind of a span
type SpanKind int

const (
	SpanKindInternal SpanKind = 0
	SpanKindServer   SpanKind = 1
	SpanKindClient   SpanKind = 2
	SpanKindProducer SpanKind = 3
	SpanKindConsumer SpanKind = 4
)

// Tracer represents a distributed tracing tracer
type Tracer struct {
	name      string
	version   string
	processor SpanProcessor
	sampler   Sampler
	mu        sync.RWMutex
}

// TracerConfig represents the configuration for a tracer
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	SamplingRate   float64
	Processor      SpanProcessor
}

// DefaultTracerConfig creates a default tracer configuration
func DefaultTracerConfig(serviceName string) TracerConfig {
	return TracerConfig{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		SamplingRate:   1.0,
		Processor:      NewSimpleSpanProcessor(),
	}
}

// NewTracer creates a new tracer
func NewTracer(config TracerConfig) (*Tracer, error) {
	if config.Processor == nil {
		config.Processor = NewSimpleSpanProcessor()
	}

	sampler := NewTraceIDRatioBased(config.SamplingRate)

	return &Tracer{
		name:      config.ServiceName,
		version:   config.ServiceVersion,
		processor: config.Processor,
		sampler:   sampler,
	}, nil
}

// Start starts a new span
func (t *Tracer) Start(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, *Span) {
	span := &Span{
		TraceID:    generateTraceID(),
		SpanID:     generateSpanID(),
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Events:     make([]SpanEvent, 0),
		Status:     SpanStatus{Code: StatusCodeUnset},
		Kind:       SpanKindInternal,
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	// Check if should sample
	if !t.sampler.ShouldSample(span.TraceID) {
		span.Status.Code = StatusCodeUnset
	}

	// Store span in context
	ctx = context.WithValue(ctx, spanKey{}, span)

	return ctx, span
}

// End ends a span
func (t *Tracer) End(span *Span, err error) {
	if span == nil {
		return
	}

	span.EndTime = time.Now()

	if err != nil {
		span.Status.Code = StatusCodeError
		span.Status.Message = err.Error()
		span.Attributes["error"] = true
		span.Attributes["error.message"] = err.Error()
	} else {
		span.Status.Code = StatusCodeOK
	}

	// Process the span
	t.processor.OnEnd(span)
}

// AddEvent adds an event to a span
func (t *Tracer) AddEvent(span *Span, name string, attrs map[string]interface{}) {
	if span == nil {
		return
	}

	event := SpanEvent{
		Time:       time.Now(),
		Name:       name,
		Attributes: attrs,
	}
	span.Events = append(span.Events, event)
}

// SetAttributes sets attributes on a span
func (t *Tracer) SetAttributes(span *Span, attrs map[string]interface{}) {
	if span == nil {
		return
	}

	for k, v := range attrs {
		span.Attributes[k] = v
	}
}

// SetStatus sets the status of a span
func (t *Tracer) SetStatus(span *Span, code SpanStatusCode, message string) {
	if span == nil {
		return
	}

	span.Status.Code = code
	span.Status.Message = message
}

// Shutdown shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.processor != nil {
		return t.processor.Shutdown(ctx)
	}
	return nil
}

// SpanStartOption represents a span start option
type SpanStartOption func(*Span)

// WithSpanKind sets the span kind
func WithSpanKind(kind SpanKind) SpanStartOption {
	return func(s *Span) {
		s.Kind = kind
	}
}

// WithAttributes sets attributes
func WithAttributes(attrs map[string]interface{}) SpanStartOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// WithParentID sets the parent span ID
func WithParentID(parentID string) SpanStartOption {
	return func(s *Span) {
		s.ParentID = parentID
	}
}

// SpanProcessor processes spans
type SpanProcessor interface {
	OnEnd(span *Span)
	Shutdown(ctx context.Context) error
}

// SimpleSpanProcessor is a simple span processor
type SimpleSpanProcessor struct {
	exporter SpanExporter
}

// NewSimpleSpanProcessor creates a new simple span processor
func NewSimpleSpanProcessor() *SimpleSpanProcessor {
	return &SimpleSpanProcessor{
		exporter: NewConsoleExporter(),
	}
}

// OnEnd processes a span when it ends
func (s *SimpleSpanProcessor) OnEnd(span *Span) {
	if s.exporter != nil {
		s.exporter.Export(span)
	}
}

// Shutdown shuts down the processor
func (s *SimpleSpanProcessor) Shutdown(ctx context.Context) error {
	if s.exporter != nil {
		return s.exporter.Shutdown(ctx)
	}
	return nil
}

// BatchSpanProcessor processes spans in batches
type BatchSpanProcessor struct {
	exporter  SpanExporter
	batchSize int
	batch     []*Span
	mu        sync.Mutex
	timer     *time.Timer
	timeout   time.Duration
}

// NewBatchSpanProcessor creates a new batch span processor
func NewBatchSpanProcessor(exporter SpanExporter, batchSize int, timeout time.Duration) *BatchSpanProcessor {
	processor := &BatchSpanProcessor{
		exporter:  exporter,
		batchSize: batchSize,
		batch:     make([]*Span, 0, batchSize),
		timeout:   timeout,
	}

	// Start a timer to flush periodically
	if timeout > 0 {
		processor.timer = time.AfterFunc(timeout, func() {
			processor.Flush()
		})
	}

	return processor
}

// OnEnd processes a span when it ends
func (b *BatchSpanProcessor) OnEnd(span *Span) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.batch = append(b.batch, span)

	if len(b.batch) >= b.batchSize {
		b.flushLocked()
	}
}

// Flush flushes the current batch
func (b *BatchSpanProcessor) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked()
}

func (b *BatchSpanProcessor) flushLocked() {
	if len(b.batch) == 0 {
		return
	}

	// Export all spans in the batch
	for _, span := range b.batch {
		b.exporter.Export(span)
	}

	// Clear the batch
	b.batch = b.batch[:0]

	// Reset the timer
	if b.timer != nil {
		b.timer.Reset(b.timeout)
	}
}

// Shutdown shuts down the processor
func (b *BatchSpanProcessor) Shutdown(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.flushLocked()

	if b.timer != nil {
		b.timer.Stop()
	}

	if b.exporter != nil {
		return b.exporter.Shutdown(ctx)
	}
	return nil
}

// SpanExporter exports spans
type SpanExporter interface {
	Export(span *Span) error
	Shutdown(ctx context.Context) error
}

// ConsoleExporter exports spans to the console
type ConsoleExporter struct{}

// NewConsoleExporter creates a new console exporter
func NewConsoleExporter() *ConsoleExporter {
	return &ConsoleExporter{}
}

// Export exports a span to the console
func (c *ConsoleExporter) Export(span *Span) error {
	fmt.Printf("[TRACE] %s | TraceID: %s | SpanID: %s | Duration: %v | Status: %d\n",
		span.Name, span.TraceID, span.SpanID, span.EndTime.Sub(span.StartTime), span.Status.Code)
	return nil
}

// Shutdown shuts down the exporter
func (c *ConsoleExporter) Shutdown(ctx context.Context) error {
	return nil
}

// Sampler determines which spans to sample
type Sampler interface {
	ShouldSample(traceID string) bool
}

// AlwaysSampler always samples
type AlwaysSampler struct{}

func (a *AlwaysSampler) ShouldSample(traceID string) bool {
	return true
}

// NeverSampler never samples
type NeverSampler struct{}

func (n *NeverSampler) ShouldSample(traceID string) bool {
	return false
}

// TraceIDRatioBased samples based on a ratio
type TraceIDRatioBased struct {
	ratio float64
}

// NewTraceIDRatioBased creates a new trace ID ratio based sampler
func NewTraceIDRatioBased(ratio float64) *TraceIDRatioBased {
	return &TraceIDRatioBased{ratio: ratio}
}

func (t *TraceIDRatioBased) ShouldSample(traceID string) bool {
	if t.ratio >= 1.0 {
		return true
	}
	if t.ratio <= 0.0 {
		return false
	}
	// Simple hash-based sampling
	return hash(traceID)%100 < int(t.ratio*100)
}

// TracerMiddleware is a middleware for HTTP tracing
type TracerMiddleware struct {
	tracer *Tracer
}

// NewTracerMiddleware creates a new tracer middleware
func NewTracerMiddleware(tracer *Tracer) *TracerMiddleware {
	return &TracerMiddleware{
		tracer: tracer,
	}
}

// Middleware wraps an HTTP handler with tracing
func (m *TracerMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Start a span for the HTTP request
		ctx, span := m.tracer.Start(ctx, "http.request",
			WithSpanKind(SpanKindServer),
			WithAttributes(map[string]interface{}{
				"http.method": r.Method,
				"http.url":    r.URL.String(),
				"http.host":   r.Host,
			}),
		)
		defer m.tracer.End(span, nil)

		// Wrap the response writer to capture status code
		ww := &statusRecorder{ResponseWriter: w, statusCode: 200}

		// Call the next handler with the new context
		next.ServeHTTP(ww, r.WithContext(ctx))

		// Set attributes based on the response
		m.tracer.SetAttributes(span, map[string]interface{}{
			"http.status_code": ww.statusCode,
		})
	})
}

// statusRecorder wraps an http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// TracedOperation represents a traced operation
type TracedOperation struct {
	tracer *Tracer
	name   string
}

// NewTracedOperation creates a new traced operation
func NewTracedOperation(tracer *Tracer, name string) *TracedOperation {
	return &TracedOperation{
		tracer: tracer,
		name:   name,
	}
}

// Execute executes a function with tracing
func (t *TracedOperation) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := t.tracer.Start(ctx, t.name)
	defer t.tracer.End(span, nil)

	return fn(ctx)
}

// ExecuteWithResult executes a function with tracing and returns a result
func (t *TracedOperation) ExecuteWithResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	ctx, span := t.tracer.Start(ctx, t.name)
	defer t.tracer.End(span, nil)

	result, err := fn(ctx)
	if err != nil {
		return nil, err
	}

	if result != nil {
		t.tracer.SetAttributes(span, map[string]interface{}{
			"result.size": fmt.Sprintf("%v", result),
		})
	}

	return result, nil
}

// TracerFactory creates and manages tracers
type TracerFactory struct {
	tracers map[string]*Tracer
	mu      sync.RWMutex
}

// NewTracerFactory creates a new tracer factory
func NewTracerFactory() *TracerFactory {
	return &TracerFactory{
		tracers: make(map[string]*Tracer),
	}
}

// GetTracer gets or creates a tracer
func (f *TracerFactory) GetTracer(serviceName string) (*Tracer, error) {
	f.mu.RLock()
	tracer, exists := f.tracers[serviceName]
	f.mu.RUnlock()

	if exists {
		return tracer, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double check
	if tracer, exists := f.tracers[serviceName]; exists {
		return tracer, nil
	}

	config := DefaultTracerConfig(serviceName)
	tracer, err := NewTracer(config)
	if err != nil {
		return nil, err
	}

	f.tracers[serviceName] = tracer
	return tracer, nil
}

// GetTracerWithConfig gets a tracer with specific configuration
func (f *TracerFactory) GetTracerWithConfig(config TracerConfig) (*Tracer, error) {
	f.mu.RLock()
	tracer, exists := f.tracers[config.ServiceName]
	f.mu.RUnlock()

	if exists {
		return tracer, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double check
	if tracer, exists := f.tracers[config.ServiceName]; exists {
		return tracer, nil
	}

	tracer, err := NewTracer(config)
	if err != nil {
		return nil, err
	}

	f.tracers[config.ServiceName] = tracer
	return tracer, nil
}

// ShutdownAll shuts down all tracers
func (f *TracerFactory) ShutdownAll(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var lastErr error
	for name, tracer := range f.tracers {
		if err := tracer.Shutdown(ctx); err != nil {
			lastErr = fmt.Errorf("failed to shutdown tracer %s: %w", name, err)
		}
	}
	return lastErr
}

// DistributedTracingConfig represents the configuration for distributed tracing
type DistributedTracingConfig struct {
	Enable             bool
	ServiceName        string
	ServiceVersion     string
	SamplingRate       float64
	EnableBatching     bool
	BatchTimeout       time.Duration
	EnableMiddleware   bool
	EnableDBTracing    bool
	EnableCacheTracing bool
	EnableHTTPTracing  bool
}

// DefaultDistributedTracingConfig creates a default distributed tracing configuration
func DefaultDistributedTracingConfig(serviceName string) DistributedTracingConfig {
	return DistributedTracingConfig{
		Enable:             true,
		ServiceName:        serviceName,
		ServiceVersion:     "1.0.0",
		SamplingRate:       1.0,
		EnableBatching:     true,
		BatchTimeout:       5 * time.Second,
		EnableMiddleware:   true,
		EnableDBTracing:    true,
		EnableCacheTracing: true,
		EnableHTTPTracing:  true,
	}
}

// DistributedTracer represents a distributed tracer
type DistributedTracer struct {
	tracer  *Tracer
	factory *TracerFactory
	config  DistributedTracingConfig
}

// NewDistributedTracer creates a new distributed tracer
func NewDistributedTracer(config DistributedTracingConfig) (*DistributedTracer, error) {
	if !config.Enable {
		return &DistributedTracer{
			config: config,
		}, nil
	}

	tracerConfig := TracerConfig{
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		SamplingRate:   config.SamplingRate,
	}

	if config.EnableBatching {
		exporter := NewConsoleExporter()
		tracerConfig.Processor = NewBatchSpanProcessor(exporter, 100, config.BatchTimeout)
	}

	tracer, err := NewTracer(tracerConfig)
	if err != nil {
		return nil, err
	}

	return &DistributedTracer{
		tracer:  tracer,
		factory: NewTracerFactory(),
		config:  config,
	}, nil
}

// GetTracer gets the tracer
func (d *DistributedTracer) GetTracer() *Tracer {
	return d.tracer
}

// GetFactory gets the factory
func (d *DistributedTracer) GetFactory() *TracerFactory {
	return d.factory
}

// Shutdown shuts down the tracer
func (d *DistributedTracer) Shutdown(ctx context.Context) error {
	if d.tracer != nil {
		if err := d.tracer.Shutdown(ctx); err != nil {
			return err
		}
	}
	if d.factory != nil {
		if err := d.factory.ShutdownAll(ctx); err != nil {
			return err
		}
	}
	return nil
}

// TraceDBQuery traces a database query
func (d *DistributedTracer) TraceDBQuery(ctx context.Context, query string, fn func(ctx context.Context) error) error {
	if !d.config.EnableDBTracing {
		return fn(ctx)
	}

	operation := NewTracedOperation(d.tracer, "db.query")
	return operation.Execute(ctx, func(ctx context.Context) error {
		span := getSpanFromContext(ctx)
		if span != nil {
			d.tracer.SetAttributes(span, map[string]interface{}{
				"db.statement": query,
				"db.system":    "postgresql",
			})
		}
		return fn(ctx)
	})
}

// TraceCacheOperation traces a cache operation
func (d *DistributedTracer) TraceCacheOperation(ctx context.Context, operation, key string, fn func(ctx context.Context) error) error {
	if !d.config.EnableCacheTracing {
		return fn(ctx)
	}

	op := NewTracedOperation(d.tracer, "cache."+operation)
	return op.Execute(ctx, func(ctx context.Context) error {
		span := getSpanFromContext(ctx)
		if span != nil {
			d.tracer.SetAttributes(span, map[string]interface{}{
				"cache.key":       key,
				"cache.operation": operation,
			})
		}
		return fn(ctx)
	})
}

// TraceHTTPCall traces an HTTP call
func (d *DistributedTracer) TraceHTTPCall(ctx context.Context, method, url string, fn func(ctx context.Context) error) error {
	if !d.config.EnableHTTPTracing {
		return fn(ctx)
	}

	op := NewTracedOperation(d.tracer, "http.client")
	return op.Execute(ctx, func(ctx context.Context) error {
		span := getSpanFromContext(ctx)
		if span != nil {
			d.tracer.SetAttributes(span, map[string]interface{}{
				"http.method": method,
				"http.url":    url,
			})
		}
		return fn(ctx)
	})
}

// TraceBusinessOperation traces a business operation
func (d *DistributedTracer) TraceBusinessOperation(ctx context.Context, name string, attrs map[string]interface{}, fn func(ctx context.Context) error) error {
	op := NewTracedOperation(d.tracer, name)
	return op.Execute(ctx, func(ctx context.Context) error {
		span := getSpanFromContext(ctx)
		if span != nil && len(attrs) > 0 {
			d.tracer.SetAttributes(span, attrs)
		}
		return fn(ctx)
	})
}

// GetTraceID gets the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := getSpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.TraceID
}

// GetSpanID gets the span ID from context
func GetSpanID(ctx context.Context) string {
	span := getSpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanID
}

// IsSampled checks if the span is sampled
func IsSampled(ctx context.Context) bool {
	span := getSpanFromContext(ctx)
	if span == nil {
		return false
	}
	return span.Status.Code != StatusCodeUnset
}

// Helper functions

type spanKey struct{}

func getSpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value(spanKey{}).(*Span)
	return span
}

func generateTraceID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

func hash(s string) int {
	h := 0
	for i := 0; i < len(s); i++ {
		h = 31*h + int(s[i])
	}
	if h < 0 {
		return -h
	}
	return h
}

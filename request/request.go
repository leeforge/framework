package request

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RequestIDGenerator generates unique request IDs
type RequestIDGenerator struct {
	prefix  string
	mu      sync.Mutex
	counter uint64
}

// NewRequestIDGenerator creates a new request ID generator
func NewRequestIDGenerator(prefix string) *RequestIDGenerator {
	return &RequestIDGenerator{
		prefix:  prefix,
		counter: 0,
	}
}

// Generate generates a unique request ID
func (g *RequestIDGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.counter++

	// Format: prefix-timestamp-counter-random
	timestamp := time.Now().UnixNano()
	random := make([]byte, 4)
	rand.Read(random)

	id := fmt.Sprintf("%s-%d-%d-%s",
		g.prefix,
		timestamp,
		g.counter,
		hex.EncodeToString(random),
	)

	return id
}

// GenerateTraceID generates a trace ID
func GenerateTraceID() string {
	// 16 bytes = 32 hex characters
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateSpanID generates a span ID
func GenerateSpanID() string {
	// 8 bytes = 16 hex characters
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateCorrelationID generates a correlation ID
func GenerateCorrelationID() string {
	// Generate UUID-like string without external dependency
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// GenerateShortID generates a short unique ID
func GenerateShortID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// RequestContext represents the request context with tracing information
type RequestContext struct {
	RequestID     string
	TraceID       string
	SpanID        string
	CorrelationID string
	UserID        string
	TenantID      string
	IPAddress     string
	UserAgent     string
	Method        string
	Path          string
	Timestamp     time.Time
	Metadata      map[string]string
}

// NewRequestContext creates a new request context
func NewRequestContext() *RequestContext {
	return &RequestContext{
		RequestID:     GenerateShortID(16),
		TraceID:       GenerateTraceID(),
		SpanID:        GenerateSpanID(),
		CorrelationID: GenerateCorrelationID(),
		Timestamp:     time.Now(),
		Metadata:      make(map[string]string),
	}
}

// FromHTTPRequest extracts request context from HTTP request
func FromHTTPRequest(r *http.Request) *RequestContext {
	ctx := &RequestContext{
		Method:    r.Method,
		Path:      r.URL.Path,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	// Extract headers
	ctx.RequestID = getHeader(r, "X-Request-ID", "X-Request-Id")
	ctx.TraceID = getHeader(r, "X-Trace-ID", "X-Trace-Id")
	ctx.SpanID = getHeader(r, "X-Span-ID", "X-Span-Id")
	ctx.CorrelationID = getHeader(r, "X-Correlation-ID", "X-Correlation-Id")
	ctx.UserID = getHeader(r, "X-User-ID", "X-User-Id")
	ctx.TenantID = getHeader(r, "X-Tenant-ID", "X-Tenant-Id")

	// Generate missing IDs
	if ctx.RequestID == "" {
		ctx.RequestID = GenerateShortID(16)
	}
	if ctx.TraceID == "" {
		ctx.TraceID = GenerateTraceID()
	}
	if ctx.SpanID == "" {
		ctx.SpanID = GenerateSpanID()
	}
	if ctx.CorrelationID == "" {
		ctx.CorrelationID = GenerateCorrelationID()
	}

	// Extract custom metadata headers
	for k, v := range r.Header {
		if strings.HasPrefix(k, "X-Meta-") {
			key := strings.TrimPrefix(k, "X-Meta-")
			if len(v) > 0 {
				ctx.Metadata[key] = v[0]
			}
		}
	}

	return ctx
}

// ToContext converts request context to context.Context
func (rc *RequestContext) ToContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestIDKey{}, rc.RequestID)
	ctx = context.WithValue(ctx, traceIDKey{}, rc.TraceID)
	ctx = context.WithValue(ctx, spanIDKey{}, rc.SpanID)
	ctx = context.WithValue(ctx, correlationIDKey{}, rc.CorrelationID)
	ctx = context.WithValue(ctx, userIDKey{}, rc.UserID)
	ctx = context.WithValue(ctx, tenantIDKey{}, rc.TenantID)
	ctx = context.WithValue(ctx, contextKey{}, rc)
	return ctx
}

// FromContext extracts request context from context.Context
func FromContext(ctx context.Context) *RequestContext {
	if rc, ok := ctx.Value(contextKey{}).(*RequestContext); ok {
		return rc
	}

	// Build from individual values
	rc := &RequestContext{
		RequestID:     getRequestIDFromContext(ctx),
		TraceID:       getTraceIDFromContext(ctx),
		SpanID:        getSpanIDFromContext(ctx),
		CorrelationID: getCorrelationIDFromContext(ctx),
		UserID:        getUserIDFromContext(ctx),
		TenantID:      getTenantIDFromContext(ctx),
		Timestamp:     time.Now(),
		Metadata:      make(map[string]string),
	}

	return rc
}

// ToHeaders converts request context to HTTP headers
func (rc *RequestContext) ToHeaders() http.Header {
	headers := make(http.Header)
	if rc.RequestID != "" {
		headers.Set("X-Request-ID", rc.RequestID)
	}
	if rc.TraceID != "" {
		headers.Set("X-Trace-ID", rc.TraceID)
	}
	if rc.SpanID != "" {
		headers.Set("X-Span-ID", rc.SpanID)
	}
	if rc.CorrelationID != "" {
		headers.Set("X-Correlation-ID", rc.CorrelationID)
	}
	if rc.UserID != "" {
		headers.Set("X-User-ID", rc.UserID)
	}
	if rc.TenantID != "" {
		headers.Set("X-Tenant-ID", rc.TenantID)
	}
	for k, v := range rc.Metadata {
		headers.Set(fmt.Sprintf("X-Meta-%s", k), v)
	}
	return headers
}

// String returns a string representation of the request context
func (rc *RequestContext) String() string {
	parts := []string{
		fmt.Sprintf("RequestID: %s", rc.RequestID),
		fmt.Sprintf("TraceID: %s", rc.TraceID),
		fmt.Sprintf("SpanID: %s", rc.SpanID),
		fmt.Sprintf("CorrelationID: %s", rc.CorrelationID),
	}
	if rc.UserID != "" {
		parts = append(parts, fmt.Sprintf("UserID: %s", rc.UserID))
	}
	if rc.TenantID != "" {
		parts = append(parts, fmt.Sprintf("TenantID: %s", rc.TenantID))
	}
	return strings.Join(parts, " | ")
}

// RequestIDMiddleware adds request ID to all incoming requests
type RequestIDMiddleware struct {
	generator *RequestIDGenerator
}

// NewRequestIDMiddleware creates a new request ID middleware
func NewRequestIDMiddleware(prefix string) *RequestIDMiddleware {
	return &RequestIDMiddleware{
		generator: NewRequestIDGenerator(prefix),
	}
}

// Middleware wraps an HTTP handler with request ID generation
func (m *RequestIDMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get or generate request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = m.generator.Generate()
		}

		// Generate trace ID if not present
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = GenerateTraceID()
		}

		// Generate span ID if not present
		spanID := r.Header.Get("X-Span-ID")
		if spanID == "" {
			spanID = GenerateSpanID()
		}

		// Generate correlation ID if not present
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = GenerateCorrelationID()
		}

		// Create request context
		rc := &RequestContext{
			RequestID:     requestID,
			TraceID:       traceID,
			SpanID:        spanID,
			CorrelationID: correlationID,
			UserID:        r.Header.Get("X-User-ID"),
			TenantID:      r.Header.Get("X-Tenant-ID"),
			IPAddress:     getClientIP(r),
			UserAgent:     r.UserAgent(),
			Method:        r.Method,
			Path:          r.URL.Path,
			Timestamp:     time.Now(),
			Metadata:      make(map[string]string),
		}

		// Add to response headers
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Trace-ID", traceID)
		w.Header().Set("X-Span-ID", spanID)
		w.Header().Set("X-Correlation-ID", correlationID)

		// Add to request context
		ctx := rc.ToContext()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestLogger logs request information
type RequestLogger struct {
	logger LoggerInterface
}

// LoggerInterface defines the interface for logging
type LoggerInterface interface {
	Info(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	WithField(key string, value interface{}) interface{}
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger LoggerInterface) *RequestLogger {
	return &RequestLogger{
		logger: logger,
	}
}

// Middleware wraps an HTTP handler with request logging
func (l *RequestLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &statusRecorder{ResponseWriter: w, statusCode: 200}

		// Get request context
		rc := FromContext(r.Context())

		// Log request
		l.logger.Info("http.request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"request_id", rc.RequestID,
			"trace_id", rc.TraceID,
		)

		// Call next handler
		next.ServeHTTP(ww, r)

		// Log response
		duration := time.Since(start)
		l.logger.Info("http.response",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration", duration,
			"request_id", rc.RequestID,
			"trace_id", rc.TraceID,
		)
	})
}

// RequestTracer traces requests
type RequestTracer struct {
	tracer TracerInterface
}

// TracerInterface defines the interface for tracing
type TracerInterface interface {
	Start(ctx context.Context, name string, opts ...interface{}) (context.Context, interface{})
	End(span interface{}, err error)
	SetAttributes(span interface{}, attrs map[string]interface{})
}

// NewRequestTracer creates a new request tracer
func NewRequestTracer(tracer TracerInterface) *RequestTracer {
	return &RequestTracer{
		tracer: tracer,
	}
}

// Middleware wraps an HTTP handler with request tracing
func (t *RequestTracer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Start trace span
		ctx, span := t.tracer.Start(ctx, "http.request")
		defer t.tracer.End(span, nil)

		// Add attributes
		rc := FromContext(ctx)
		t.tracer.SetAttributes(span, map[string]interface{}{
			"http.method":     r.Method,
			"http.url":        r.URL.String(),
			"http.host":       r.Host,
			"http.user_agent": r.UserAgent(),
			"request_id":      rc.RequestID,
			"trace_id":        rc.TraceID,
			"span_id":         rc.SpanID,
		})

		// Wrap response writer
		ww := &statusRecorder{ResponseWriter: w, statusCode: 200}

		// Call next handler
		next.ServeHTTP(ww, r.WithContext(ctx))

		// Add response attributes
		t.tracer.SetAttributes(span, map[string]interface{}{
			"http.status_code": ww.statusCode,
		})
	})
}

// RequestValidator validates requests
type RequestValidator struct {
	validator ValidatorInterface
}

// ValidatorInterface defines the interface for validation
type ValidatorInterface interface {
	ValidateStruct(interface{}) error
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(validator ValidatorInterface) *RequestValidator {
	return &RequestValidator{
		validator: validator,
	}
}

// Validate validates a request body
func (v *RequestValidator) Validate(body interface{}) error {
	return v.validator.ValidateStruct(body)
}

// RequestThrottler throttles requests
type RequestThrottler struct {
	limit    int
	window   time.Duration
	requests map[string][]time.Time
	mu       sync.RWMutex
}

// NewRequestThrottler creates a new request throttler
func NewRequestThrottler(limit int, window time.Duration) *RequestThrottler {
	return &RequestThrottler{
		limit:    limit,
		window:   window,
		requests: make(map[string][]time.Time),
	}
}

// Allow checks if a request is allowed
func (t *RequestThrottler) Allow(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-t.window)

	// Clean old requests
	if timestamps, exists := t.requests[key]; exists {
		var valid []time.Time
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}
		t.requests[key] = valid

		// Check limit
		if len(valid) >= t.limit {
			return false
		}
	}

	// Record request
	t.requests[key] = append(t.requests[key], now)
	return true
}

// GetRemaining gets remaining requests
func (t *RequestThrottler) GetRemaining(key string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-t.window)

	if timestamps, exists := t.requests[key]; exists {
		count := 0
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				count++
			}
		}
		return t.limit - count
	}

	return t.limit
}

// RequestCorrelation correlates requests across services
type RequestCorrelation struct {
	enabled bool
}

// NewRequestCorrelation creates a new request correlator
func NewRequestCorrelation(enabled bool) *RequestCorrelation {
	return &RequestCorrelation{
		enabled: enabled,
	}
}

// Correlate adds correlation headers to an outgoing request
func (rc *RequestCorrelation) Correlate(ctx context.Context, req *http.Request) *http.Request {
	if !rc.enabled {
		return req
	}

	requestCtx := FromContext(ctx)
	if requestCtx == nil {
		return req
	}

	// Add correlation headers
	req.Header.Set("X-Request-ID", requestCtx.RequestID)
	req.Header.Set("X-Trace-ID", requestCtx.TraceID)
	req.Header.Set("X-Span-ID", requestCtx.SpanID)
	req.Header.Set("X-Correlation-ID", requestCtx.CorrelationID)

	if requestCtx.UserID != "" {
		req.Header.Set("X-User-ID", requestCtx.UserID)
	}
	if requestCtx.TenantID != "" {
		req.Header.Set("X-Tenant-ID", requestCtx.TenantID)
	}

	// Add metadata headers
	for k, v := range requestCtx.Metadata {
		req.Header.Set(fmt.Sprintf("X-Meta-%s", k), v)
	}

	return req
}

// RequestIDGeneratorFactory creates request ID generators
type RequestIDGeneratorFactory struct {
	generators map[string]*RequestIDGenerator
	mu         sync.RWMutex
}

// NewRequestIDGeneratorFactory creates a new factory
func NewRequestIDGeneratorFactory() *RequestIDGeneratorFactory {
	return &RequestIDGeneratorFactory{
		generators: make(map[string]*RequestIDGenerator),
	}
}

// GetGenerator gets or creates a generator
func (f *RequestIDGeneratorFactory) GetGenerator(prefix string) *RequestIDGenerator {
	f.mu.RLock()
	gen, exists := f.generators[prefix]
	f.mu.RUnlock()

	if exists {
		return gen
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double check
	if gen, exists := f.generators[prefix]; exists {
		return gen
	}

	gen = NewRequestIDGenerator(prefix)
	f.generators[prefix] = gen
	return gen
}

// Helper functions

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Use RemoteAddr
	return r.RemoteAddr
}

func getHeader(r *http.Request, keys ...string) string {
	for _, key := range keys {
		if value := r.Header.Get(key); value != "" {
			return value
		}
	}
	return ""
}

// Context keys
type requestIDKey struct{}
type traceIDKey struct{}
type spanIDKey struct{}
type correlationIDKey struct{}
type userIDKey struct{}
type tenantIDKey struct{}
type contextKey struct{}

func getRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

func getTraceIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey{}).(string); ok {
		return id
	}
	return ""
}

func getSpanIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(spanIDKey{}).(string); ok {
		return id
	}
	return ""
}

func getCorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}

func getUserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey{}).(string); ok {
		return id
	}
	return ""
}

func getTenantIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(tenantIDKey{}).(string); ok {
		return id
	}
	return ""
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

// RequestIDConfig represents the configuration for request ID generation
type RequestIDConfig struct {
	Prefix      string
	Length      int
	EnableTrace bool
}

// DefaultRequestIDConfig creates a default configuration
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		Prefix:      "req",
		Length:      16,
		EnableTrace: true,
	}
}

// RequestIDManager manages request ID generation and tracking
type RequestIDManager struct {
	generator *RequestIDGenerator
	config    RequestIDConfig
}

// NewRequestIDManager creates a new request ID manager
func NewRequestIDManager(config RequestIDConfig) *RequestIDManager {
	return &RequestIDManager{
		generator: NewRequestIDGenerator(config.Prefix),
		config:    config,
	}
}

// Generate generates a new request ID
func (m *RequestIDManager) Generate() string {
	return m.generator.Generate()
}

// GenerateWithTrace generates a request ID with trace information
func (m *RequestIDManager) GenerateWithTrace() (requestID, traceID, spanID string) {
	requestID = m.Generate()
	traceID = GenerateTraceID()
	spanID = GenerateSpanID()
	return
}

// MiddlewareChain creates a middleware chain for request handling
func MiddlewareChain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// RequestIDValidator validates request IDs
type RequestIDValidator struct {
	pattern string
}

// NewRequestIDValidator creates a new request ID validator
func NewRequestIDValidator(pattern string) *RequestIDValidator {
	return &RequestIDValidator{
		pattern: pattern,
	}
}

// Validate validates a request ID
func (v *RequestIDValidator) Validate(requestID string) bool {
	if requestID == "" {
		return false
	}

	// Basic validation: check length and format
	if len(requestID) < 8 {
		return false
	}

	// Check for expected patterns
	if v.pattern != "" {
		return strings.Contains(requestID, v.pattern)
	}

	return true
}

// RequestIDExtractor extracts request IDs from various sources
type RequestIDExtractor struct {
	sources []string
}

// NewRequestIDExtractor creates a new extractor
func NewRequestIDExtractor(sources ...string) *RequestIDExtractor {
	if len(sources) == 0 {
		sources = []string{"header", "query", "cookie"}
	}
	return &RequestIDExtractor{
		sources: sources,
	}
}

// Extract extracts request ID from HTTP request
func (e *RequestIDExtractor) Extract(r *http.Request) string {
	for _, source := range e.sources {
		switch source {
		case "header":
			if id := r.Header.Get("X-Request-ID"); id != "" {
				return id
			}
		case "query":
			if id := r.URL.Query().Get("request_id"); id != "" {
				return id
			}
		case "cookie":
			if cookie, err := r.Cookie("request_id"); err == nil {
				return cookie.Value
			}
		}
	}
	return ""
}

// RequestIDEnricher enriches request context with additional information
type RequestIDEnricher struct {
	fields map[string]func(*http.Request) string
}

// NewRequestIDEnricher creates a new enricher
func NewRequestIDEnricher() *RequestIDEnricher {
	return &RequestIDEnricher{
		fields: make(map[string]func(*http.Request) string),
	}
}

// AddField adds a field to be enriched
func (e *RequestIDEnricher) AddField(name string, extractor func(*http.Request) string) {
	e.fields[name] = extractor
}

// Enrich enriches a request context
func (e *RequestIDEnricher) Enrich(rc *RequestContext, r *http.Request) {
	for name, extractor := range e.fields {
		rc.Metadata[name] = extractor(r)
	}
}

// DefaultRequestIDEnricher creates a default enricher
func DefaultRequestIDEnricher() *RequestIDEnricher {
	enricher := NewRequestIDEnricher()

	// Add common fields
	enricher.AddField("host", func(r *http.Request) string {
		return r.Host
	})
	enricher.AddField("protocol", func(r *http.Request) string {
		return r.Proto
	})
	enricher.AddField("content_length", func(r *http.Request) string {
		return fmt.Sprintf("%d", r.ContentLength)
	})

	return enricher
}

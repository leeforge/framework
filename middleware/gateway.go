package middleware

import (
	"fmt"
	"net/http"
	"time"
)

type LoggerAdapter interface {
	Info(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
}

type Field struct {
	Key   string
	Value interface{}
}

type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, fields ...Field) {
	fmt.Printf("[INFO] %s\n", msg)
}

func (l *SimpleLogger) Error(msg string, err error, fields ...Field) {
	fmt.Printf("[ERROR] %s: %v\n", msg, err)
}

type MetricsAdapter interface {
	RecordRequest(method, path string, status int, duration float64)
	RecordError(method, path string, err error)
}

type TracerAdapter interface {
	StartSpan(name string, attrs map[string]interface{}) Span
	EndSpan(span Span, err error)
}

type Span interface {
	SetAttribute(key string, value interface{})
	End()
}

type AuthMiddleware interface {
	Middleware(next http.Handler) http.Handler
}

type GatewayMiddleware struct {
	rateLimiter *RateLimiter
	auth        AuthMiddleware
	logger      LoggerAdapter
	metrics     MetricsAdapter
	tracer      TracerAdapter
}

func NewGatewayMiddleware(
	rateLimiter *RateLimiter,
	auth AuthMiddleware,
	logger LoggerAdapter,
	metrics MetricsAdapter,
	tracer TracerAdapter,
) *GatewayMiddleware {
	return &GatewayMiddleware{
		rateLimiter: rateLimiter,
		auth:        auth,
		logger:      logger,
		metrics:     metrics,
		tracer:      tracer,
	}
}

func (g *GatewayMiddleware) Chain() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := next
		h = g.loggingMiddleware(h)
		if g.rateLimiter != nil {
			h = g.rateLimiter.Middleware(h)
		}
		if g.auth != nil {
			h = g.auth.Middleware(h)
		}
		h = g.tracingMiddleware(h)
		h = g.metricsMiddleware(h)
		return h
	}
}

func (g *GatewayMiddleware) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriterWrapper{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(ww, r)
		duration := time.Since(start)
		if g.logger != nil {
			g.logger.Info("HTTP Request")
		}
		_ = duration
	})
}

func (g *GatewayMiddleware) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.tracer == nil {
			next.ServeHTTP(w, r)
			return
		}
		span := g.tracer.StartSpan("http_request", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		defer g.tracer.EndSpan(span, nil)
		next.ServeHTTP(w, r)
	})
}

func (g *GatewayMiddleware) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriterWrapper{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(ww, r)
		duration := time.Since(start).Seconds()
		if g.metrics != nil {
			g.metrics.RecordRequest(r.Method, r.URL.Path, ww.statusCode, duration)
			if ww.statusCode >= 400 {
				g.metrics.RecordError(r.Method, r.URL.Path, fmt.Errorf("status %d", ww.statusCode))
			}
		}
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

type GatewayConfig struct {
	RateLimit  RateLimitConfig
	EnableAuth bool
	LogLevel   string
	Metrics    bool
	Tracing    bool
}

type Gateway struct {
	config     GatewayConfig
	middleware *GatewayMiddleware
	server     *http.Server
	logger     LoggerAdapter
}

func NewGateway(config GatewayConfig, logger LoggerAdapter) (*Gateway, error) {
	backend := NewRedisBackend()
	rateLimiter := NewRateLimiter(backend, config.RateLimit)

	var auth AuthMiddleware
	if config.EnableAuth {
		auth = nil
	}

	var metrics MetricsAdapter
	var tracer TracerAdapter

	gwMiddleware := NewGatewayMiddleware(rateLimiter, auth, logger, metrics, tracer)

	return &Gateway{
		config:     config,
		middleware: gwMiddleware,
		logger:     logger,
	}, nil
}

func (g *Gateway) Handler(next http.Handler) http.Handler {
	return g.middleware.Chain()(next)
}

func (g *Gateway) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	limiterHandler := NewRateLimitHandler(g.middleware.rateLimiter)
	RegisterRateLimitRoutes(mux, limiterHandler)
}

func (g *Gateway) Start(addr string) error {
	mux := http.NewServeMux()
	g.RegisterRoutes(mux)

	g.server = &http.Server{
		Addr:    addr,
		Handler: g.Handler(mux),
	}

	if g.logger != nil {
		g.logger.Info("Gateway starting")
	}
	return g.server.ListenAndServe()
}

func (g *Gateway) Stop() error {
	if g.server == nil {
		return nil
	}
	return g.server.Close()
}

type MiddlewareChain struct {
	middlewares []func(http.Handler) http.Handler
}

func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

func (c *MiddlewareChain) Use(middleware func(http.Handler) http.Handler) *MiddlewareChain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *MiddlewareChain) Then(handler http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

type SecurityMiddleware struct {
	cors        CORSConfig
	helmet      bool
	ipWhitelist []string
	ipBlacklist []string
	requestSize int64
}

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func NewSecurityMiddleware(config SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		cors:        config.CORS,
		helmet:      config.Helmet,
		ipWhitelist: config.IPWhitelist,
		ipBlacklist: config.IPBlacklist,
		requestSize: config.RequestSize,
	}
}

type SecurityConfig struct {
	CORS        CORSConfig
	Helmet      bool
	IPWhitelist []string
	IPBlacklist []string
	RequestSize int64
}

func (s *SecurityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.checkIP(r.RemoteAddr) {
			http.Error(w, "IP not allowed", http.StatusForbidden)
			return
		}

		if s.requestSize > 0 && r.ContentLength > s.requestSize {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}

		if s.cors.Enabled {
			s.applyCORS(w, r)
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if s.helmet {
			s.applyHelmet(w)
		}

		next.ServeHTTP(w, r)
	})
}

func (s *SecurityMiddleware) checkIP(remoteAddr string) bool {
	ip := remoteAddr

	for _, black := range s.ipBlacklist {
		if ip == black {
			return false
		}
	}

	if len(s.ipWhitelist) > 0 {
		for _, white := range s.ipWhitelist {
			if ip == white {
				return true
			}
		}
		return false
	}

	return true
}

func (s *SecurityMiddleware) applyCORS(w http.ResponseWriter, r *http.Request) {
	for _, origin := range s.cors.AllowedOrigins {
		if origin == "*" || origin == r.Header.Get("Origin") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			break
		}
	}

	w.Header().Set("Access-Control-Allow-Methods", joinStrings(s.cors.AllowedMethods, ", "))
	w.Header().Set("Access-Control-Allow-Headers", joinStrings(s.cors.AllowedHeaders, ", "))
	w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%t", s.cors.AllowCredentials))
	w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", s.cors.MaxAge))
}

func (s *SecurityMiddleware) applyHelmet(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}

type GatewayBuilder struct {
	config      GatewayConfig
	logger      LoggerAdapter
	middlewares []func(http.Handler) http.Handler
}

func NewGatewayBuilder(logger LoggerAdapter) *GatewayBuilder {
	return &GatewayBuilder{
		config: GatewayConfig{
			RateLimit: RateLimitConfig{
				DefaultRate:  100,
				DefaultDaily: 1000,
				Burst:        10,
			},
		},
		logger:      logger,
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

func (b *GatewayBuilder) WithRateLimit(config RateLimitConfig) *GatewayBuilder {
	b.config.RateLimit = config
	return b
}

func (b *GatewayBuilder) WithAuth(enabled bool) *GatewayBuilder {
	b.config.EnableAuth = enabled
	return b
}

func (b *GatewayBuilder) WithMetrics(enabled bool) *GatewayBuilder {
	b.config.Metrics = enabled
	return b
}

func (b *GatewayBuilder) WithTracing(enabled bool) *GatewayBuilder {
	b.config.Tracing = enabled
	return b
}

func (b *GatewayBuilder) WithMiddleware(middleware func(http.Handler) http.Handler) *GatewayBuilder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

func (b *GatewayBuilder) Build() (*Gateway, error) {
	return NewGateway(b.config, b.logger)
}

func (b *GatewayBuilder) Handler(next http.Handler) (http.Handler, error) {
	gw, err := b.Build()
	if err != nil {
		return nil, err
	}

	handler := gw.Handler(next)

	for _, middleware := range b.middlewares {
		handler = middleware(handler)
	}

	return handler, nil
}

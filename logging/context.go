package logging

import (
	"context"

	"go.uber.org/zap"
)

// Context keys for trace information.
type ctxKey string

const (
	// TraceIDKey is the context key for trace ID.
	TraceIDKey ctxKey = "trace_id"
	// SpanIDKey is the context key for span ID.
	SpanIDKey ctxKey = "span_id"
	// RequestIDKey is the context key for request ID.
	RequestIDKey ctxKey = "request_id"
	// UserIDKey is the context key for user ID.
	UserIDKey ctxKey = "user_id"
)

// WithContext creates a child logger with fields extracted from the context.
// It extracts trace_id, span_id, request_id, and user_id if present.
func WithContext(logger Logger, ctx context.Context) Logger {
	if ctx == nil {
		return logger
	}

	var fields []zap.Field

	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if spanID := GetSpanID(ctx); spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if userID := GetUserID(ctx); userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields...)
}

// GetTraceID extracts trace ID from context.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetSpanID extracts span ID from context.
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(SpanIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetRequestID extracts request ID from context.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserID extracts user ID from context.
func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(UserIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetTraceID adds trace ID to context.
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// SetSpanID adds span ID to context.
func SetSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// SetRequestID adds request ID to context.
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// SetUserID adds user ID to context.
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// loggerKey is the context key for storing a logger in context.
type loggerKey struct{}

// FromContext returns the Logger stored in the context, or the global logger if none.
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return Global()
	}
	if l, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return l
	}
	return Global()
}

// ToContext stores the Logger in the context.
func ToContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

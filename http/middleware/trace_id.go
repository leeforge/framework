package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	// TraceIDKey is the key for trace ID in context
	TraceIDKey contextKey = "trace_id"
	// TraceIDHeader is the HTTP header name for trace ID
	TraceIDHeader = "X-Trace-ID"
)

// TraceIDMiddleware adds a trace ID to each request
// If the request already has a trace ID in the header, it will be used
// Otherwise, a new UUID will be generated
func TraceIDMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get trace ID from request header
			traceID := r.Header.Get(TraceIDHeader)

			// Generate new trace ID if not present
			if traceID == "" {
				traceID = uuid.New().String()
			}

			// Add trace ID to response header
			w.Header().Set(TraceIDHeader, traceID)

			// Add trace ID to request context
			ctx := context.WithValue(r.Context(), TraceIDKey, traceID)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetTraceIDFromRequest retrieves the trace ID from request context
func GetTraceIDFromRequest(r *http.Request) string {
	return GetTraceID(r.Context())
}

package middleware

import (
	"context"
	"net/http"
	"time"
)

type timingContextKey string

const (
	// StartTimeKey is the key for request start time in context
	StartTimeKey timingContextKey = "start_time"
)

// TimingMiddleware records request start time for calculating processing duration
func TimingMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Record start time
			startTime := time.Now()

			// Add start time to context
			ctx := context.WithValue(r.Context(), StartTimeKey, startTime)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestDuration calculates the duration since request start time
// Returns duration in milliseconds
func GetRequestDuration(ctx context.Context) int64 {
	if startTime, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return time.Since(startTime).Milliseconds()
	}
	return 0
}

// GetRequestDurationFromRequest calculates the duration from request context
func GetRequestDurationFromRequest(r *http.Request) int64 {
	return GetRequestDuration(r.Context())
}

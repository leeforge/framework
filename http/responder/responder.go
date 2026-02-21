package responder

import (
	"net/http"

	"github.com/leeforge/framework/json"
)

// ============================================
// Factory Pattern (Optional, for advanced use)
// ============================================

// ErrorConfig defines custom error code configuration
type ErrorConfig struct {
	Code       int    // Error code (e.g., 4001, 5001)
	Message    string // Default error message
	HTTPStatus int    // HTTP status code (e.g., 400, 500)
}

type ResponderFactory struct {
	panicFn      PanicFn
	customErrors map[int]*ErrorConfig // Custom error code mapping
}

// FactoryOption defines configuration options for ResponderFactory
type FactoryOption func(*ResponderFactory)

// WithPanicFn sets a custom panic handler
func WithPanicFn(panicFn PanicFn) FactoryOption {
	return func(f *ResponderFactory) {
		f.panicFn = panicFn
	}
}

// WithCustomErrors adds custom error code mappings
// Example:
//
//	WithCustomErrors(map[int]*ErrorConfig{
//	  4100: {Code: 4100, Message: "User not found", HTTPStatus: 404},
//	  5100: {Code: 5100, Message: "Payment service error", HTTPStatus: 500},
//	})
func WithCustomErrors(errors map[int]*ErrorConfig) FactoryOption {
	return func(f *ResponderFactory) {
		if f.customErrors == nil {
			f.customErrors = make(map[int]*ErrorConfig)
		}
		for code, config := range errors {
			f.customErrors[code] = config
		}
	}
}

// NewResponderFactory creates a new ResponderFactory with options
func NewResponderFactory(opts ...FactoryOption) *ResponderFactory {
	f := &ResponderFactory{
		panicFn:      DefaultPanicFn,
		customErrors: make(map[int]*ErrorConfig),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

func (f *ResponderFactory) FromRequest(w http.ResponseWriter, r *http.Request) *Responder {
	return &Responder{
		w:            w,
		r:            r,
		panicFn:      f.panicFn,
		customErrors: f.customErrors,
	}
}

// ============================================
// Responder Instance (Optional)
// ============================================

type Responder struct {
	w            http.ResponseWriter
	r            *http.Request
	panicFn      func(http.ResponseWriter, *http.Request, error)
	customErrors map[int]*ErrorConfig
}

func New(w http.ResponseWriter, r *http.Request, panicFn PanicFn) *Responder {
	if panicFn == nil {
		panicFn = func(_ http.ResponseWriter, _ *http.Request, err error) {
			panic(err)
		}
	}
	return &Responder{
		w:            w,
		r:            r,
		panicFn:      panicFn,
		customErrors: make(map[int]*ErrorConfig),
	}
}

// GetCustomError retrieves custom error configuration by code
func (r *Responder) GetCustomError(code int) *ErrorConfig {
	if config, ok := r.customErrors[code]; ok {
		return config
	}
	return nil
}

func (r *Responder) writeRaw(status int, payload []byte, contentType string) {
	r.w.Header().Set("Content-Type", contentType)
	r.w.WriteHeader(status)
	if _, err := r.w.Write(payload); err != nil {
		r.panicFn(r.w, r.r, err)
	}
}

func (r *Responder) writeJson(status int, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		fallback := []byte("{\"error\":{\"code\":500,\"message\":\"encode failed\"}}")
		r.writeRaw(http.StatusInternalServerError, fallback, "application/json")
		r.panicFn(r.w, r.r, err)
		return
	}
	r.writeRaw(status, raw, "application/json")
}

// Write sends a success response with data
func (r *Responder) Write(status int, payload any, opts ...Option) {
	meta := NewMeta(opts...)
	res := &Response{
		Data: payload,
		Meta: *meta,
	}
	r.writeJson(status, res)
}

// WriteList sends a success response with data and pagination
func (r *Responder) WriteList(status int, payload any, pager *PaginationMeta, opts ...Option) {
	opts = append(opts, WithPagination(pager))
	meta := NewMeta(opts...)
	res := &Response{
		Data: payload,
		Meta: *meta,
	}
	r.writeJson(status, res)
}

// WriteError sends an error response
func (r *Responder) WriteError(status int, err Error, opts ...Option) {
	meta := NewMeta(opts...)
	res := &Response{
		Error: &err,
		Meta:  *meta,
	}
	r.writeJson(status, res)
}

// ============================================
// Global Convenience Functions (Recommended)
// ============================================

// writeJSON is the internal helper for all global functions
func writeJSON(w http.ResponseWriter, status int, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		fallback := []byte("{\"error\":{\"code\":500,\"message\":\"encode failed\"}}")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(fallback)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(raw)
}

// Write sends a success response with data
func Write(w http.ResponseWriter, r *http.Request, status int, data any, opts ...Option) {
	meta := NewMeta(opts...)
	res := &Response{
		Data: data,
		Meta: *meta,
	}
	writeJSON(w, status, res)
}

// WriteList sends a success response with data and pagination
func WriteList(w http.ResponseWriter, r *http.Request, status int, data any, pager *PaginationMeta, opts ...Option) {
	opts = append(opts, WithPagination(pager))
	meta := NewMeta(opts...)
	res := &Response{
		Data: data,
		Meta: *meta,
	}
	writeJSON(w, status, res)
}

// WriteError sends an error response
func WriteError(w http.ResponseWriter, r *http.Request, status int, err Error, opts ...Option) {
	meta := NewMeta(opts...)
	res := &Response{
		Error: &err,
		Meta:  *meta,
	}
	writeJSON(w, status, res)
}

// ============================================
// Success Response Shortcuts
// ============================================

// OK responds with 200 OK and data
func OK(w http.ResponseWriter, r *http.Request, data any, opts ...Option) {
	Write(w, r, http.StatusOK, data, opts...)
}

// Created responds with 201 Created and data
func Created(w http.ResponseWriter, r *http.Request, data any, opts ...Option) {
	Write(w, r, http.StatusCreated, data, opts...)
}

// NoContent responds with 204 No Content
func NoContent(w http.ResponseWriter, r *http.Request, opts ...Option) {
	meta := NewMeta(opts...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	if meta.TraceId != "" {
		w.Header().Set("X-Trace-ID", meta.TraceId)
	}
}

// ============================================
// Error Response Shortcuts
// ============================================

// BadRequest responds with 400 Bad Request
func BadRequest(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrBadRequest
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusBadRequest, err, opts...)
}

// Unauthorized responds with 401 Unauthorized
func Unauthorized(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrUnauthorized
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusUnauthorized, err, opts...)
}

// Forbidden responds with 403 Forbidden
func Forbidden(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrForbidden
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusForbidden, err, opts...)
}

// NotFound responds with 404 Not Found
func NotFound(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrNotFound
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusNotFound, err, opts...)
}

// Conflict responds with 409 Conflict
func Conflict(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrConflict
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusConflict, err, opts...)
}

// ValidationError responds with 400 Bad Request and validation details
// Accepts any type of details ([]FieldError, map[string]string, or custom struct)
func ValidationError(w http.ResponseWriter, r *http.Request, details any, opts ...Option) {
	err := NewErrorWithDetails(ErrCodeValidationFailed, "Validation Failed", details)
	WriteError(w, r, http.StatusBadRequest, err, opts...)
}

// BindError responds with 400 Bad Request for binding errors
func BindError(w http.ResponseWriter, r *http.Request, details any, opts ...Option) {
	err := NewErrorWithDetails(ErrCodeBindFailed, "Invalid Request Body", details)
	WriteError(w, r, http.StatusBadRequest, err, opts...)
}

// InternalServerError responds with 500 Internal Server Error
func InternalServerError(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrInternalServer
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusInternalServerError, err, opts...)
}

// DatabaseError responds with 500 Internal Server Error for database errors
func DatabaseError(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrDatabase
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusInternalServerError, err, opts...)
}

// ServiceUnavailable responds with 503 Service Unavailable
func ServiceUnavailable(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := NewError(5003, message)
	if message == "" {
		err.Message = "Service Unavailable"
	}
	WriteError(w, r, http.StatusServiceUnavailable, err, opts...)
}

// TooManyRequests responds with 429 Too Many Requests
func TooManyRequests(w http.ResponseWriter, r *http.Request, message string, opts ...Option) {
	err := ErrTooManyRequests
	if message != "" {
		err.Message = message
	}
	WriteError(w, r, http.StatusTooManyRequests, err, opts...)
}

// CustomError responds with custom status code and error
func CustomError(w http.ResponseWriter, r *http.Request, status int, code int, message string, details any, opts ...Option) {
	err := NewErrorWithDetails(code, message, details)
	WriteError(w, r, status, err, opts...)
}

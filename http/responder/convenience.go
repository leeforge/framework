package responder

import (
	"net/http"
)

// Convenience methods for common success responses

// OK responds with 200 OK and data
func (r *Responder) OK(data any, opts ...Option) {
	r.Write(http.StatusOK, data, opts...)
}

// Created responds with 201 Created and data
func (r *Responder) Created(data any, opts ...Option) {
	r.Write(http.StatusCreated, data, opts...)
}

// NoContent responds with 204 No Content
func (r *Responder) NoContent(opts ...Option) {
	meta := NewMeta(opts...)
	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(http.StatusNoContent)
	// Add meta to response header if traceId exists
	if meta.TraceId != "" {
		r.w.Header().Set("X-Trace-ID", meta.TraceId)
	}
}

// Convenience methods for common error responses

// BadRequest responds with 400 Bad Request
func (r *Responder) BadRequest(message string, opts ...Option) {
	err := ErrBadRequest
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusBadRequest, err, opts...)
}

// Unauthorized responds with 401 Unauthorized
func (r *Responder) Unauthorized(message string, opts ...Option) {
	err := ErrUnauthorized
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusUnauthorized, err, opts...)
}

// Forbidden responds with 403 Forbidden
func (r *Responder) Forbidden(message string, opts ...Option) {
	err := ErrForbidden
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusForbidden, err, opts...)
}

// NotFound responds with 404 Not Found
func (r *Responder) NotFound(message string, opts ...Option) {
	err := ErrNotFound
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusNotFound, err, opts...)
}

// Conflict responds with 409 Conflict
func (r *Responder) Conflict(message string, opts ...Option) {
	err := ErrConflict
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusConflict, err, opts...)
}

// ValidationError responds with 400 Bad Request and validation details
// Accepts []FieldError for detailed field-level errors
func (r *Responder) ValidationError(details any, opts ...Option) {
	err := NewErrorWithDetails(ErrCodeValidationFailed, "Validation Failed", details)
	r.WriteError(http.StatusBadRequest, err, opts...)
}

// BindError responds with 400 Bad Request for binding errors
func (r *Responder) BindError(details any, opts ...Option) {
	err := NewErrorWithDetails(ErrCodeBindFailed, "Invalid Request Body", details)
	r.WriteError(http.StatusBadRequest, err, opts...)
}

// InternalServerError responds with 500 Internal Server Error
func (r *Responder) InternalServerError(message string, opts ...Option) {
	err := ErrInternalServer
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusInternalServerError, err, opts...)
}

// DatabaseError responds with 500 Internal Server Error for database errors
func (r *Responder) DatabaseError(message string, opts ...Option) {
	err := ErrDatabase
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusInternalServerError, err, opts...)
}

// ServiceUnavailable responds with 503 Service Unavailable
func (r *Responder) ServiceUnavailable(message string, opts ...Option) {
	err := NewError(5003, message)
	if message == "" {
		err.Message = "Service Unavailable"
	}
	r.WriteError(http.StatusServiceUnavailable, err, opts...)
}

// TooManyRequests responds with 429 Too Many Requests
func (r *Responder) TooManyRequests(message string, opts ...Option) {
	err := ErrTooManyRequests
	if message != "" {
		err.Message = message
	}
	r.WriteError(http.StatusTooManyRequests, err, opts...)
}

// CustomError responds with custom status code and error
func (r *Responder) CustomError(status int, code int, message string, details any, opts ...Option) {
	err := NewErrorWithDetails(code, message, details)
	r.WriteError(status, err, opts...)
}

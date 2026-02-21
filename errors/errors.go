package errors

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Validation errors
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeRequired   ErrorType = "required"
	ErrorTypeInvalid    ErrorType = "invalid"

	// Database errors
	ErrorTypeDatabase ErrorType = "database"
	ErrorTypeNotFound ErrorType = "not_found"
	ErrorTypeConflict ErrorType = "conflict"

	// Authorization errors
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeForbidden    ErrorType = "forbidden"

	// Business errors
	ErrorTypeBusiness  ErrorType = "business"
	ErrorTypeRateLimit ErrorType = "rate_limit"
	ErrorTypeTimeout   ErrorType = "timeout"

	// System errors
	ErrorTypeInternal ErrorType = "internal"
	ErrorTypeExternal ErrorType = "external"
	ErrorTypeUnknown  ErrorType = "unknown"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType              `json:"type"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	InnerError error                  `json:"-"`
	Stack      []string               `json:"-"`
	HTTPStatus int                    `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.InnerError != nil {
		return e.InnerError.Error()
	}
	return string(e.Type)
}

// Unwrap returns the inner error
func (e *AppError) Unwrap() error {
	return e.InnerError
}

// WithMessage adds a message to the error
func (e *AppError) WithMessage(msg string) *AppError {
	e.Message = msg
	return e
}

// WithCode adds a code to the error
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithDetails adds multiple details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithHTTPStatus sets the HTTP status code
func (e *AppError) WithHTTPStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

// WithInnerError sets the inner error
func (e *AppError) WithInnerError(err error) *AppError {
	e.InnerError = err
	return e
}

// WithStack captures the call stack
func (e *AppError) WithStack() *AppError {
	e.Stack = captureStack(3) // Skip this method and the caller
	return e
}

// Is checks if this error is of a specific type
func (e *AppError) Is(target error) bool {
	if targetApp, ok := target.(*AppError); ok {
		return e.Type == targetApp.Type
	}
	return false
}

// New creates a new AppError
func New(errType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Code:    string(errType),
	}
}

// FromError converts a standard error to AppError
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return &AppError{
		Type:       ErrorTypeUnknown,
		Message:    err.Error(),
		InnerError: err,
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) *AppError {
	return FromError(err).WithMessage(message)
}

// WrapWithType wraps an error with a specific type
func WrapWithType(err error, errType ErrorType, message string) *AppError {
	return &AppError{
		Type:       errType,
		Message:    message,
		InnerError: err,
		Code:       string(errType),
	}
}

// Validation errors
func NewValidation(message string) *AppError {
	return New(ErrorTypeValidation, message).WithHTTPStatus(http.StatusBadRequest)
}

func NewRequired(field string) *AppError {
	return New(ErrorTypeRequired, fmt.Sprintf("%s is required", field)).
		WithDetail("field", field).
		WithHTTPStatus(http.StatusBadRequest)
}

func NewInvalid(field string, value interface{}, reason string) *AppError {
	return New(ErrorTypeInvalid, fmt.Sprintf("invalid value for %s: %v", field, value)).
		WithDetail("field", field).
		WithDetail("value", value).
		WithDetail("reason", reason).
		WithHTTPStatus(http.StatusBadRequest)
}

// Database errors
func NewDatabase(message string) *AppError {
	return New(ErrorTypeDatabase, message).WithHTTPStatus(http.StatusInternalServerError)
}

func NewNotFound(resource string, id interface{}) *AppError {
	return New(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource)).
		WithDetail("resource", resource).
		WithDetail("id", id).
		WithHTTPStatus(http.StatusNotFound)
}

func NewConflict(resource string, id interface{}) *AppError {
	return New(ErrorTypeConflict, fmt.Sprintf("%s already exists", resource)).
		WithDetail("resource", resource).
		WithDetail("id", id).
		WithHTTPStatus(http.StatusConflict)
}

// Authorization errors
func NewUnauthorized(message string) *AppError {
	return New(ErrorTypeUnauthorized, message).WithHTTPStatus(http.StatusUnauthorized)
}

func NewForbidden(message string) *AppError {
	return New(ErrorTypeForbidden, message).WithHTTPStatus(http.StatusForbidden)
}

// Business errors
func NewBusiness(message string) *AppError {
	return New(ErrorTypeBusiness, message).WithHTTPStatus(http.StatusBadRequest)
}

func NewRateLimit(message string) *AppError {
	return New(ErrorTypeRateLimit, message).WithHTTPStatus(http.StatusTooManyRequests)
}

func NewTimeout(message string) *AppError {
	return New(ErrorTypeTimeout, message).WithHTTPStatus(http.StatusRequestTimeout)
}

// System errors
func NewInternal(message string) *AppError {
	return New(ErrorTypeInternal, message).WithHTTPStatus(http.StatusInternalServerError)
}

func NewExternal(message string) *AppError {
	return New(ErrorTypeExternal, message).WithHTTPStatus(http.StatusBadGateway)
}

// Error codes for specific scenarios
const (
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeRequiredField      = "REQUIRED_FIELD"
	CodeInvalidField       = "INVALID_FIELD"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeRateLimit          = "RATE_LIMIT"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ErrorRegistry manages error definitions
type ErrorRegistry struct {
	errors map[string]*AppError
}

// NewErrorRegistry creates a new error registry
func NewErrorRegistry() *ErrorRegistry {
	return &ErrorRegistry{
		errors: make(map[string]*AppError),
	}
}

// Register registers an error template
func (r *ErrorRegistry) Register(code string, err *AppError) {
	r.errors[code] = err
}

// Get retrieves a registered error template
func (r *ErrorRegistry) Get(code string) *AppError {
	if err, ok := r.errors[code]; ok {
		return err
	}
	return nil
}

// Create creates a new error from a registered template
func (r *ErrorRegistry) Create(code string, details map[string]interface{}) *AppError {
	if template := r.Get(code); template != nil {
		err := &AppError{
			Type:       template.Type,
			Code:       code,
			Message:    template.Message,
			Details:    make(map[string]interface{}),
			HTTPStatus: template.HTTPStatus,
		}
		for k, v := range template.Details {
			err.Details[k] = v
		}
		for k, v := range details {
			err.Details[k] = v
		}
		return err
	}
	return New(ErrorTypeUnknown, "Unknown error code").WithDetail("code", code)
}

// DefaultErrorRegistry creates a default error registry with common errors
func DefaultErrorRegistry() *ErrorRegistry {
	registry := NewErrorRegistry()

	// Validation errors
	registry.Register(CodeValidationFailed, NewValidation("Validation failed"))
	registry.Register(CodeRequiredField, NewRequired("field"))
	registry.Register(CodeInvalidField, NewInvalid("field", "value", "reason"))

	// Database errors
	registry.Register(CodeNotFound, NewNotFound("resource", "id"))
	registry.Register(CodeConflict, NewConflict("resource", "id"))

	// Authorization errors
	registry.Register(CodeUnauthorized, NewUnauthorized("Authentication required"))
	registry.Register(CodeForbidden, NewForbidden("Access denied"))

	// System errors
	registry.Register(CodeRateLimit, NewRateLimit("Rate limit exceeded"))
	registry.Register(CodeInternalError, NewInternal("Internal server error"))
	registry.Register(CodeServiceUnavailable, NewExternal("Service unavailable"))

	return registry
}

// ErrorHandler handles errors in a standardized way
type ErrorHandler struct {
	registry *ErrorRegistry
	handlers map[ErrorType]func(*AppError) *AppError
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(registry *ErrorRegistry) *ErrorHandler {
	return &ErrorHandler{
		registry: registry,
		handlers: make(map[ErrorType]func(*AppError) *AppError),
	}
}

// Handle handles an error
func (h *ErrorHandler) Handle(err error) *AppError {
	if err == nil {
		return nil
	}

	appErr := FromError(err)

	// Apply type-specific handlers
	if handler, ok := h.handlers[appErr.Type]; ok {
		return handler(appErr)
	}

	return appErr
}

// HandleFunc registers a handler for a specific error type
func (h *ErrorHandler) HandleFunc(errType ErrorType, fn func(*AppError) *AppError) {
	h.handlers[errType] = fn
}

// Wrap wraps an error with context
func (h *ErrorHandler) Wrap(err error, message string) *AppError {
	return h.Handle(Wrap(err, message))
}

// ErrorConverter converts errors to HTTP responses
type ErrorConverter struct {
	errorHandler *ErrorHandler
}

// NewErrorConverter creates a new error converter
func NewErrorConverter(errorHandler *ErrorHandler) *ErrorConverter {
	return &ErrorConverter{
		errorHandler: errorHandler,
	}
}

// ToHTTPResponse converts an error to an HTTP response
func (c *ErrorConverter) ToHTTPResponse(err error) HTTPErrorResponse {
	appErr := c.errorHandler.Handle(err)

	response := HTTPErrorResponse{
		Error: ErrorResponse{
			Type:    string(appErr.Type),
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	}

	if len(appErr.Details) > 0 {
		response.Error.Details = appErr.Details
	}

	if appErr.HTTPStatus > 0 {
		response.HTTPStatus = appErr.HTTPStatus
	} else {
		response.HTTPStatus = http.StatusInternalServerError
	}

	return response
}

// HTTPErrorResponse represents an HTTP error response
type HTTPErrorResponse struct {
	HTTPStatus int           `json:"-"`
	Error      ErrorResponse `json:"error"`
}

// ErrorResponse represents the error part of an HTTP response
type ErrorResponse struct {
	Type    string                 `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorFormatter formats errors for display
type ErrorFormatter struct {
	showStack bool
	showInner bool
}

// NewErrorFormatter creates a new error formatter
func NewErrorFormatter(showStack bool, showInner bool) *ErrorFormatter {
	return &ErrorFormatter{
		showStack: showStack,
		showInner: showInner,
	}
}

// Format formats an error as a string
func (f *ErrorFormatter) Format(err error) string {
	if err == nil {
		return ""
	}

	appErr := FromError(err)

	var parts []string
	parts = append(parts, fmt.Sprintf("[%s] %s", appErr.Type, appErr.Message))

	if appErr.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", appErr.Code))
	}

	if len(appErr.Details) > 0 {
		for k, v := range appErr.Details {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}

	if f.showStack && len(appErr.Stack) > 0 {
		parts = append(parts, "stack:")
		for _, s := range appErr.Stack {
			parts = append(parts, "  "+s)
		}
	}

	if f.showInner && appErr.InnerError != nil {
		parts = append(parts, "caused_by: "+appErr.InnerError.Error())
	}

	return strings.Join(parts, " | ")
}

// ErrorLogger logs errors with context
type ErrorLogger struct {
	logger ErrorLoggerInterface
}

// ErrorLoggerInterface defines the interface for logging
type ErrorLoggerInterface interface {
	Error(msg string, fields ...interface{})
	Errorf(format string, args ...interface{})
	WithField(key string, value interface{}) interface{}
}

// NewErrorLogger creates a new error logger
func NewErrorLogger(logger ErrorLoggerInterface) *ErrorLogger {
	return &ErrorLogger{
		logger: logger,
	}
}

// Log logs an error with context
func (l *ErrorLogger) Log(err error, context map[string]interface{}) {
	if err == nil {
		return
	}

	appErr := FromError(err)

	fields := make(map[string]interface{})
	fields["error_type"] = appErr.Type
	fields["error_code"] = appErr.Code
	fields["error_message"] = appErr.Message

	if len(appErr.Details) > 0 {
		for k, v := range appErr.Details {
			fields["detail_"+k] = v
		}
	}

	if len(appErr.Stack) > 0 {
		fields["stack"] = appErr.Stack
	}

	if context != nil {
		for k, v := range context {
			fields[k] = v
		}
	}

	l.logger.Errorf("Error occurred: %s", appErr.Error())
}

// ErrorRecover recovers from panics and converts them to errors
func ErrorRecover() (err error) {
	if r := recover(); r != nil {
		switch v := r.(type) {
		case error:
			err = v
		case string:
			err = errors.New(v)
		default:
			err = fmt.Errorf("%v", v)
		}
		err = Wrap(err, "panic recovered")
	}
	return
}

// ErrorRecoverWithHandler recovers from panics and handles them
func ErrorRecoverWithHandler(handler func(*AppError)) {
	if r := recover(); r != nil {
		var appErr *AppError
		switch v := r.(type) {
		case error:
			appErr = Wrap(v, "panic recovered")
		case string:
			appErr = New(ErrorTypeInternal, v)
		default:
			appErr = New(ErrorTypeInternal, fmt.Sprintf("%v", v))
		}
		appErr = appErr.WithStack()
		handler(appErr)
	}
}

// ErrorRetryer retries operations that may fail
type ErrorRetryer struct {
	maxAttempts int
	retryDelay  func(attempt int) int64
	retryable   func(error) bool
}

// NewErrorRetryer creates a new error retryer
func NewErrorRetryer(maxAttempts int) *ErrorRetryer {
	return &ErrorRetryer{
		maxAttempts: maxAttempts,
		retryDelay: func(attempt int) int64 {
			return int64(attempt * attempt * 100) // Exponential backoff
		},
		retryable: func(err error) bool {
			appErr := FromError(err)
			// Retry on internal, external, timeout, and rate limit errors
			return appErr.Type == ErrorTypeInternal ||
				appErr.Type == ErrorTypeExternal ||
				appErr.Type == ErrorTypeTimeout ||
				appErr.Type == ErrorTypeRateLimit
		},
	}
}

// WithRetryDelay sets the retry delay function
func (r *ErrorRetryer) WithRetryDelay(fn func(int) int64) *ErrorRetryer {
	r.retryDelay = fn
	return r
}

// WithRetryable sets the retryable function
func (r *ErrorRetryer) WithRetryable(fn func(error) bool) *ErrorRetryer {
	r.retryable = fn
	return r
}

// Do executes a function with retry logic
func (r *ErrorRetryer) Do(fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if attempt < r.maxAttempts && r.retryable(err) {
			// Wait before retry (in production, use time.Sleep)
			_ = r.retryDelay(attempt)
			continue
		}

		// Don't retry or max attempts reached
		break
	}

	return lastErr
}

// ErrorValidator validates errors
type ErrorValidator struct {
	allowedTypes map[ErrorType]bool
}

// NewErrorValidator creates a new error validator
func NewErrorValidator(allowedTypes ...ErrorType) *ErrorValidator {
	v := &ErrorValidator{
		allowedTypes: make(map[ErrorType]bool),
	}
	for _, t := range allowedTypes {
		v.allowedTypes[t] = true
	}
	return v
}

// IsAllowed checks if an error type is allowed
func (v *ErrorValidator) IsAllowed(err error) bool {
	if err == nil {
		return true
	}

	appErr := FromError(err)
	return v.allowedTypes[appErr.Type]
}

// Validate validates an error against rules
func (v *ErrorValidator) Validate(err error) error {
	if err == nil {
		return nil
	}

	appErr := FromError(err)
	if !v.IsAllowed(err) {
		return New(ErrorTypeInternal, "Unexpected error type").
			WithDetail("type", appErr.Type).
			WithInnerError(err)
	}

	return err
}

// captureStack captures the call stack
func captureStack(skip int) []string {
	var stack []string
	for i := skip; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		funcName := fn.Name()
		// Shorten function name
		if idx := strings.LastIndex(funcName, "/"); idx >= 0 {
			funcName = funcName[idx+1:]
		}

		stack = append(stack, fmt.Sprintf("%s:%d %s", file, line, funcName))
	}
	return stack
}

// ErrorChain represents a chain of errors
type ErrorChain struct {
	errors []*AppError
}

// NewErrorChain creates a new error chain
func NewErrorChain() *ErrorChain {
	return &ErrorChain{
		errors: make([]*AppError, 0),
	}
}

// Add adds an error to the chain
func (c *ErrorChain) Add(err *AppError) *ErrorChain {
	if err != nil {
		c.errors = append(c.errors, err)
	}
	return c
}

// HasErrors checks if the chain has errors
func (c *ErrorChain) HasErrors() bool {
	return len(c.errors) > 0
}

// Error returns the combined error message
func (c *ErrorChain) Error() string {
	if !c.HasErrors() {
		return ""
	}

	var messages []string
	for _, err := range c.errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, " | ")
}

// Errors returns all errors in the chain
func (c *ErrorChain) Errors() []*AppError {
	return c.errors
}

// Last returns the last error in the chain
func (c *ErrorChain) Last() *AppError {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[len(c.errors)-1]
}

// First returns the first error in the chain
func (c *ErrorChain) First() *AppError {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[0]
}

// Filter filters errors by type
func (c *ErrorChain) Filter(errType ErrorType) *ErrorChain {
	filtered := NewErrorChain()
	for _, err := range c.errors {
		if err.Type == errType {
			filtered.Add(err)
		}
	}
	return filtered
}

// HasType checks if the chain has an error of the specified type
func (c *ErrorChain) HasType(errType ErrorType) bool {
	for _, err := range c.errors {
		if err.Type == errType {
			return true
		}
	}
	return false
}

// ToHTTPStatus converts the error chain to an HTTP status code
func (c *ErrorChain) ToHTTPStatus() int {
	if !c.HasErrors() {
		return http.StatusOK
	}

	// Return the highest priority status
	// Priority: 401 > 403 > 404 > 400 > 500
	statusMap := make(map[int]bool)
	for _, err := range c.errors {
		if err.HTTPStatus > 0 {
			statusMap[err.HTTPStatus] = true
		}
	}

	// Check in priority order
	priorities := []int{401, 403, 404, 400, 500}
	for _, status := range priorities {
		if statusMap[status] {
			return status
		}
	}

	return http.StatusInternalServerError
}

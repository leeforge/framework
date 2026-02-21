package responder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponderWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var panicCalled bool
	res := New(rr, req, func(_ http.ResponseWriter, _ *http.Request, _ error) {
		panicCalled = true
	})

	res.Write(http.StatusCreated, "hello", WithTraceID("trace"), WithTook(42))

	if panicCalled {
		t.Fatalf("panicFn should not be called on success")
	}

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Data == nil {
		t.Fatalf("expected data, got nil")
	}

	dataStr, ok := resp.Data.(string)
	if !ok || dataStr != "hello" {
		t.Fatalf("unexpected data payload: %+v", resp.Data)
	}

	if resp.Error != nil {
		t.Fatalf("expected nil error, got %+v", resp.Error)
	}

	if resp.Meta.TraceId != "trace" || resp.Meta.Took != 42 {
		t.Fatalf("unexpected meta: %+v", resp.Meta)
	}
}

func TestResponderWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	res := New(rr, req, nil) // nil panicFn should default to panic, but handler shouldn't error

	errPayload := Error{Code: 4001, Message: "bad request"}
	res.WriteError(http.StatusBadRequest, errPayload, WithTraceID("trace-err"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("expected error payload")
	}

	if resp.Error.Code != errPayload.Code || resp.Error.Message != errPayload.Message {
		t.Fatalf("unexpected error payload: %+v", resp.Error)
	}

	if resp.Meta.TraceId != "trace-err" {
		t.Fatalf("unexpected meta: %+v", resp.Meta)
	}
}

func TestResponderWriteFallbackOnMarshalError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var capturedErr error
	res := New(rr, req, func(_ http.ResponseWriter, _ *http.Request, err error) {
		capturedErr = err
	})

	payload := map[string]any{
		"unsupported": make(chan int),
	}

	res.Write(http.StatusOK, payload)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected fallback status 500, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}

	expected := "{\"error\":{\"code\":500,\"message\":\"encode failed\"}}"
	if body := rr.Body.String(); body != expected {
		t.Fatalf("unexpected fallback body: %s", body)
	}

	if capturedErr == nil {
		t.Fatalf("expected panicFn to receive the marshal error")
	}
}

func TestResponderFactory(t *testing.T) {
	customErrors := map[int]*ErrorConfig{
		4100: {Code: 4100, Message: "User not found", HTTPStatus: 404},
		5100: {Code: 5100, Message: "Payment service error", HTTPStatus: 500},
	}

	factory := NewResponderFactory(
		WithCustomErrors(customErrors),
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	res := factory.FromRequest(rr, req)

	// Test custom error retrieval
	if config := res.GetCustomError(4100); config == nil {
		t.Fatalf("expected to find custom error 4100")
	} else {
		if config.Code != 4100 || config.Message != "User not found" || config.HTTPStatus != 404 {
			t.Fatalf("unexpected custom error config: %+v", config)
		}
	}

	// Test non-existent custom error
	if config := res.GetCustomError(9999); config != nil {
		t.Fatalf("expected nil for non-existent error code")
	}
}

func TestResponderFactoryWithOptions(t *testing.T) {
	panicCalled := false
	customPanic := func(_ http.ResponseWriter, _ *http.Request, _ error) {
		panicCalled = true
	}

	factory := NewResponderFactory(
		WithPanicFn(customPanic),
		WithCustomErrors(map[int]*ErrorConfig{
			4200: {Code: 4200, Message: "Custom error", HTTPStatus: 400},
		}),
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	res := factory.FromRequest(rr, req)

	// Trigger marshal error to test custom panic handler
	payload := map[string]any{
		"unsupported": make(chan int),
	}
	res.Write(http.StatusOK, payload)

	if !panicCalled {
		t.Fatalf("expected custom panic handler to be called")
	}

	// Verify custom error is accessible
	if config := res.GetCustomError(4200); config == nil {
		t.Fatalf("expected custom error 4200 to be accessible")
	}
}

func TestValidationError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// Test with FieldError slice
	validationErrors := []FieldError{
		{Field: "email", Message: "Invalid email format"},
		{Field: "password", Message: "Password too short"},
	}

	ValidationError(rr, req, validationErrors)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("expected error in response")
	}

	if resp.Error.Code != ErrCodeValidationFailed {
		t.Fatalf("expected error code %d, got %d", ErrCodeValidationFailed, resp.Error.Code)
	}

	if resp.Error.Details == nil {
		t.Fatalf("expected details in error response")
	}
}

func TestValidationErrorWithMap(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// Test with map[string]string
	validationErrors := map[string]string{
		"email":    "Invalid email format",
		"password": "Password too short",
	}

	ValidationError(rr, req, validationErrors)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error == nil || resp.Error.Details == nil {
		t.Fatalf("expected error with details in response")
	}
}

func TestResponderValidationError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	res := New(rr, req, nil)

	details := []FieldError{
		{Field: "username", Message: "Username is required"},
	}

	res.ValidationError(details)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("expected error in response")
	}

	if resp.Error.Code != ErrCodeValidationFailed {
		t.Fatalf("expected validation error code")
	}
}

func TestCustomErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	CustomError(rr, req, http.StatusTeapot, 4180, "I'm a teapot", map[string]string{
		"reason": "Coffee not supported",
	})

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("expected error in response")
	}

	if resp.Error.Code != 4180 {
		t.Fatalf("expected error code 4180, got %d", resp.Error.Code)
	}

	if resp.Error.Message != "I'm a teapot" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}

	if resp.Error.Details == nil {
		t.Fatalf("expected details in error")
	}
}

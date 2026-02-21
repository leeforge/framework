package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestContext holds test context and utilities
type TestContext struct {
	t          *testing.T
	ctx        context.Context
	cancel     context.CancelFunc
	components map[string]interface{}
	mu         sync.Mutex
}

// NewTestContext creates a new test context
func NewTestContext(t *testing.T) *TestContext {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	return &TestContext{
		t:          t,
		ctx:        ctx,
		cancel:     cancel,
		components: make(map[string]interface{}),
	}
}

// Context returns the context
func (tc *TestContext) Context() context.Context {
	return tc.ctx
}

// Cleanup cleans up resources
func (tc *TestContext) Cleanup() {
	tc.cancel()
	for name, component := range tc.components {
		if cleaner, ok := component.(interface{ Cleanup() }); ok {
			cleaner.Cleanup()
		}
		if closer, ok := component.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				tc.t.Logf("Failed to close %s: %v", name, err)
			}
		}
	}
}

// Set stores a component
func (tc *TestContext) Set(name string, component interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.components[name] = component
}

// Get retrieves a component
func (tc *TestContext) Get(name string) interface{} {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.components[name]
}

// MustGet retrieves a component or fails the test
func (tc *TestContext) MustGet(name string) interface{} {
	val := tc.Get(name)
	if val == nil {
		tc.t.Fatalf("Component %s not found", name)
	}
	return val
}

// AssertEqual asserts two values are equal
func (tc *TestContext) AssertEqual(expected, actual interface{}, msg string) {
	if !reflect.DeepEqual(expected, actual) {
		tc.t.Errorf("%s\nExpected: %v\nActual: %v", msg, expected, actual)
	}
}

// AssertNotEqual asserts two values are not equal
func (tc *TestContext) AssertNotEqual(expected, actual interface{}, msg string) {
	if reflect.DeepEqual(expected, actual) {
		tc.t.Errorf("%s\nValues should not be equal: %v", msg, actual)
	}
}

// AssertNil asserts value is nil
func (tc *TestContext) AssertNil(value interface{}, msg string) {
	if value != nil {
		tc.t.Errorf("%s\nExpected nil, got: %v", msg, value)
	}
}

// AssertNotNil asserts value is not nil
func (tc *TestContext) AssertNotNil(value interface{}, msg string) {
	if value == nil {
		tc.t.Errorf("%s\nExpected non-nil value", msg)
	}
}

// AssertTrue asserts value is true
func (tc *TestContext) AssertTrue(value bool, msg string) {
	if !value {
		tc.t.Errorf("%s\nExpected true, got false", msg)
	}
}

// AssertFalse asserts value is false
func (tc *TestContext) AssertFalse(value bool, msg string) {
	if value {
		tc.t.Errorf("%s\nExpected false, got true", msg)
	}
}

// AssertError asserts error is not nil
func (tc *TestContext) AssertError(err error, msg string) {
	if err == nil {
		tc.t.Errorf("%s\nExpected error, got nil", msg)
	}
}

// AssertNoError asserts error is nil
func (tc *TestContext) AssertNoError(err error, msg string) {
	if err != nil {
		tc.t.Errorf("%s\nExpected no error, got: %v", msg, err)
	}
}

// AssertContains asserts string contains substring
func (tc *TestContext) AssertContains(s, substr string, msg string) {
	if !strings.Contains(s, substr) {
		tc.t.Errorf("%s\nExpected string to contain %q, got: %q", msg, substr, s)
	}
}

// AssertLen asserts length
func (tc *TestContext) AssertLen(value interface{}, expected int, msg string) {
	v := reflect.ValueOf(value)
	if v.Len() != expected {
		tc.t.Errorf("%s\nExpected length %d, got %d", msg, expected, v.Len())
	}
}

// HTTPTestClient is a test HTTP client
type HTTPTestClient struct {
	server *httptest.Server
	client *http.Client
}

// NewHTTPTestClient creates a new HTTP test client
func NewHTTPTestClient(handler http.Handler) *HTTPTestClient {
	server := httptest.NewServer(handler)
	return &HTTPTestClient{
		server: server,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Get makes a GET request
func (c *HTTPTestClient) Get(path string, headers map[string]string) (*http.Response, string, error) {
	req, err := http.NewRequest("GET", c.server.URL+path, nil)
	if err != nil {
		return nil, "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}

	return resp, string(body), nil
}

// Post makes a POST request
func (c *HTTPTestClient) Post(path string, body interface{}, headers map[string]string) (*http.Response, string, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest("POST", c.server.URL+path, bodyReader)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}

	return resp, string(respBody), nil
}

// Put makes a PUT request
func (c *HTTPTestClient) Put(path string, body interface{}, headers map[string]string) (*http.Response, string, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest("PUT", c.server.URL+path, bodyReader)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}

	return resp, string(respBody), nil
}

// Delete makes a DELETE request
func (c *HTTPTestClient) Delete(path string, headers map[string]string) (*http.Response, string, error) {
	req, err := http.NewRequest("DELETE", c.server.URL+path, nil)
	if err != nil {
		return nil, "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}

	return resp, string(body), nil
}

// Close closes the test server
func (c *HTTPTestClient) Close() {
	if c.server != nil {
		c.server.Close()
	}
}

// MockLogger is a mock logger for testing
type MockLogger struct {
	mu      sync.Mutex
	entries []LogEntry
}

// LogEntry represents a log entry
type LogEntry struct {
	Level  string
	Msg    string
	Fields map[string]interface{}
}

// NewMockLogger creates a new mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{
		entries: make([]LogEntry, 0),
	}
}

// Debug logs debug level
func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.log("DEBUG", msg, fields)
}

// Info logs info level
func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.log("INFO", msg, fields)
}

// Warn logs warn level
func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.log("WARN", msg, fields)
}

// Error logs error level
func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.log("ERROR", msg, fields)
}

// Fatal logs fatal level
func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.log("FATAL", msg, fields)
}

// Debugf logs debug level with format
func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.log("DEBUG", fmt.Sprintf(format, args...), nil)
}

// Infof logs info level with format
func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.log("INFO", fmt.Sprintf(format, args...), nil)
}

// Warnf logs warn level with format
func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.log("WARN", fmt.Sprintf(format, args...), nil)
}

// Errorf logs error level with format
func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.log("ERROR", fmt.Sprintf(format, args...), nil)
}

// Fatalf logs fatal level with format
func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.log("FATAL", fmt.Sprintf(format, args...), nil)
}

func (m *MockLogger) log(level, msg string, fields []interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := LogEntry{
		Level:  level,
		Msg:    msg,
		Fields: make(map[string]interface{}),
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	m.entries = append(m.entries, entry)
}

// GetEntries returns all log entries
func (m *MockLogger) GetEntries() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]LogEntry(nil), m.entries...)
}

// Clear clears all entries
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make([]LogEntry, 0)
}

// FindEntry finds an entry by message
func (m *MockLogger) FindEntry(msg string) *LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range m.entries {
		if entry.Msg == msg {
			return &entry
		}
	}
	return nil
}

// CountEntries counts entries with a level
func (m *MockLogger) CountEntries(level string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, entry := range m.entries {
		if entry.Level == level {
			count++
		}
	}
	return count
}

// MockRepository is a mock repository for testing
type MockRepository struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		data: make(map[string]interface{}),
	}
}

// Set sets a value
func (m *MockRepository) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Get gets a value
func (m *MockRepository) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key]
}

// Delete deletes a value
func (m *MockRepository) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Clear clears all data
func (m *MockRepository) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
}

// MockCache is a mock cache for testing
type MockCache struct {
	data map[string]interface{}
	ttl  map[string]time.Time
	mu   sync.RWMutex
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
		ttl:  make(map[string]time.Time),
	}
}

// Set sets a value with TTL
func (m *MockCache) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	if ttl > 0 {
		m.ttl[key] = time.Now().Add(ttl)
	}
}

// Get gets a value
func (m *MockCache) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if ttl, exists := m.ttl[key]; exists {
		if time.Now().After(ttl) {
			return nil
		}
	}
	return m.data[key]
}

// Delete deletes a value
func (m *MockCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	delete(m.ttl, key)
}

// Clear clears all data
func (m *MockCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
	m.ttl = make(map[string]time.Time)
}

// MockDB is a mock database for testing
type MockDB struct {
	tables map[string][]map[string]interface{}
	mu     sync.RWMutex
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		tables: make(map[string][]map[string]interface{}),
	}
}

// Insert inserts a record
func (m *MockDB) Insert(table string, record map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tables[table] = append(m.tables[table], record)
}

// Find finds records
func (m *MockDB) Find(table string, filter func(map[string]interface{}) bool) []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []map[string]interface{}
	if records, exists := m.tables[table]; exists {
		for _, record := range records {
			if filter == nil || filter(record) {
				results = append(results, record)
			}
		}
	}
	return results
}

// FindOne finds one record
func (m *MockDB) FindOne(table string, filter func(map[string]interface{}) bool) map[string]interface{} {
	results := m.Find(table, filter)
	if len(results) > 0 {
		return results[0]
	}
	return nil
}

// Delete deletes records
func (m *MockDB) Delete(table string, filter func(map[string]interface{}) bool) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if records, exists := m.tables[table]; exists {
		var newRecords []map[string]interface{}
		deleted := 0
		for _, record := range records {
			if filter == nil || !filter(record) {
				newRecords = append(newRecords, record)
			} else {
				deleted++
			}
		}
		m.tables[table] = newRecords
		return deleted
	}
	return 0
}

// Clear clears a table
func (m *MockDB) Clear(table string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tables, table)
}

// ClearAll clears all tables
func (m *MockDB) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tables = make(map[string][]map[string]interface{})
}

// MockService is a mock service for testing
type MockService struct {
	calls []Call
	mu    sync.RWMutex
}

// Call represents a service call
type Call struct {
	Method string
	Args   []interface{}
	Result interface{}
	Error  error
}

// NewMockService creates a new mock service
func NewMockService() *MockService {
	return &MockService{
		calls: make([]Call, 0),
	}
}

// Call records a call
func (m *MockService) Call(method string, args ...interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := Call{
		Method: method,
		Args:   args,
	}
	m.calls = append(m.calls, call)

	// Return the last recorded result if available
	if len(m.calls) > 0 {
		lastCall := m.calls[len(m.calls)-1]
		return lastCall.Result, lastCall.Error
	}

	return nil, nil
}

// GetCalls returns all calls
func (m *MockService) GetCalls() []Call {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Call(nil), m.calls...)
}

// GetCall gets a specific call
func (m *MockService) GetCall(index int) *Call {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if index >= 0 && index < len(m.calls) {
		return &m.calls[index]
	}
	return nil
}

// Clear clears all calls
func (m *MockService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]Call, 0)
}

// MockValidator is a mock validator for testing
type MockValidator struct {
	errors map[string]error
	mu     sync.RWMutex
}

// NewMockValidator creates a new mock validator
func NewMockValidator() *MockValidator {
	return &MockValidator{
		errors: make(map[string]error),
	}
}

// SetError sets an error for a key
func (m *MockValidator) SetError(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[key] = err
}

// Validate validates a value
func (m *MockValidator) Validate(key string, value interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err, exists := m.errors[key]; exists {
		return err
	}
	return nil
}

// Clear clears all errors
func (m *MockValidator) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = make(map[string]error)
}

// TestRunner runs tests with setup and teardown
type TestRunner struct {
	setup    func(*TestContext) error
	teardown func(*TestContext) error
	tests    map[string]func(*TestContext)
	mu       sync.Mutex
}

// NewTestRunner creates a new test runner
func NewTestRunner() *TestRunner {
	return &TestRunner{
		tests: make(map[string]func(*TestContext)),
	}
}

// Setup sets up the test runner
func (tr *TestRunner) Setup(fn func(*TestContext) error) *TestRunner {
	tr.setup = fn
	return tr
}

// Teardown sets up the teardown
func (tr *TestRunner) Teardown(fn func(*TestContext) error) *TestRunner {
	tr.teardown = fn
	return tr
}

// Add adds a test
func (tr *TestRunner) Add(name string, fn func(*TestContext)) *TestRunner {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.tests[name] = fn
	return tr
}

// Run runs all tests
func (tr *TestRunner) Run(t *testing.T) {
	for name, test := range tr.tests {
		t.Run(name, func(t *testing.T) {
			tc := NewTestContext(t)
			defer tc.Cleanup()

			if tr.setup != nil {
				if err := tr.setup(tc); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			if tr.teardown != nil {
				defer func() {
					if err := tr.teardown(tc); err != nil {
						t.Errorf("Teardown failed: %v", err)
					}
				}()
			}

			test(tc)
		})
	}
}

// BenchmarkRunner runs benchmarks with setup and teardown
type BenchmarkRunner struct {
	setup      func(*TestContext) error
	teardown   func(*TestContext) error
	benchmarks map[string]func(*TestContext, *testing.B)
	mu         sync.Mutex
}

// NewBenchmarkRunner creates a new benchmark runner
func NewBenchmarkRunner() *BenchmarkRunner {
	return &BenchmarkRunner{
		benchmarks: make(map[string]func(*TestContext, *testing.B)),
	}
}

// Setup sets up the benchmark runner
func (br *BenchmarkRunner) Setup(fn func(*TestContext) error) *BenchmarkRunner {
	br.setup = fn
	return br
}

// Teardown sets up the teardown
func (br *BenchmarkRunner) Teardown(fn func(*TestContext) error) *BenchmarkRunner {
	br.teardown = fn
	return br
}

// Add adds a benchmark
func (br *BenchmarkRunner) Add(name string, fn func(*TestContext, *testing.B)) *BenchmarkRunner {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.benchmarks[name] = fn
	return br
}

// Run runs all benchmarks
func (br *BenchmarkRunner) Run(b *testing.B) {
	for name, benchmark := range br.benchmarks {
		b.Run(name, func(b *testing.B) {
			tc := NewTestContext(&testing.T{})
			defer tc.Cleanup()

			if br.setup != nil {
				if err := br.setup(tc); err != nil {
					b.Fatalf("Setup failed: %v", err)
				}
			}

			if br.teardown != nil {
				defer func() {
					if err := br.teardown(tc); err != nil {
						b.Errorf("Teardown failed: %v", err)
					}
				}()
			}

			b.ResetTimer()
			benchmark(tc, b)
		})
	}
}

// TestHelper provides common test helpers
type TestHelper struct{}

// NewTestHelper creates a new test helper
func NewTestHelper() *TestHelper {
	return &TestHelper{}
}

// TempFile creates a temporary file
func (h *TestHelper) TempFile(content string) (*os.File, func()) {
	file, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		panic(err)
	}

	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			file.Close()
			panic(err)
		}
		file.Seek(0, 0)
	}

	cleanup := func() {
		file.Close()
		os.Remove(file.Name())
	}

	return file, cleanup
}

// TempDir creates a temporary directory
func (h *TestHelper) TempDir() (string, func()) {
	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

// CaptureOutput captures stdout/stderr
func (h *TestHelper) CaptureOutput(fn func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return string(outBytes), string(errBytes)
}

// Retry retries a function until it succeeds or max attempts reached
func (h *TestHelper) Retry(fn func() error, maxAttempts int, delay time.Duration) error {
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(delay)
			}
		}
	}
	return lastErr
}

// Eventually waits for a condition to be true
func (h *TestHelper) Eventually(fn func() bool, timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("condition not met within %v", timeout)
}

// JSONEquals compares two JSON strings
func (h *TestHelper) JSONEquals(a, b string) bool {
	var aJSON, bJSON interface{}
	if err := json.Unmarshal([]byte(a), &aJSON); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bJSON); err != nil {
		return false
	}
	return reflect.DeepEqual(aJSON, bJSON)
}

// Diff returns the difference between two JSON strings
func (h *TestHelper) Diff(a, b string) string {
	var aJSON, bJSON interface{}
	json.Unmarshal([]byte(a), &aJSON)
	json.Unmarshal([]byte(b), &bJSON)

	aBytes, _ := json.MarshalIndent(aJSON, "", "  ")
	bBytes, _ := json.MarshalIndent(bJSON, "", "  ")

	return fmt.Sprintf("Expected:\n%s\n\nActual:\n%s", string(aBytes), string(bBytes))
}

// MockHTTPHandler is a mock HTTP handler for testing
type MockHTTPHandler struct {
	mu       sync.Mutex
	requests []*http.Request
	response http.Response
}

// NewMockHTTPHandler creates a new mock HTTP handler
func NewMockHTTPHandler() *MockHTTPHandler {
	return &MockHTTPHandler{
		requests: make([]*http.Request, 0),
		response: http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
		},
	}
}

// ServeHTTP implements http.Handler
func (m *MockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store request
	reqCopy := r.Clone(context.Background())
	m.requests = append(m.requests, reqCopy)

	// Write response
	for k, v := range m.response.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(m.response.StatusCode)
	if m.response.Body != nil {
		io.Copy(w, m.response.Body)
	}
}

// GetRequests returns all requests
func (m *MockHTTPHandler) GetRequests() []*http.Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*http.Request(nil), m.requests...)
}

// SetResponse sets the response
func (m *MockHTTPHandler) SetResponse(status int, body string, headers map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.response.StatusCode = status
	m.response.Body = io.NopCloser(strings.NewReader(body))
	m.response.Header = make(http.Header)
	for k, v := range headers {
		m.response.Header.Set(k, v)
	}
}

// Clear clears all requests
func (m *MockHTTPHandler) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = make([]*http.Request, 0)
}

// TestSuite is a collection of tests
type TestSuite struct {
	name  string
	tests map[string]func(*testing.T)
	mu    sync.Mutex
}

// NewTestSuite creates a new test suite
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		name:  name,
		tests: make(map[string]func(*testing.T)),
	}
}

// Add adds a test to the suite
func (ts *TestSuite) Add(name string, fn func(*testing.T)) *TestSuite {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tests[name] = fn
	return ts
}

// Run runs all tests in the suite
func (ts *TestSuite) Run(t *testing.T) {
	for name, test := range ts.tests {
		t.Run(fmt.Sprintf("%s/%s", ts.name, name), test)
	}
}

// TestReporter is a custom test reporter
type TestReporter struct {
	mu      sync.Mutex
	results []TestResult
}

// TestResult represents a test result
type TestResult struct {
	Name     string
	Pass     bool
	Error    string
	Duration time.Duration
}

// NewTestReporter creates a new test reporter
func NewTestReporter() *TestReporter {
	return &TestReporter{
		results: make([]TestResult, 0),
	}
}

// Report records a test result
func (tr *TestReporter) Report(name string, pass bool, err error, duration time.Duration) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	result := TestResult{
		Name:     name,
		Pass:     pass,
		Duration: duration,
	}
	if err != nil {
		result.Error = err.Error()
	}
	tr.results = append(tr.results, result)
}

// GetResults returns all results
func (tr *TestReporter) GetResults() []TestResult {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return append([]TestResult(nil), tr.results...)
}

// Summary returns a summary
func (tr *TestReporter) Summary() (passed, failed int) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for _, result := range tr.results {
		if result.Pass {
			passed++
		} else {
			failed++
		}
	}
	return
}

// Clear clears all results
func (tr *TestReporter) Clear() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.results = make([]TestResult, 0)
}

// TestConfig holds test configuration
type TestConfig struct {
	Timeout    time.Duration
	RetryCount int
	RetryDelay time.Duration
	Verbose    bool
}

// DefaultTestConfig creates a default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
		Verbose:    false,
	}
}

// TestRunnerV2 runs tests with configuration
type TestRunnerV2 struct {
	config TestConfig
	tests  map[string]func(*TestContext) error
	mu     sync.Mutex
}

// NewTestRunnerV2 creates a new test runner v2
func NewTestRunnerV2(config TestConfig) *TestRunnerV2 {
	return &TestRunnerV2{
		config: config,
		tests:  make(map[string]func(*TestContext) error),
	}
}

// Add adds a test
func (tr *TestRunnerV2) Add(name string, fn func(*TestContext) error) *TestRunnerV2 {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.tests[name] = fn
	return tr
}

// Run runs all tests
func (tr *TestRunnerV2) Run(t *testing.T) {
	for name, test := range tr.tests {
		t.Run(name, func(t *testing.T) {
			tc := NewTestContext(t)
			defer tc.Cleanup()

			// Retry logic
			var lastErr error
			for attempt := 0; attempt <= tr.config.RetryCount; attempt++ {
				err := test(tc)
				if err == nil {
					return
				}
				lastErr = err
				if attempt < tr.config.RetryCount {
					time.Sleep(tr.config.RetryDelay)
				}
			}

			t.Fatalf("Test failed after %d attempts: %v", tr.config.RetryCount+1, lastErr)
		})
	}
}

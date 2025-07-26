// Package testutils provides comprehensive testing utilities for the CCE project
package testutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/stretchr/testify/require"
)

// TestEnvironment provides isolated test environment setup
type TestEnvironment struct {
	TempDir      string
	ConfigDir    string
	ConfigPath   string
	BackupPath   string
	OriginalHome string
	MockServer   *httptest.Server
	cleanup      func()
	t            *testing.T
}

// SetupTestEnvironment creates a comprehensive test environment
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	require.NoError(t, err)

	// Set up config directory structure
	configDir := filepath.Join(tempDir, ".claude-code-env")
	configPath := filepath.Join(configDir, "config.json")
	backupPath := configPath + ".backup"

	require.NoError(t, os.MkdirAll(configDir, 0700))

	// Override HOME for isolated testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	// Create mock HTTP server
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/v1/auth":
			auth := r.Header.Get("Authorization")
			if auth == "Bearer valid-api-key" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"user": "test"})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid key"})
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	testEnv := &TestEnvironment{
		TempDir:      tempDir,
		ConfigDir:    configDir,
		ConfigPath:   configPath,
		BackupPath:   backupPath,
		OriginalHome: originalHome,
		MockServer:   mockServer,
		t:            t,
		cleanup: func() {
			mockServer.Close()
			os.Setenv("HOME", originalHome)
			os.RemoveAll(tempDir)
		},
	}

	return testEnv
}

// Cleanup removes the test environment
func (te *TestEnvironment) Cleanup() {
	if te.cleanup != nil {
		te.cleanup()
	}
}

// CreateConfigFile creates a test configuration file
func (te *TestEnvironment) CreateConfigFile(config *types.Config) {
	data, err := json.MarshalIndent(config, "", "  ")
	require.NoError(te.t, err)
	require.NoError(te.t, os.WriteFile(te.ConfigPath, data, 0600))
}

// CreateCorruptedConfigFile creates a corrupted JSON config file
func (te *TestEnvironment) CreateCorruptedConfigFile() {
	corruptedJSON := `{"version": "1.0.0", "environments": {`
	require.NoError(te.t, os.WriteFile(te.ConfigPath, []byte(corruptedJSON), 0600))
}

// CreateReadOnlyConfigDir creates a read-only config directory for permission tests
func (te *TestEnvironment) CreateReadOnlyConfigDir() {
	require.NoError(te.t, os.Chmod(te.ConfigDir, 0500))
}

// RestoreConfigDirPermissions restores normal permissions to config directory
func (te *TestEnvironment) RestoreConfigDirPermissions() {
	require.NoError(te.t, os.Chmod(te.ConfigDir, 0700))
}

// GetMockServerURL returns the mock server URL for testing
func (te *TestEnvironment) GetMockServerURL() string {
	return te.MockServer.URL
}

// MockHTTPServer provides a configurable HTTP server for testing
type MockHTTPServer struct {
	server    *httptest.Server
	responses map[string]MockResponse
	requests  []MockRequest
}

// MockResponse defines a mock HTTP response
type MockResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Delay      time.Duration
}

// MockRequest records details of received requests
type MockRequest struct {
	Method    string
	URL       string
	Headers   map[string]string
	Body      string
	Timestamp time.Time
}

// NewMockHTTPServer creates a new mock HTTP server
func NewMockHTTPServer() *MockHTTPServer {
	m := &MockHTTPServer{
		responses: make(map[string]MockResponse),
		requests:  make([]MockRequest, 0),
	}

	m.server = httptest.NewTLSServer(http.HandlerFunc(m.handler))
	return m
}

// AddResponse adds a mock response for a specific path
func (m *MockHTTPServer) AddResponse(path string, response MockResponse) {
	m.responses[path] = response
}

// GetRequests returns all recorded requests
func (m *MockHTTPServer) GetRequests() []MockRequest {
	return m.requests
}

// URL returns the server URL
func (m *MockHTTPServer) URL() string {
	return m.server.URL
}

// Close shuts down the mock server
func (m *MockHTTPServer) Close() {
	m.server.Close()
}

// handler handles HTTP requests to the mock server
func (m *MockHTTPServer) handler(w http.ResponseWriter, r *http.Request) {
	// Record the request
	body, _ := io.ReadAll(r.Body)
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	m.requests = append(m.requests, MockRequest{
		Method:    r.Method,
		URL:       r.URL.Path,
		Headers:   headers,
		Body:      string(body),
		Timestamp: time.Now(),
	})

	// Look for matching response
	response, exists := m.responses[r.URL.Path]
	if !exists {
		response = MockResponse{StatusCode: 404, Body: "Not Found"}
	}

	// Apply delay if specified
	if response.Delay > 0 {
		time.Sleep(response.Delay)
	}

	// Set headers
	for k, v := range response.Headers {
		w.Header().Set(k, v)
	}

	// Write response
	w.WriteHeader(response.StatusCode)
	w.Write([]byte(response.Body))
}

// FileSystemHelper provides file system testing utilities
type FileSystemHelper struct {
	TempDir string
	t       *testing.T
}

// NewFileSystemHelper creates a new file system helper
func NewFileSystemHelper(t *testing.T) *FileSystemHelper {
	tempDir, err := os.MkdirTemp("", "cce-fs-test-*")
	require.NoError(t, err)

	return &FileSystemHelper{
		TempDir: tempDir,
		t:       t,
	}
}

// Cleanup removes temporary files
func (fsh *FileSystemHelper) Cleanup() {
	os.RemoveAll(fsh.TempDir)
}

// CreateFile creates a file with specific content and permissions
func (fsh *FileSystemHelper) CreateFile(path string, content []byte, mode os.FileMode) {
	fullPath := filepath.Join(fsh.TempDir, path)
	dir := filepath.Dir(fullPath)
	require.NoError(fsh.t, os.MkdirAll(dir, 0755))
	require.NoError(fsh.t, os.WriteFile(fullPath, content, mode))
}

// CreateDirectory creates a directory with specific permissions
func (fsh *FileSystemHelper) CreateDirectory(path string, mode os.FileMode) {
	fullPath := filepath.Join(fsh.TempDir, path)
	require.NoError(fsh.t, os.MkdirAll(fullPath, mode))
}

// GetPath returns the full path for a relative path
func (fsh *FileSystemHelper) GetPath(path string) string {
	return filepath.Join(fsh.TempDir, path)
}

// FileExists checks if a file exists
func (fsh *FileSystemHelper) FileExists(path string) bool {
	_, err := os.Stat(filepath.Join(fsh.TempDir, path))
	return err == nil
}

// GetFileMode returns the file mode for a file
func (fsh *FileSystemHelper) GetFileMode(path string) os.FileMode {
	info, err := os.Stat(filepath.Join(fsh.TempDir, path))
	require.NoError(fsh.t, err)
	return info.Mode()
}

// ProcessHelper provides process and command testing utilities
type ProcessHelper struct {
	ExecutablePath string
	TempDir        string
	t              *testing.T
}

// NewProcessHelper creates a new process helper
func NewProcessHelper(t *testing.T) *ProcessHelper {
	tempDir, err := os.MkdirTemp("", "cce-process-test-*")
	require.NoError(t, err)

	// Create a mock executable
	execPath := filepath.Join(tempDir, "mock-claude-code")
	mockScript := `#!/bin/bash
echo "Mock Claude Code v1.0.0"
echo "Arguments: $@"
echo "Environment variables:"
env | grep ANTHROPIC_ || true
exit 0`

	require.NoError(t, os.WriteFile(execPath, []byte(mockScript), 0755))

	return &ProcessHelper{
		ExecutablePath: execPath,
		TempDir:        tempDir,
		t:              t,
	}
}

// Cleanup removes temporary files
func (ph *ProcessHelper) Cleanup() {
	os.RemoveAll(ph.TempDir)
}

// CreateFailingExecutable creates an executable that always fails
func (ph *ProcessHelper) CreateFailingExecutable() string {
	execPath := filepath.Join(ph.TempDir, "failing-claude-code")
	failScript := `#!/bin/bash
echo "Mock failure" >&2
exit 1`

	require.NoError(ph.t, os.WriteFile(execPath, []byte(failScript), 0755))
	return execPath
}

// CreateSlowExecutable creates an executable that takes time to complete
func (ph *ProcessHelper) CreateSlowExecutable(delay time.Duration) string {
	execPath := filepath.Join(ph.TempDir, "slow-claude-code")
	slowScript := fmt.Sprintf(`#!/bin/bash
sleep %.0f
echo "Slow execution complete"
exit 0`, delay.Seconds())

	require.NoError(ph.t, os.WriteFile(execPath, []byte(slowScript), 0755))
	return execPath
}

// PerformanceHelper provides performance testing utilities
type PerformanceHelper struct {
	measurements []PerformanceMeasurement
}

// PerformanceMeasurement records a performance measurement
type PerformanceMeasurement struct {
	Name      string
	Duration  time.Duration
	MemoryMB  float64
	Timestamp time.Time
}

// NewPerformanceHelper creates a new performance helper
func NewPerformanceHelper() *PerformanceHelper {
	return &PerformanceHelper{
		measurements: make([]PerformanceMeasurement, 0),
	}
}

// MeasureOperation measures the performance of an operation
func (ph *PerformanceHelper) MeasureOperation(name string, operation func()) {
	start := time.Now()
	operation()
	duration := time.Since(start)

	ph.measurements = append(ph.measurements, PerformanceMeasurement{
		Name:      name,
		Duration:  duration,
		Timestamp: start,
	})
}

// GetMeasurements returns all recorded measurements
func (ph *PerformanceHelper) GetMeasurements() []PerformanceMeasurement {
	return ph.measurements
}

// GetAverageDuration returns the average duration for operations with the given name
func (ph *PerformanceHelper) GetAverageDuration(name string) time.Duration {
	var total time.Duration
	var count int

	for _, m := range ph.measurements {
		if m.Name == name {
			total += m.Duration
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// SecurityTestHelper provides security testing utilities
type SecurityTestHelper struct {
	t *testing.T
}

// NewSecurityTestHelper creates a new security test helper
func NewSecurityTestHelper(t *testing.T) *SecurityTestHelper {
	return &SecurityTestHelper{t: t}
}

// ValidateFilePermissions ensures file has secure permissions
func (sth *SecurityTestHelper) ValidateFilePermissions(path string, expectedMode os.FileMode) {
	info, err := os.Stat(path)
	require.NoError(sth.t, err)

	actualMode := info.Mode().Perm()
	require.Equal(sth.t, expectedMode, actualMode,
		"File %s has permissions %o, expected %o", path, actualMode, expectedMode)
}

// ValidateNoSensitiveDataInLogs ensures no sensitive data appears in output
func (sth *SecurityTestHelper) ValidateNoSensitiveDataInLogs(output string, sensitiveData []string) {
	for _, data := range sensitiveData {
		require.NotContains(sth.t, output, data,
			"Sensitive data '%s' found in output", data)
	}
}

// ValidateAPIKeyMasking ensures API keys are properly masked
func (sth *SecurityTestHelper) ValidateAPIKeyMasking(maskedKey, originalKey string) {
	require.NotEqual(sth.t, originalKey, maskedKey, "API key not masked")
	require.Contains(sth.t, maskedKey, "***", "API key mask format incorrect")

	if len(originalKey) >= 4 {
		expectedSuffix := originalKey[len(originalKey)-4:]
		require.Contains(sth.t, maskedKey, expectedSuffix,
			"API key mask should show last 4 characters")
	}
}

// ConcurrencyTestHelper provides concurrency testing utilities
type ConcurrencyTestHelper struct {
	t *testing.T
}

// NewConcurrencyTestHelper creates a new concurrency test helper
func NewConcurrencyTestHelper(t *testing.T) *ConcurrencyTestHelper {
	return &ConcurrencyTestHelper{t: t}
}

// RunConcurrentOperations runs multiple operations concurrently and waits for completion
func (cth *ConcurrencyTestHelper) RunConcurrentOperations(operations []func() error, maxGoroutines int) []error {
	resultChan := make(chan error, len(operations))
	semaphore := make(chan struct{}, maxGoroutines)

	// Start operations
	for _, op := range operations {
		go func(operation func() error) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			resultChan <- operation()
		}(op)
	}

	// Collect results
	results := make([]error, 0, len(operations))
	for i := 0; i < len(operations); i++ {
		results = append(results, <-resultChan)
	}

	return results
}

// TestDataGenerator provides test data generation utilities
type TestDataGenerator struct{}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{}
}

// GenerateEnvironment creates a test environment with specified parameters
func (tdg *TestDataGenerator) GenerateEnvironment(name string, baseURL string) *types.Environment {
	return &types.Environment{
		Name:        name,
		Description: fmt.Sprintf("Generated test environment: %s", name),
		BaseURL:     baseURL,
		APIKey:      fmt.Sprintf("test-api-key-%s-12345", name),
		Headers: map[string]string{
			"X-Test-Environment": name,
			"X-Generated":        "true",
		},
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
		NetworkInfo: &types.NetworkInfo{
			Status:       "unchecked",
			ResponseTime: 0,
			SSLValid:     false,
		},
	}
}

// GenerateConfig creates a test configuration with specified environments
func (tdg *TestDataGenerator) GenerateConfig(environmentNames []string) *types.Config {
	environments := make(map[string]types.Environment)

	for _, name := range environmentNames {
		env := tdg.GenerateEnvironment(name, fmt.Sprintf("https://%s.api.test.com/v1", name))
		environments[name] = *env
	}

	config := &types.Config{
		Version:      "1.0.0",
		Environments: environments,
		CreatedAt:    time.Now().Add(-24 * time.Hour),
		UpdatedAt:    time.Now(),
	}

	// Set first environment as default
	if len(environmentNames) > 0 {
		config.DefaultEnv = environmentNames[0]
	}

	return config
}

// GenerateInvalidEnvironments creates environments with validation issues
func (tdg *TestDataGenerator) GenerateInvalidEnvironments() map[string]*types.Environment {
	return map[string]*types.Environment{
		"empty-name": {
			Name:    "", // Invalid: empty name
			BaseURL: "https://api.test.com/v1",
			APIKey:  "valid-key-12345",
		},
		"invalid-url": {
			Name:    "invalid-url",
			BaseURL: "not-a-url", // Invalid: malformed URL
			APIKey:  "valid-key-12345",
		},
		"short-api-key": {
			Name:    "short-key",
			BaseURL: "https://api.test.com/v1",
			APIKey:  "short", // Invalid: too short
		},
		"long-description": {
			Name:        "long-desc",
			Description: string(make([]byte, 250)), // Invalid: too long
			BaseURL:     "https://api.test.com/v1",
			APIKey:      "valid-key-12345",
		},
	}
}

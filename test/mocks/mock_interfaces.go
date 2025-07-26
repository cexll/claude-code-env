// Package mocks provides mock implementations for testing
package mocks

import (
	"time"
	"github.com/claude-code/env-switcher/pkg/types"
)

// MockConfigManager provides a mock implementation of ConfigManager for testing
type MockConfigManager struct {
	LoadFunc                        func() (*types.Config, error)
	SaveFunc                        func(*types.Config) error
	ValidateFunc                    func(*types.Config) error
	BackupFunc                      func() error
	GetConfigPathFunc               func() string
	ValidateNetworkConnectivityFunc func(*types.Environment) error
	
	// Test data storage
	StoredConfig *types.Config
	CallLog      []string
}

func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		CallLog: make([]string, 0),
	}
}

func (m *MockConfigManager) Load() (*types.Config, error) {
	m.CallLog = append(m.CallLog, "Load")
	if m.LoadFunc != nil {
		return m.LoadFunc()
	}
	if m.StoredConfig != nil {
		return m.StoredConfig, nil
	}
	return &types.Config{
		Version:      "1.0.0",
		Environments: make(map[string]types.Environment),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *MockConfigManager) Save(config *types.Config) error {
	m.CallLog = append(m.CallLog, "Save")
	if m.SaveFunc != nil {
		return m.SaveFunc(config)
	}
	m.StoredConfig = config
	return nil
}

func (m *MockConfigManager) Validate(config *types.Config) error {
	m.CallLog = append(m.CallLog, "Validate")
	if m.ValidateFunc != nil {
		return m.ValidateFunc(config)
	}
	return nil
}

func (m *MockConfigManager) Backup() error {
	m.CallLog = append(m.CallLog, "Backup")
	if m.BackupFunc != nil {
		return m.BackupFunc()
	}
	return nil
}

func (m *MockConfigManager) GetConfigPath() string {
	m.CallLog = append(m.CallLog, "GetConfigPath")
	if m.GetConfigPathFunc != nil {
		return m.GetConfigPathFunc()
	}
	return "/tmp/test-config.json"
}

func (m *MockConfigManager) ValidateNetworkConnectivity(env *types.Environment) error {
	m.CallLog = append(m.CallLog, "ValidateNetworkConnectivity")
	if m.ValidateNetworkConnectivityFunc != nil {
		return m.ValidateNetworkConnectivityFunc(env)
	}
	return nil
}

// MockNetworkValidator provides a mock implementation of NetworkValidator for testing
type MockNetworkValidator struct {
	ValidateEndpointFunc            func(string) (*types.NetworkValidationResult, error)
	ValidateEndpointWithTimeoutFunc func(string, time.Duration) (*types.NetworkValidationResult, error)
	TestAPIConnectivityFunc         func(*types.Environment) error
	ClearCacheFunc                  func()
	
	CallLog []string
	Results map[string]*types.NetworkValidationResult
}

func NewMockNetworkValidator() *MockNetworkValidator {
	return &MockNetworkValidator{
		CallLog: make([]string, 0),
		Results: make(map[string]*types.NetworkValidationResult),
	}
}

func (m *MockNetworkValidator) ValidateEndpoint(url string) (*types.NetworkValidationResult, error) {
	m.CallLog = append(m.CallLog, "ValidateEndpoint:"+url)
	if m.ValidateEndpointFunc != nil {
		return m.ValidateEndpointFunc(url)
	}
	if result, exists := m.Results[url]; exists {
		return result, nil
	}
	return &types.NetworkValidationResult{
		Success:      true,
		ResponseTime: 100 * time.Millisecond,
		StatusCode:   200,
		SSLValid:     true,
		Timestamp:    time.Now(),
	}, nil
}

func (m *MockNetworkValidator) ValidateEndpointWithTimeout(url string, timeout time.Duration) (*types.NetworkValidationResult, error) {
	m.CallLog = append(m.CallLog, "ValidateEndpointWithTimeout:"+url)
	if m.ValidateEndpointWithTimeoutFunc != nil {
		return m.ValidateEndpointWithTimeoutFunc(url, timeout)
	}
	return m.ValidateEndpoint(url)
}

func (m *MockNetworkValidator) TestAPIConnectivity(env *types.Environment) error {
	m.CallLog = append(m.CallLog, "TestAPIConnectivity:"+env.Name)
	if m.TestAPIConnectivityFunc != nil {
		return m.TestAPIConnectivityFunc(env)
	}
	return nil
}

func (m *MockNetworkValidator) ClearCache() {
	m.CallLog = append(m.CallLog, "ClearCache")
	if m.ClearCacheFunc != nil {
		m.ClearCacheFunc()
	}
	m.Results = make(map[string]*types.NetworkValidationResult)
}

// MockInteractiveUI provides a mock implementation of InteractiveUI for testing
type MockInteractiveUI struct {
	SelectFunc     func(string, []types.SelectItem) (int, string, error)
	PromptFunc     func(string, func(string) error) (string, error)
	PromptPasswordFunc func(string, func(string) error) (string, error)
	ConfirmFunc    func(string) (bool, error)
	MultiInputFunc func([]types.InputField) (map[string]string, error)
	
	CallLog     []string
	Responses   map[string]interface{}
	CallCount   int
}

func NewMockInteractiveUI() *MockInteractiveUI {
	return &MockInteractiveUI{
		CallLog:   make([]string, 0),
		Responses: make(map[string]interface{}),
	}
}

func (m *MockInteractiveUI) Select(label string, items []types.SelectItem) (int, string, error) {
	m.CallLog = append(m.CallLog, "Select:"+label)
	m.CallCount++
	if m.SelectFunc != nil {
		return m.SelectFunc(label, items)
	}
	if response, exists := m.Responses["select"]; exists {
		if result, ok := response.([]interface{}); ok && len(result) >= 3 {
			return result[0].(int), result[1].(string), nil
		}
	}
	return 0, items[0].Label, nil
}

func (m *MockInteractiveUI) Prompt(label string, validate func(string) error) (string, error) {
	m.CallLog = append(m.CallLog, "Prompt:"+label)
	m.CallCount++
	if m.PromptFunc != nil {
		return m.PromptFunc(label, validate)
	}
	if response, exists := m.Responses["prompt"]; exists {
		return response.(string), nil
	}
	return "mock-input", nil
}

func (m *MockInteractiveUI) PromptPassword(label string, validate func(string) error) (string, error) {
	m.CallLog = append(m.CallLog, "PromptPassword:"+label)
	m.CallCount++
	if m.PromptPasswordFunc != nil {
		return m.PromptPasswordFunc(label, validate)
	}
	if response, exists := m.Responses["password"]; exists {
		return response.(string), nil
	}
	return "mock-password", nil
}

func (m *MockInteractiveUI) Confirm(label string) (bool, error) {
	m.CallLog = append(m.CallLog, "Confirm:"+label)
	m.CallCount++
	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(label)
	}
	if response, exists := m.Responses["confirm"]; exists {
		return response.(bool), nil
	}
	return true, nil
}

func (m *MockInteractiveUI) MultiInput(fields []types.InputField) (map[string]string, error) {
	m.CallLog = append(m.CallLog, "MultiInput")
	m.CallCount++
	if m.MultiInputFunc != nil {
		return m.MultiInputFunc(fields)
	}
	if response, exists := m.Responses["multiinput"]; exists {
		return response.(map[string]string), nil
	}
	
	// Default mock responses
	results := make(map[string]string)
	for _, field := range fields {
		switch field.Name {
		case "description":
			results[field.Name] = "Mock description"
		case "base_url":
			results[field.Name] = "https://mock.api.com/v1"
		case "api_key":
			results[field.Name] = "mock-api-key-12345"
		default:
			results[field.Name] = "mock-" + field.Name
		}
	}
	return results, nil
}

// MockClaudeCodeLauncher provides a mock implementation of ClaudeCodeLauncher for testing
type MockClaudeCodeLauncher struct {
	LaunchFunc           func(*types.Environment, []string) error
	ValidateClaudeCodeFunc func() error
	GetClaudeCodePathFunc  func() (string, error)
	
	CallLog       []string
	LaunchCalls   []LaunchCall
	ClaudeCodePath string
}

type LaunchCall struct {
	Environment *types.Environment
	Args        []string
	Timestamp   time.Time
}

func NewMockClaudeCodeLauncher() *MockClaudeCodeLauncher {
	return &MockClaudeCodeLauncher{
		CallLog:        make([]string, 0),
		LaunchCalls:    make([]LaunchCall, 0),
		ClaudeCodePath: "/usr/local/bin/claude-code",
	}
}

func (m *MockClaudeCodeLauncher) Launch(env *types.Environment, args []string) error {
	m.CallLog = append(m.CallLog, "Launch")
	m.LaunchCalls = append(m.LaunchCalls, LaunchCall{
		Environment: env,
		Args:        args,
		Timestamp:   time.Now(),
	})
	if m.LaunchFunc != nil {
		return m.LaunchFunc(env, args)
	}
	return nil
}

func (m *MockClaudeCodeLauncher) ValidateClaudeCode() error {
	m.CallLog = append(m.CallLog, "ValidateClaudeCode")
	if m.ValidateClaudeCodeFunc != nil {
		return m.ValidateClaudeCodeFunc()
	}
	return nil
}

func (m *MockClaudeCodeLauncher) GetClaudeCodePath() (string, error) {
	m.CallLog = append(m.CallLog, "GetClaudeCodePath")
	if m.GetClaudeCodePathFunc != nil {
		return m.GetClaudeCodePathFunc()
	}
	return m.ClaudeCodePath, nil
}

// TestHelper provides common test utilities
type TestHelper struct {
	TempDir string
}

func NewTestHelper() *TestHelper {
	return &TestHelper{}
}

func (h *TestHelper) CreateTestEnvironment() *types.Environment {
	return &types.Environment{
		Name:        "test-env",
		Description: "Test environment for unit testing",
		BaseURL:     "https://api.test.com/v1",
		APIKey:      "test-api-key-12345",
		Headers:     map[string]string{"X-Test": "true"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		NetworkInfo: &types.NetworkInfo{
			Status:       "connected",
			ResponseTime: 150,
			SSLValid:     true,
		},
	}
}

func (h *TestHelper) CreateTestConfig() *types.Config {
	env := h.CreateTestEnvironment()
	return &types.Config{
		Version:    "1.0.0",
		DefaultEnv: "test-env",
		Environments: map[string]types.Environment{
			"test-env": *env,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (h *TestHelper) CreateTestError(errorType interface{}, message string) error {
	switch errorType {
	case "config":
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Message: message,
		}
	case "network":
		return &types.NetworkError{
			Type:    types.NetworkConnectionFailed,
			Message: message,
		}
	case "launcher":
		return &types.LauncherError{
			Type:    types.ClaudeCodeNotFound,
			Message: message,
		}
	case "environment":
		return &types.EnvironmentError{
			Type:    types.EnvironmentNotFound,
			Message: message,
		}
	default:
		return &types.ConfigError{Type: types.ConfigValidationFailed, Message: message}
	}
}
// Package performance provides comprehensive performance tests for the CCE application
package performance

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/internal/launcher"
	"github.com/cexll/claude-code-env/internal/network"
	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/mocks"
	"github.com/cexll/claude-code-env/test/testutils"
)

// Performance thresholds (adjust based on requirements)
const (
	ConfigSaveThreshold      = 50 * time.Millisecond
	ConfigLoadThreshold      = 20 * time.Millisecond
	ConfigValidateThreshold  = 5 * time.Millisecond
	NetworkValidateThreshold = 100 * time.Millisecond
	LaunchThreshold          = 1000 * time.Millisecond
)

func TestConfigManager_SavePerformance(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	perfHelper := testutils.NewPerformanceHelper()

	// Test with different config sizes
	testCases := []struct {
		name       string
		envCount   int
		iterations int
	}{
		{"small_config", 1, 100},
		{"medium_config", 10, 50},
		{"large_config", 50, 10},
		{"xlarge_config", 100, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate config with specified number of environments
			generator := testutils.NewTestDataGenerator()
			envNames := make([]string, tc.envCount)
			for i := 0; i < tc.envCount; i++ {
				envNames[i] = generateEnvName(i)
			}
			testConfig := generator.GenerateConfig(envNames)

			// Measure save performance
			for i := 0; i < tc.iterations; i++ {
				perfHelper.MeasureOperation(tc.name+"_save", func() {
					manager.Save(testConfig)
				})
			}

			avgDuration := perfHelper.GetAverageDuration(tc.name + "_save")

			// Verify performance meets threshold
			assert.True(t, avgDuration < ConfigSaveThreshold*time.Duration(tc.envCount/10+1),
				"Save performance for %s should be under threshold: %v (actual: %v)",
				tc.name, ConfigSaveThreshold*time.Duration(tc.envCount/10+1), avgDuration)
		})
	}
}

func TestConfigManager_LoadPerformance(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	perfHelper := testutils.NewPerformanceHelper()

	// Create test configs of different sizes
	testCases := []struct {
		name       string
		envCount   int
		iterations int
	}{
		{"small_config", 1, 200},
		{"medium_config", 10, 100},
		{"large_config", 50, 20},
		{"xlarge_config", 100, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate and save test config
			generator := testutils.NewTestDataGenerator()
			envNames := make([]string, tc.envCount)
			for i := 0; i < tc.envCount; i++ {
				envNames[i] = generateEnvName(i)
			}
			testConfig := generator.GenerateConfig(envNames)
			manager.Save(testConfig)

			// Measure load performance
			for i := 0; i < tc.iterations; i++ {
				perfHelper.MeasureOperation(tc.name+"_load", func() {
					manager.Load()
				})
			}

			avgDuration := perfHelper.GetAverageDuration(tc.name + "_load")

			// Verify performance meets threshold
			assert.True(t, avgDuration < ConfigLoadThreshold*time.Duration(tc.envCount/20+1),
				"Load performance for %s should be under threshold: %v (actual: %v)",
				tc.name, ConfigLoadThreshold*time.Duration(tc.envCount/20+1), avgDuration)
		})
	}
}

func TestConfigManager_ValidationPerformance(t *testing.T) {
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	perfHelper := testutils.NewPerformanceHelper()

	// Test validation performance with different config sizes
	testCases := []struct {
		name       string
		envCount   int
		iterations int
	}{
		{"small_validation", 1, 1000},
		{"medium_validation", 10, 500},
		{"large_validation", 50, 100},
		{"xlarge_validation", 100, 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate test config
			generator := testutils.NewTestDataGenerator()
			envNames := make([]string, tc.envCount)
			for i := 0; i < tc.envCount; i++ {
				envNames[i] = generateEnvName(i)
			}
			testConfig := generator.GenerateConfig(envNames)

			// Measure validation performance
			for i := 0; i < tc.iterations; i++ {
				perfHelper.MeasureOperation(tc.name, func() {
					manager.Validate(testConfig)
				})
			}

			avgDuration := perfHelper.GetAverageDuration(tc.name)

			// Validation should scale linearly with environment count
			threshold := ConfigValidateThreshold * time.Duration(tc.envCount+1)
			assert.True(t, avgDuration < threshold,
				"Validation performance for %s should be under threshold: %v (actual: %v)",
				tc.name, threshold, avgDuration)
		})
	}
}

func TestNetworkValidator_ValidationPerformance(t *testing.T) {
	// Create mock server for testing
	mockServer := testutils.NewMockHTTPServer()
	defer mockServer.Close()

	mockServer.AddResponse("/", testutils.MockResponse{
		StatusCode: 200,
		Body:       `{"status": "ok"}`,
	})

	validator := network.NewValidator()
	perfHelper := testutils.NewPerformanceHelper()

	testCases := []struct {
		name       string
		iterations int
		concurrent bool
	}{
		{"sequential_validation", 100, false},
		{"concurrent_validation", 100, true},
		{"cached_validation", 100, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "cached_validation" {
				// Pre-populate cache
				validator.ValidateEndpoint(mockServer.URL())
			}

			if tc.concurrent {
				// Concurrent validation test
				var wg sync.WaitGroup
				startTime := time.Now()

				for i := 0; i < tc.iterations; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						validator.ValidateEndpoint(mockServer.URL())
					}()
				}

				wg.Wait()
				totalDuration := time.Since(startTime)
				avgDuration := totalDuration / time.Duration(tc.iterations)

				assert.True(t, avgDuration < NetworkValidateThreshold,
					"Concurrent validation average should be under threshold: %v (actual: %v)",
					NetworkValidateThreshold, avgDuration)
			} else {
				// Sequential validation test
				for i := 0; i < tc.iterations; i++ {
					perfHelper.MeasureOperation(tc.name, func() {
						validator.ValidateEndpoint(mockServer.URL())
					})
				}

				avgDuration := perfHelper.GetAverageDuration(tc.name)

				expectedThreshold := NetworkValidateThreshold
				if tc.name == "cached_validation" {
					expectedThreshold = NetworkValidateThreshold / 10 // Should be much faster when cached
				}

				assert.True(t, avgDuration < expectedThreshold,
					"Validation performance for %s should be under threshold: %v (actual: %v)",
					tc.name, expectedThreshold, avgDuration)
			}
		})
	}
}

func TestSystemLauncher_LaunchPerformance(t *testing.T) {
	processHelper := testutils.NewProcessHelper(t)
	defer processHelper.Cleanup()

	launcher := launcher.NewSystemLauncher()
	launcher.SetClaudeCodePath(processHelper.ExecutablePath)

	perfHelper := testutils.NewPerformanceHelper()

	testCases := []struct {
		name       string
		envSize    string // small, medium, large
		iterations int
	}{
		{"small_env_launch", "small", 20},
		{"medium_env_launch", "medium", 10},
		{"large_env_launch", "large", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create environment based on size
			env := createTestEnvironmentBySize(tc.envSize)

			// Measure launch performance
			for i := 0; i < tc.iterations; i++ {
				perfHelper.MeasureOperation(tc.name, func() {
					params := &types.LaunchParameters{
						Environment: env,
						Arguments:   []string{"--version"},
					}
					launcher.Launch(params)
				})
			}

			avgDuration := perfHelper.GetAverageDuration(tc.name)

			assert.True(t, avgDuration < LaunchThreshold,
				"Launch performance for %s should be under threshold: %v (actual: %v)",
				tc.name, LaunchThreshold, avgDuration)
		})
	}
}

func TestConcurrentOperations_ConfigManager(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	concHelper := testutils.NewConcurrencyTestHelper(t)
	_ = testutils.NewPerformanceHelper() // for potential future use

	// Test concurrent save operations
	t.Run("concurrent_saves", func(t *testing.T) {
		operations := make([]func() error, 20)

		for i := 0; i < 20; i++ {
			envName := generateEnvName(i)
			operations[i] = func() error {
				generator := testutils.NewTestDataGenerator()
				config := generator.GenerateConfig([]string{envName})
				return manager.Save(config)
			}
		}

		startTime := time.Now()
		results := concHelper.RunConcurrentOperations(operations, 5)
		totalDuration := time.Since(startTime)

		// Check that all operations succeeded
		for i, err := range results {
			assert.NoError(t, err, "Concurrent save operation %d should succeed", i)
		}

		// Performance should be reasonable
		avgDuration := totalDuration / time.Duration(len(operations))
		assert.True(t, avgDuration < ConfigSaveThreshold*2,
			"Concurrent save average should be reasonable: %v", avgDuration)
	})

	// Test concurrent load operations
	t.Run("concurrent_loads", func(t *testing.T) {
		// Set up a config to load
		helper := mocks.NewTestHelper()
		testConfig := helper.CreateTestConfig()
		manager.Save(testConfig)

		operations := make([]func() error, 50)

		for i := 0; i < 50; i++ {
			operations[i] = func() error {
				_, err := manager.Load()
				return err
			}
		}

		startTime := time.Now()
		results := concHelper.RunConcurrentOperations(operations, 10)
		totalDuration := time.Since(startTime)

		// Check that all operations succeeded
		for i, err := range results {
			assert.NoError(t, err, "Concurrent load operation %d should succeed", i)
		}

		// Performance should be reasonable
		avgDuration := totalDuration / time.Duration(len(operations))
		assert.True(t, avgDuration < ConfigLoadThreshold*2,
			"Concurrent load average should be reasonable: %v", avgDuration)
	})
}

func TestMemoryUsage_ConfigOperations(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Test memory usage with large configurations
	t.Run("large_config_memory", func(t *testing.T) {
		generator := testutils.NewTestDataGenerator()

		// Create a large configuration
		envNames := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			envNames[i] = generateEnvName(i)
		}

		largeConfig := generator.GenerateConfig(envNames)

		// Save and load multiple times to test memory stability
		for i := 0; i < 10; i++ {
			err := manager.Save(largeConfig)
			require.NoError(t, err)

			_, err = manager.Load()
			require.NoError(t, err)
		}

		// Test should complete without memory issues
		// In a real scenario, you might use runtime.ReadMemStats to check memory usage
	})
}

func TestNetworkValidator_CachePerformance(t *testing.T) {
	mockServer := testutils.NewMockHTTPServer()
	defer mockServer.Close()

	mockServer.AddResponse("/", testutils.MockResponse{
		StatusCode: 200,
		Body:       `{"status": "ok"}`,
	})

	validator := network.NewValidator()
	perfHelper := testutils.NewPerformanceHelper()

	// Test cache performance with different cache sizes
	testCases := []struct {
		name       string
		urlCount   int
		iterations int
	}{
		{"small_cache", 10, 100},
		{"medium_cache", 50, 50},
		{"large_cache", 200, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate unique URLs to test cache
			urls := make([]string, tc.urlCount)
			for i := 0; i < tc.urlCount; i++ {
				urls[i] = mockServer.URL() + "/path" + generateEnvName(i)
				mockServer.AddResponse("/path"+generateEnvName(i), testutils.MockResponse{
					StatusCode: 200,
					Body:       `{"status": "ok"}`,
				})
			}

			// Pre-populate cache
			for _, url := range urls {
				validator.ValidateEndpoint(url)
			}

			// Measure cached validation performance
			for i := 0; i < tc.iterations; i++ {
				perfHelper.MeasureOperation(tc.name, func() {
					// Validate random URL from cache
					url := urls[i%len(urls)]
					validator.ValidateEndpoint(url)
				})
			}

			avgDuration := perfHelper.GetAverageDuration(tc.name)

			// Cached validations should be very fast
			assert.True(t, avgDuration < 1*time.Millisecond,
				"Cached validation for %s should be very fast: %v", tc.name, avgDuration)
		})
	}
}

func TestSystemLauncher_ConcurrentLaunches(t *testing.T) {
	processHelper := testutils.NewProcessHelper(t)
	defer processHelper.Cleanup()

	launcher := launcher.NewSystemLauncher()
	launcher.SetClaudeCodePath(processHelper.ExecutablePath)

	concHelper := testutils.NewConcurrencyTestHelper(t)

	// Test concurrent launches
	operations := make([]func() error, 10)
	for i := 0; i < 10; i++ {
		envName := generateEnvName(i)
		operations[i] = func() error {
			env := &types.Environment{
				Name:    envName,
				BaseURL: "https://" + envName + ".api.com/v1",
				APIKey:  envName + "-key-12345",
			}
			params := &types.LaunchParameters{
				Environment: env,
				Arguments:   []string{"--version"},
			}
			return launcher.Launch(params)
		}
	}

	startTime := time.Now()
	results := concHelper.RunConcurrentOperations(operations, 5)
	totalDuration := time.Since(startTime)

	// All launches should succeed
	for i, err := range results {
		assert.NoError(t, err, "Concurrent launch %d should succeed", i)
	}

	// Total time should be reasonable (less than sequential time)
	expectedSequentialTime := LaunchThreshold * time.Duration(len(operations))
	assert.True(t, totalDuration < expectedSequentialTime/2,
		"Concurrent launches should be faster than sequential: %v vs expected max %v",
		totalDuration, expectedSequentialTime/2)
}

func TestPerformanceRegression_Benchmarks(t *testing.T) {
	// These tests serve as performance regression benchmarks
	// They establish baseline performance expectations

	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	perfHelper := testutils.NewPerformanceHelper()

	// Benchmark basic operations
	benchmarks := []struct {
		name      string
		operation func()
		threshold time.Duration
	}{
		{
			name: "config_save_benchmark",
			operation: func() {
				manager.Save(testConfig)
			},
			threshold: ConfigSaveThreshold,
		},
		{
			name: "config_load_benchmark",
			operation: func() {
				manager.Load()
			},
			threshold: ConfigLoadThreshold,
		},
		{
			name: "config_validate_benchmark",
			operation: func() {
				manager.Validate(testConfig)
			},
			threshold: ConfigValidateThreshold,
		},
	}

	for _, bm := range benchmarks {
		t.Run(bm.name, func(t *testing.T) {
			// Run benchmark multiple times
			iterations := 100
			for i := 0; i < iterations; i++ {
				perfHelper.MeasureOperation(bm.name, bm.operation)
			}

			avgDuration := perfHelper.GetAverageDuration(bm.name)

			assert.True(t, avgDuration < bm.threshold,
				"Benchmark %s should meet threshold: %v (actual: %v)",
				bm.name, bm.threshold, avgDuration)
		})
	}
}

// Helper functions

func generateEnvName(index int) string {
	return "env-" + string(rune('a'+index%26)) + string(rune('0'+index/26))
}

func createTestEnvironmentBySize(size string) *types.Environment {
	env := &types.Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com/v1",
		APIKey:  "test-api-key-12345",
		Headers: make(map[string]string),
	}

	switch size {
	case "small":
		env.Headers["X-Test"] = "small"
	case "medium":
		// Add moderate number of headers
		for i := 0; i < 10; i++ {
			env.Headers["X-Header-"+string(rune('0'+i))] = "value-" + string(rune('0'+i))
		}
	case "large":
		// Add many headers
		for i := 0; i < 50; i++ {
			env.Headers["X-Header-"+string(rune('A'+i%26))+string(rune('0'+i/26))] = "large-value-" + string(rune('0'+i%10))
		}
	}

	return env
}

// Benchmark tests for Go's testing framework

func BenchmarkConfigManager_Save(b *testing.B) {
	testEnv := testutils.SetupTestEnvironment(&testing.T{})
	defer testEnv.Cleanup()

	manager, _ := config.NewFileConfigManager()
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Save(testConfig)
	}
}

func BenchmarkConfigManager_Load(b *testing.B) {
	testEnv := testutils.SetupTestEnvironment(&testing.T{})
	defer testEnv.Cleanup()

	manager, _ := config.NewFileConfigManager()
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()
	manager.Save(testConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Load()
	}
}

func BenchmarkConfigManager_Validate(b *testing.B) {
	manager, _ := config.NewFileConfigManager()
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Validate(testConfig)
	}
}

func BenchmarkNetworkValidator_ValidateEndpoint(b *testing.B) {
	mockServer := testutils.NewMockHTTPServer()
	defer mockServer.Close()

	mockServer.AddResponse("/", testutils.MockResponse{
		StatusCode: 200,
		Body:       `{"status": "ok"}`,
	})

	validator := network.NewValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateEndpoint(mockServer.URL())
	}
}

func BenchmarkNetworkValidator_CachedValidation(b *testing.B) {
	mockServer := testutils.NewMockHTTPServer()
	defer mockServer.Close()

	mockServer.AddResponse("/", testutils.MockResponse{
		StatusCode: 200,
		Body:       `{"status": "ok"}`,
	})

	validator := network.NewValidator()
	// Pre-populate cache
	validator.ValidateEndpoint(mockServer.URL())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateEndpoint(mockServer.URL())
	}
}

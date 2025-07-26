// Package performance provides enhanced performance testing for CCE 96/100 quality validation
package performance

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/builder"
	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/internal/validation"
	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/testutils"
)

// Enhanced performance thresholds for 96/100 quality validation
const (
	// Core operation thresholds (strict)
	ConfigSaveThresholdStrict     = 50 * time.Millisecond
	ConfigLoadThresholdStrict     = 20 * time.Millisecond
	ConfigValidateThresholdStrict = 5 * time.Millisecond

	// New component thresholds
	LaunchParametersBuildThreshold = 1 * time.Millisecond
	EnvironmentBuilderThreshold    = 2 * time.Millisecond
	ModelValidationThreshold       = 100 * time.Millisecond
	ModelValidationCachedThreshold = 1 * time.Millisecond

	// Concurrent operation thresholds
	ConcurrentConfigOpsThreshold  = 100 * time.Millisecond
	ConcurrentValidationThreshold = 150 * time.Millisecond

	// Memory thresholds (rough estimates)
	MaxMemoryPerOperation = 1024 * 1024 // 1MB per operation
)

func TestEnhancedPerformanceValidation(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	t.Run("LaunchParametersBuilderPerformance", func(t *testing.T) {
		env := &types.Environment{
			Name:    "perf-test-env",
			BaseURL: "https://api.performance.com/v1",
			APIKey:  "performance-test-key-12345",
			Model:   "claude-3-5-sonnet-20241022",
			Headers: map[string]string{
				"X-Performance": "test",
			},
		}

		// Test single operation performance
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := types.NewLaunchParametersBuilder().
				WithEnvironment(env).
				WithArguments([]string{"--help"}).
				WithTimeout(30 * time.Second).
				WithVerbose(true).
				WithDryRun(false).
				WithPassthroughMode(true).
				WithMetricsEnabled(true).
				Build()
			require.NoError(t, err)
		}

		totalTime := time.Since(start)
		avgTime := totalTime / time.Duration(iterations)

		assert.Less(t, avgTime, LaunchParametersBuildThreshold,
			"LaunchParameters build average should be < %v (actual: %v)",
			LaunchParametersBuildThreshold, avgTime)

		t.Logf("LaunchParameters build: %d iterations in %v (avg: %v)",
			iterations, totalTime, avgTime)
	})

	t.Run("EnvironmentVariableBuilderPerformance", func(t *testing.T) {
		env := &types.Environment{
			Name:    "env-builder-perf",
			BaseURL: "https://api.envbuilder.com/v1",
			APIKey:  "env-builder-key-12345",
			Model:   "claude-3-opus-20240229",
			Headers: map[string]string{
				"X-Header-1": "value1",
				"X-Header-2": "value2",
				"X-Header-3": "value3",
				"X-Header-4": "value4",
				"X-Header-5": "value5",
			},
		}

		customVars := map[string]string{
			"CUSTOM_VAR_1": "custom1",
			"CUSTOM_VAR_2": "custom2",
			"CUSTOM_VAR_3": "custom3",
		}

		iterations := 500
		start := time.Now()

		for i := 0; i < iterations; i++ {
			builder := builder.NewEnvironmentVariableBuilder()
			_ = builder.
				WithCurrentEnvironment().
				WithEnvironment(env).
				WithVariables(customVars).
				WithMasking(true).
				Build()
		}

		totalTime := time.Since(start)
		avgTime := totalTime / time.Duration(iterations)

		assert.Less(t, avgTime, EnvironmentBuilderThreshold,
			"EnvironmentBuilder average should be < %v (actual: %v)",
			EnvironmentBuilderThreshold, avgTime)

		t.Logf("EnvironmentBuilder: %d iterations in %v (avg: %v)",
			iterations, totalTime, avgTime)
	})

	t.Run("ModelValidationPerformance", func(t *testing.T) {
		validator := validation.NewEnhancedModelValidator()

		testModels := []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
			"unknown-model-1",
			"unknown-model-2",
			"",
		}

		// Test pattern validation performance
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			for _, model := range testModels {
				_, err := validator.ValidateModelName(model)
				require.NoError(t, err)
			}
		}

		totalTime := time.Since(start)
		avgTime := totalTime / time.Duration(iterations*len(testModels))

		assert.Less(t, avgTime, ModelValidationThreshold,
			"Model validation average should be < %v (actual: %v)",
			ModelValidationThreshold, avgTime)

		// Test cached validation performance
		cachedIterations := 1000
		start = time.Now()

		for i := 0; i < cachedIterations; i++ {
			// These should all be cached now
			_, err := validator.ValidateModelName("claude-3-5-sonnet-20241022")
			require.NoError(t, err)
		}

		cachedTotalTime := time.Since(start)
		cachedAvgTime := cachedTotalTime / time.Duration(cachedIterations)

		assert.Less(t, cachedAvgTime, ModelValidationCachedThreshold,
			"Cached model validation should be < %v (actual: %v)",
			ModelValidationCachedThreshold, cachedAvgTime)

		t.Logf("Model validation: %d iterations in %v (avg: %v)",
			iterations*len(testModels), totalTime, avgTime)
		t.Logf("Cached validation: %d iterations in %v (avg: %v)",
			cachedIterations, cachedTotalTime, cachedAvgTime)
	})

	t.Run("ConcurrentOperationsPerformance", func(t *testing.T) {
		// Test concurrent configuration operations
		numGoroutines := 10
		operationsPerGoroutine := 20

		testConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"concurrent-test": {
					Name:    "concurrent-test",
					BaseURL: "https://api.concurrent.com/v1",
					APIKey:  "concurrent-key-12345",
					Model:   "claude-3-5-sonnet-20241022",
				},
			},
		}

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Mix of save, load, and validate operations
					switch j % 3 {
					case 0:
						manager.Save(testConfig)
					case 1:
						manager.Load()
					case 2:
						manager.Validate(testConfig)
					}
				}
			}(i)
		}

		wg.Wait()
		totalTime := time.Since(start)
		avgTimePerOp := totalTime / time.Duration(numGoroutines*operationsPerGoroutine)

		assert.Less(t, avgTimePerOp, ConcurrentConfigOpsThreshold,
			"Concurrent config operations average should be < %v (actual: %v)",
			ConcurrentConfigOpsThreshold, avgTimePerOp)

		t.Logf("Concurrent config ops: %d total operations in %v (avg: %v)",
			numGoroutines*operationsPerGoroutine, totalTime, avgTimePerOp)
	})

	t.Run("ConcurrentModelValidationPerformance", func(t *testing.T) {
		validator := validation.NewEnhancedModelValidator()

		// Pre-populate some cache entries
		validator.ValidateModelName("claude-3-5-sonnet-20241022")
		validator.ValidateModelName("claude-3-opus-20240229")

		numGoroutines := 20
		operationsPerGoroutine := 50
		models := []string{
			"claude-3-5-sonnet-20241022", // cached
			"claude-3-opus-20240229",     // cached
			"claude-3-5-haiku-20241022",  // not cached
			"unknown-model",              // not cached
		}

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					model := models[j%len(models)]
					_, err := validator.ValidateModelName(model)
					if err != nil {
						t.Errorf("Worker %d: validation error for %s: %v", workerID, model, err)
					}
				}
			}(i)
		}

		wg.Wait()
		totalTime := time.Since(start)
		avgTimePerOp := totalTime / time.Duration(numGoroutines*operationsPerGoroutine)

		assert.Less(t, avgTimePerOp, ConcurrentValidationThreshold,
			"Concurrent validation average should be < %v (actual: %v)",
			ConcurrentValidationThreshold, avgTimePerOp)

		// Verify metrics were properly collected
		metrics := validator.GetMetrics()
		assert.Greater(t, metrics.PatternValidations, int64(0))
		assert.Greater(t, metrics.CacheHits, int64(0))
		assert.Greater(t, metrics.CacheMisses, int64(0))

		t.Logf("Concurrent validation: %d operations in %v (avg: %v)",
			numGoroutines*operationsPerGoroutine, totalTime, avgTimePerOp)
		t.Logf("Validation metrics: patterns=%d, hits=%d, misses=%d",
			metrics.PatternValidations, metrics.CacheHits, metrics.CacheMisses)
	})
}

func TestScalabilityPerformance(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	t.Run("LargeConfigurationHandling", func(t *testing.T) {
		// Test performance with large configurations
		sizes := []int{10, 50, 100, 250, 500}

		for _, size := range sizes {
			t.Run(fmt.Sprintf("Config_%d_environments", size), func(t *testing.T) {
				// Generate large configuration
				config := generateLargeConfig(size)

				// Test save performance
				start := time.Now()
				err := manager.Save(config)
				saveTime := time.Since(start)
				require.NoError(t, err)

				// Save time should scale sub-linearly
				expectedSaveTime := ConfigSaveThresholdStrict * time.Duration(1+size/50)
				assert.Less(t, saveTime, expectedSaveTime,
					"Save time for %d environments should be < %v (actual: %v)",
					size, expectedSaveTime, saveTime)

				// Test load performance
				start = time.Now()
				loadedConfig, err := manager.Load()
				loadTime := time.Since(start)
				require.NoError(t, err)
				assert.Len(t, loadedConfig.Environments, size)

				// Load time should scale sub-linearly
				expectedLoadTime := ConfigLoadThresholdStrict * time.Duration(1+size/100)
				assert.Less(t, loadTime, expectedLoadTime,
					"Load time for %d environments should be < %v (actual: %v)",
					size, expectedLoadTime, loadTime)

				// Test validate performance
				start = time.Now()
				err = manager.Validate(config)
				validateTime := time.Since(start)
				require.NoError(t, err)

				// Validate time should scale linearly
				expectedValidateTime := ConfigValidateThresholdStrict * time.Duration(1+size/20)
				assert.Less(t, validateTime, expectedValidateTime,
					"Validate time for %d environments should be < %v (actual: %v)",
					size, expectedValidateTime, validateTime)

				t.Logf("Config with %d environments: save=%v, load=%v, validate=%v",
					size, saveTime, loadTime, validateTime)
			})
		}
	})

	t.Run("EnvironmentBuilderScaling", func(t *testing.T) {
		// Test environment builder with varying numbers of headers
		headerCounts := []int{5, 25, 50, 100, 200}

		for _, headerCount := range headerCounts {
			t.Run(fmt.Sprintf("Headers_%d", headerCount), func(t *testing.T) {
				env := generateEnvironmentWithHeaders(headerCount)

				iterations := 100
				start := time.Now()

				for i := 0; i < iterations; i++ {
					builder := builder.NewEnvironmentVariableBuilder()
					_ = builder.WithEnvironment(env).Build()
				}

				totalTime := time.Since(start)
				avgTime := totalTime / time.Duration(iterations)

				// Should scale sub-linearly with header count
				expectedTime := EnvironmentBuilderThreshold * time.Duration(1+headerCount/100)
				assert.Less(t, avgTime, expectedTime,
					"EnvironmentBuilder with %d headers should be < %v (actual: %v)",
					headerCount, expectedTime, avgTime)

				t.Logf("EnvironmentBuilder with %d headers: %v avg", headerCount, avgTime)
			})
		}
	})

	t.Run("ModelValidationCacheScaling", func(t *testing.T) {
		validator := validation.NewEnhancedModelValidator()

		// Test cache performance with different numbers of cached entries
		cacheSizes := []int{10, 50, 100, 500, 1000}

		for _, cacheSize := range cacheSizes {
			t.Run(fmt.Sprintf("Cache_%d_entries", cacheSize), func(t *testing.T) {
				// Pre-populate cache
				for i := 0; i < cacheSize; i++ {
					model := fmt.Sprintf("test-model-%d", i)
					validator.ValidateModelName(model)
				}

				// Test cached lookup performance
				iterations := 200
				start := time.Now()

				for i := 0; i < iterations; i++ {
					// Access random cached entries
					model := fmt.Sprintf("test-model-%d", i%cacheSize)
					_, err := validator.ValidateModelName(model)
					require.NoError(t, err)
				}

				totalTime := time.Since(start)
				avgTime := totalTime / time.Duration(iterations)

				// Cached access should be consistently fast regardless of cache size
				assert.Less(t, avgTime, ModelValidationCachedThreshold*2,
					"Cached validation with %d entries should be < %v (actual: %v)",
					cacheSize, ModelValidationCachedThreshold*2, avgTime)

				t.Logf("Cache with %d entries: %v avg lookup time", cacheSize, avgTime)
			})
		}
	})
}

func TestMemoryPerformance(t *testing.T) {
	t.Run("MemoryUsageStability", func(t *testing.T) {
		// Test that repeated operations don't cause memory leaks
		validator := validation.NewEnhancedModelValidator()

		// Perform many operations that could potentially leak memory
		iterations := 1000
		for i := 0; i < iterations; i++ {
			// Create different patterns to avoid excessive caching
			model := fmt.Sprintf("test-model-%d", i%100)
			validator.ValidateModelName(model)

			// Environment building
			env := &types.Environment{
				Name:    fmt.Sprintf("env-%d", i%50),
				BaseURL: fmt.Sprintf("https://api-%d.com/v1", i%50),
				APIKey:  fmt.Sprintf("key-%d", i%50),
				Model:   model,
				Headers: map[string]string{
					"X-Iteration": fmt.Sprintf("%d", i),
				},
			}

			builder := builder.NewEnvironmentVariableBuilder()
			builder.WithEnvironment(env).Build()

			// Launch parameters building
			types.NewLaunchParametersBuilder().
				WithEnvironment(env).
				WithArguments([]string{fmt.Sprintf("--arg-%d", i)}).
				BuildUnsafe()
		}

		// Clear caches to test cleanup
		validator.ClearCache()

		// Additional operations after cleanup
		for i := 0; i < 100; i++ {
			validator.ValidateModelName("claude-3-5-sonnet-20241022")
		}

		// If we get here without running out of memory, the test passes
		// In a real scenario, you might use runtime.ReadMemStats to check actual usage
		t.Log("Memory stability test completed successfully")
	})
}

// Comprehensive benchmark suite for performance regression testing
func BenchmarkEnhancedComponents(b *testing.B) {
	b.Run("LaunchParametersBuilder", func(b *testing.B) {
		env := &types.Environment{
			Name:    "benchmark-env",
			BaseURL: "https://api.benchmark.com/v1",
			APIKey:  "benchmark-key-12345",
			Model:   "claude-3-5-sonnet-20241022",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			types.NewLaunchParametersBuilder().
				WithEnvironment(env).
				WithArguments([]string{"--benchmark"}).
				WithTimeout(30 * time.Second).
				Build()
		}
	})

	b.Run("EnvironmentVariableBuilder", func(b *testing.B) {
		env := &types.Environment{
			Name:    "benchmark-env",
			BaseURL: "https://api.benchmark.com/v1",
			APIKey:  "benchmark-key-12345",
			Model:   "claude-3-opus-20240229",
			Headers: map[string]string{
				"X-Benchmark": "true",
				"X-Version":   "1.1.0",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := builder.NewEnvironmentVariableBuilder()
			builder.WithEnvironment(env).Build()
		}
	})

	b.Run("ModelValidation_Pattern", func(b *testing.B) {
		validator := validation.NewEnhancedModelValidator()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validator.ValidateModelName("claude-3-5-sonnet-20241022")
		}
	})

	b.Run("ModelValidation_Cached", func(b *testing.B) {
		validator := validation.NewEnhancedModelValidator()
		// Pre-populate cache
		validator.ValidateModelName("claude-3-5-sonnet-20241022")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validator.ValidateModelName("claude-3-5-sonnet-20241022")
		}
	})
}

// Helper functions for performance testing

func generateLargeConfig(numEnvironments int) *types.Config {
	environments := make(map[string]types.Environment)

	for i := 0; i < numEnvironments; i++ {
		envName := fmt.Sprintf("env-%d", i)
		environments[envName] = types.Environment{
			Name:        envName,
			Description: fmt.Sprintf("Environment %d for performance testing", i),
			BaseURL:     fmt.Sprintf("https://api-%d.test.com/v1", i),
			APIKey:      fmt.Sprintf("key-%d-abcdef123456", i),
			Model:       "claude-3-5-sonnet-20241022",
			Headers: map[string]string{
				"X-Environment": envName,
				"X-Index":       fmt.Sprintf("%d", i),
				"X-Performance": "test",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	return &types.Config{
		Version:      "1.1.0",
		DefaultEnv:   "env-0",
		Environments: environments,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func generateEnvironmentWithHeaders(numHeaders int) *types.Environment {
	headers := make(map[string]string)

	for i := 0; i < numHeaders; i++ {
		key := fmt.Sprintf("X-Header-%d", i)
		value := fmt.Sprintf("value-%d", i)
		headers[key] = value
	}

	return &types.Environment{
		Name:    "header-test-env",
		BaseURL: "https://api.headers.com/v1",
		APIKey:  "header-test-key-12345",
		Model:   "claude-3-5-sonnet-20241022",
		Headers: headers,
	}
}

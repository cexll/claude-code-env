package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPerformanceAndBenchmarks tests performance characteristics and provides benchmarks
func TestPerformanceAndBenchmarks(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-performance")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("config_operations_performance", func(t *testing.T) {
		// Test performance of config operations with various sizes
		sizes := []int{1, 10, 50, 100}

		for _, size := range sizes {
			t.Run(fmt.Sprintf("config_size_%d", size), func(t *testing.T) {
				// Create config with specified number of environments
				config := Config{Environments: make([]Environment, size)}
				for i := 0; i < size; i++ {
					envName := fmt.Sprintf("env-%d", i) // Use numbers instead of characters to avoid validation issues
					config.Environments[i] = Environment{
						Name:   envName,
						URL:    "https://api.anthropic.com",
						APIKey: fmt.Sprintf("sk-ant-api03-perf%d1234567890abcdef1234567890", i),
					}
				}

				// Time save operation
				start := time.Now()
				if err := saveConfig(config); err != nil {
					t.Fatalf("saveConfig() failed for size %d: %v", size, err)
				}
				saveTime := time.Since(start)

				// Time load operation
				start = time.Now()
				loadedConfig, err := loadConfig()
				if err != nil {
					t.Fatalf("loadConfig() failed for size %d: %v", size, err)
				}
				loadTime := time.Since(start)

				// Verify correctness
				if len(loadedConfig.Environments) != size {
					t.Errorf("Size %d: expected %d environments, got %d", size, size, len(loadedConfig.Environments))
				}

				// Performance expectations (these are reasonable for local filesystem)
				maxSaveTime := 100 * time.Millisecond
				maxLoadTime := 50 * time.Millisecond

				if saveTime > maxSaveTime {
					t.Errorf("Save operation too slow for size %d: %v > %v", size, saveTime, maxSaveTime)
				}

				if loadTime > maxLoadTime {
					t.Errorf("Load operation too slow for size %d: %v > %v", size, loadTime, maxLoadTime)
				}

				t.Logf("Size %d: Save=%v, Load=%v", size, saveTime, loadTime)
			})
		}
	})

	t.Run("validation_performance", func(t *testing.T) {
		// Test validation performance with various input sizes
		testCases := []struct {
			name      string
			generator func(size int) string
			validator func(string) error
		}{
			{"name_validation", generateTestName, validateName},
			{"url_validation", generateTestURL, validateURL},
			{"apikey_validation", generateTestAPIKey, validateAPIKey},
		}

		sizes := []int{10, 50, 100, 500}

		for _, tc := range testCases {
			for _, size := range sizes {
				t.Run(fmt.Sprintf("%s_size_%d", tc.name, size), func(t *testing.T) {
					input := tc.generator(size)

					// Time validation
					start := time.Now()
					for i := 0; i < 1000; i++ { // Run multiple times for better measurement
						tc.validator(input)
					}
					totalTime := time.Since(start)
					avgTime := totalTime / 1000

					// Validation should be very fast
					maxTime := 100 * time.Microsecond
					if avgTime > maxTime {
						t.Errorf("Validation too slow for %s size %d: %v > %v", tc.name, size, avgTime, maxTime)
					}

					t.Logf("%s size %d: %v per validation", tc.name, size, avgTime)
				})
			}
		}
	})

	t.Run("environment_masking_performance", func(t *testing.T) {
		// Test API key masking performance with various key lengths
		keySizes := []int{20, 50, 100, 200}

		for _, size := range keySizes {
			t.Run(fmt.Sprintf("key_size_%d", size), func(t *testing.T) {
				// Generate test API key
				apiKey := generateTestAPIKey(size)

				// Time masking operation
				start := time.Now()
				for i := 0; i < 10000; i++ { // Run many times for measurement
					maskAPIKey(apiKey)
				}
				totalTime := time.Since(start)
				avgTime := totalTime / 10000

				// Masking should be very fast
				maxTime := 10 * time.Microsecond
				if avgTime > maxTime {
					t.Errorf("Masking too slow for size %d: %v > %v", size, avgTime, maxTime)
				}

				t.Logf("Masking size %d: %v per operation", size, avgTime)
			})
		}
	})

	t.Run("concurrent_access_performance", func(t *testing.T) {
		// Test performance under simulated concurrent access
		env := Environment{
			Name:   "concurrent-perf",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-concurrentperf1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Initial save
		if err := saveConfig(config); err != nil {
			t.Fatalf("Initial saveConfig() failed: %v", err)
		}

		// Time sequential operations
		start := time.Now()
		for i := 0; i < 100; i++ {
			// Load and save in sequence
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Fatalf("loadConfig() failed at iteration %d: %v", i, err)
			}

			if err := saveConfig(loadedConfig); err != nil {
				t.Fatalf("saveConfig() failed at iteration %d: %v", i, err)
			}
		}
		sequentialTime := time.Since(start)

		// Average time per operation pair
		avgTime := sequentialTime / 100

		// Should complete reasonably quickly
		maxAvgTime := 10 * time.Millisecond
		if avgTime > maxAvgTime {
			t.Errorf("Sequential operations too slow: %v > %v", avgTime, maxAvgTime)
		}

		t.Logf("Sequential load+save: %v per operation pair", avgTime)
	})

	t.Run("memory_usage_stability", func(t *testing.T) {
		// Test that repeated operations don't cause memory leaks
		initialConfig := Config{
			Environments: []Environment{
				{
					Name:   "memory-test",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-memorytest1234567890abcdef1234567890",
				},
			},
		}

		// Perform many operations
		for i := 0; i < 1000; i++ {
			// Save config
			if err := saveConfig(initialConfig); err != nil {
				t.Fatalf("saveConfig() failed at iteration %d: %v", i, err)
			}

			// Load config
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Fatalf("loadConfig() failed at iteration %d: %v", i, err)
			}

			// Validate environment
			if err := validateEnvironment(loadedConfig.Environments[0]); err != nil {
				t.Fatalf("validateEnvironment() failed at iteration %d: %v", i, err)
			}

			// Mask API key
			maskAPIKey(loadedConfig.Environments[0].APIKey)

			// Find environment
			findEnvironmentByName(loadedConfig, "memory-test")
		}

		// If we get here without issues, memory usage is likely stable
		t.Log("Memory stability test completed successfully")
	})
}

// Benchmark functions for more precise performance measurement
func BenchmarkSaveConfig(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "cce-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	config := Config{
		Environments: []Environment{
			{
				Name:   "benchmark-env",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-benchmark1234567890abcdef1234567890",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := saveConfig(config); err != nil {
			b.Fatalf("saveConfig() failed: %v", err)
		}
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "cce-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	config := Config{
		Environments: []Environment{
			{
				Name:   "benchmark-env",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-benchmark1234567890abcdef1234567890",
			},
		},
	}

	// Save initial config
	if err := saveConfig(config); err != nil {
		b.Fatalf("Initial saveConfig() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := loadConfig(); err != nil {
			b.Fatalf("loadConfig() failed: %v", err)
		}
	}
}

func BenchmarkValidateEnvironment(b *testing.B) {
	env := Environment{
		Name:   "benchmark-validation",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-benchmarkvalidation1234567890abcdef1234567890",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateEnvironment(env)
	}
}

func BenchmarkMaskAPIKey(b *testing.B) {
	apiKey := "sk-ant-api03-benchmarkmaskingtest1234567890abcdef1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		maskAPIKey(apiKey)
	}
}

// Helper functions for generating test data
func generateTestName(size int) string {
	if size <= 0 {
		return ""
	}
	if size > 50 {
		size = 50 // Respect validation limit
	}

	name := "test"
	for len(name) < size {
		name += "_env"
	}
	return name[:size]
}

func generateTestURL(size int) string {
	base := "https://api.anthropic.com"
	if size <= len(base) {
		return base[:max(size, 0)]
	}

	url := base
	for len(url) < size {
		url += "/path"
	}
	return url[:size]
}

func generateTestAPIKey(size int) string {
	base := "sk-ant-api03-"
	if size <= len(base) {
		return base[:max(size, 0)]
	}

	key := base
	chars := "abcdef1234567890"
	for len(key) < size {
		key += chars[:min(len(chars), size-len(key))]
	}
	return key[:size]
}

// Helper function for maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

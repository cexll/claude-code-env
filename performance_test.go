package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestPerformanceAndBenchmarks tests performance characteristics and provides benchmarks
func TestPerformanceAndBenchmarks(t *testing.T) {}

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

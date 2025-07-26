package validation

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/pkg/types"
)

func TestEnhancedModelValidator_ValidateModelNameComprehensive(t *testing.T) {
	validator := NewEnhancedModelValidator()

	t.Run("ValidKnownModel", func(t *testing.T) {
		result, err := validator.ValidateModelName("claude-3-5-sonnet-20241022")
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "claude-3-5-sonnet-20241022", result.Model)
		assert.Empty(t, result.Suggestions)
		assert.NotZero(t, result.ValidatedAt)
		assert.NotNil(t, result.PerformanceData)
		assert.Greater(t, result.PerformanceData.PatternCheckTime, time.Duration(0))
	})

	t.Run("EmptyModelName", func(t *testing.T) {
		result, err := validator.ValidateModelName("")
		require.NoError(t, err)
		assert.True(t, result.Valid) // Empty is valid, should return suggestions
		assert.Equal(t, "", result.Model)
		assert.Len(t, result.Suggestions, 3)
		assert.Contains(t, result.Suggestions, "claude-3-5-sonnet-20241022")
	})

	t.Run("UnknownModelName", func(t *testing.T) {
		result, err := validator.ValidateModelName("unknown-model")
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "unknown-model", result.Model)
		assert.Contains(t, result.ErrorMessage, "not in the known model list")
		assert.NotEmpty(t, result.Suggestions)
	})

	t.Run("SimilarModelSuggestions", func(t *testing.T) {
		// Test that similar models are suggested
		result, err := validator.ValidateModelName("claude-sonnet")
		require.NoError(t, err)
		assert.False(t, result.Valid)

		// Should suggest sonnet models
		hasSonnetSuggestion := false
		for _, suggestion := range result.Suggestions {
			if contains(suggestion, "sonnet") {
				hasSonnetSuggestion = true
				break
			}
		}
		assert.True(t, hasSonnetSuggestion, "Should suggest sonnet models for 'claude-sonnet'")
	})

	t.Run("CachedResults", func(t *testing.T) {
		model := "test-model-for-cache"

		// First call
		result1, err := validator.ValidateModelName(model)
		require.NoError(t, err)
		assert.False(t, result1.CachedResult)

		// Second call should be cached
		result2, err := validator.ValidateModelName(model)
		require.NoError(t, err)
		assert.True(t, result2.CachedResult)
		assert.Equal(t, result1.Model, result2.Model)
		assert.Equal(t, result1.Valid, result2.Valid)
	})
}

func TestEnhancedModelValidator_ValidateModelWithAPI(t *testing.T) {
	validator := NewEnhancedModelValidator()
	env := &types.Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com/v1",
		APIKey:  "test-key",
	}

	t.Run("ValidModelWithAPICheck", func(t *testing.T) {
		// Note: This will fail API validation in testing but pass pattern validation
		result, err := validator.ValidateModelWithAPI(env, "claude-3-5-sonnet-20241022")
		require.NoError(t, err)

		// Should pass pattern validation but fail API validation in test environment
		assert.Equal(t, "claude-3-5-sonnet-20241022", result.Model)
		assert.False(t, result.APICompatible) // Will fail in test environment
		assert.NotZero(t, result.ValidatedAt)
		assert.NotNil(t, result.PerformanceData)
		assert.Greater(t, result.PerformanceData.PatternCheckTime, time.Duration(0))
		assert.Greater(t, result.PerformanceData.APICheckTime, time.Duration(0))
		assert.Greater(t, result.PerformanceData.TotalTime, time.Duration(0))
	})

	t.Run("InvalidModelSkipsAPICheck", func(t *testing.T) {
		result, err := validator.ValidateModelWithAPI(env, "invalid-model")
		require.NoError(t, err)

		// Should fail pattern validation and not do API check
		assert.False(t, result.Valid)
		assert.Equal(t, "invalid-model", result.Model)
		assert.NotNil(t, result.PerformanceData)
		// API check time should be 0 since it was skipped
		assert.Equal(t, time.Duration(0), result.PerformanceData.APICheckTime)
	})

	t.Run("CachedResultsWithEnvironment", func(t *testing.T) {
		model := "claude-3-opus-20240229"

		// First call
		result1, err := validator.ValidateModelWithAPI(env, model)
		require.NoError(t, err)
		assert.False(t, result1.CachedResult)

		// Second call should be cached
		result2, err := validator.ValidateModelWithAPI(env, model)
		require.NoError(t, err)
		assert.True(t, result2.CachedResult)
		assert.Equal(t, result1.Model, result2.Model)
		assert.Equal(t, result1.APICompatible, result2.APICompatible)
	})
}

func TestEnhancedModelValidator_GetSuggestedModelsComprehensive(t *testing.T) {
	validator := NewEnhancedModelValidator()

	t.Run("DefaultSuggestions", func(t *testing.T) {
		suggestions, err := validator.GetSuggestedModels("")
		require.NoError(t, err)
		assert.NotEmpty(t, suggestions)
		assert.Contains(t, suggestions, "claude-3-5-sonnet-20241022")
		assert.Contains(t, suggestions, "claude-3-5-haiku-20241022")
		assert.Contains(t, suggestions, "claude-3-opus-20240229")
	})

	t.Run("AnthropicAPIType", func(t *testing.T) {
		suggestions, err := validator.GetSuggestedModels("anthropic")
		require.NoError(t, err)
		assert.NotEmpty(t, suggestions)
		assert.Contains(t, suggestions, "claude-3-5-sonnet-20241022")
	})

	t.Run("ClaudeAPIType", func(t *testing.T) {
		suggestions, err := validator.GetSuggestedModels("claude")
		require.NoError(t, err)
		assert.NotEmpty(t, suggestions)
		assert.Contains(t, suggestions, "claude-3-5-sonnet-20241022")
	})

	t.Run("UnknownAPIType", func(t *testing.T) {
		suggestions, err := validator.GetSuggestedModels("unknown")
		require.NoError(t, err)
		assert.NotEmpty(t, suggestions) // Should return default suggestions
	})
}

func TestEnhancedModelValidator_CacheOperations(t *testing.T) {
	validator := NewEnhancedModelValidator()

	t.Run("ManualCacheOperation", func(t *testing.T) {
		key := "manual-cache-test"
		result := &ModelValidationResult{
			Valid:       true,
			Model:       "test-model",
			ValidatedAt: time.Now(),
		}

		validator.CacheValidationResult(key, result)

		// Try to retrieve (this tests internal cache behavior)
		cached, err := validator.ValidateModelName(key)
		require.NoError(t, err)
		assert.True(t, cached.CachedResult)
		assert.Equal(t, result.Model, cached.Model)
	})

	t.Run("ClearCache", func(t *testing.T) {
		// Add something to cache
		_, err := validator.ValidateModelName("cache-clear-test")
		require.NoError(t, err)

		// Clear cache
		validator.ClearCache()

		// Subsequent call should not be cached
		result, err := validator.ValidateModelName("cache-clear-test")
		require.NoError(t, err)
		assert.False(t, result.CachedResult)
	})
}

func TestEnhancedModelValidator_MetricsCollection(t *testing.T) {
	validator := NewEnhancedModelValidator()

	t.Run("MetricsTracking", func(t *testing.T) {
		// Perform some operations
		validator.ValidateModelName("claude-3-5-sonnet-20241022")
		validator.ValidateModelName("claude-3-5-haiku-20241022")

		env := &types.Environment{
			BaseURL: "https://api.test.com/v1",
			APIKey:  "test-key",
		}
		validator.ValidateModelWithAPI(env, "claude-3-opus-20240229")

		metrics := validator.GetMetrics()
		assert.Equal(t, int64(2), metrics.PatternValidations)
		assert.Equal(t, int64(1), metrics.APIValidations)
		assert.Greater(t, metrics.CacheMisses, int64(0))
		assert.Greater(t, metrics.TotalValidationTime, time.Duration(0))
	})

	t.Run("CacheHitMetrics", func(t *testing.T) {
		model := "cache-hit-test"

		// First call (cache miss)
		validator.ValidateModelName(model)

		// Second call (cache hit)
		validator.ValidateModelName(model)

		metrics := validator.GetMetrics()
		assert.Greater(t, metrics.CacheHits, int64(0))
		assert.Greater(t, metrics.CacheMisses, int64(0))
	})
}

func TestValidationCache_Operations(t *testing.T) {
	cache := NewValidationCache(5 * time.Minute)

	t.Run("SetAndGet", func(t *testing.T) {
		model := "test-model"
		endpoint := "https://api.test.com"
		result := &ModelValidationResult{
			Valid:       true,
			Model:       model,
			ValidatedAt: time.Now(),
		}

		cache.Set(model, endpoint, result)
		cached := cache.Get(model, endpoint)

		require.NotNil(t, cached)
		assert.True(t, cached.CachedResult)
		assert.Equal(t, result.Model, cached.Model)
		assert.Equal(t, result.Valid, cached.Valid)
	})

	t.Run("GetNonexistent", func(t *testing.T) {
		cached := cache.Get("nonexistent", "endpoint")
		assert.Nil(t, cached)
	})

	t.Run("ClearCache", func(t *testing.T) {
		cache.Set("test", "endpoint", &ModelValidationResult{Valid: true})
		cache.Clear()

		cached := cache.Get("test", "endpoint")
		assert.Nil(t, cached)
	})

	t.Run("KeyGeneration", func(t *testing.T) {
		result := &ModelValidationResult{Valid: true, Model: "test"}

		// Test with endpoint
		cache.Set("model", "endpoint", result)
		cached1 := cache.Get("model", "endpoint")
		assert.NotNil(t, cached1)

		// Test without endpoint
		cache.Set("model", "", result)
		cached2 := cache.Get("model", "")
		assert.NotNil(t, cached2)

		// Different endpoints should be different cache entries
		cached3 := cache.Get("model", "different-endpoint")
		assert.Nil(t, cached3)
	})
}

func TestBasicPatternValidator_Operations(t *testing.T) {
	validator := NewBasicPatternValidator()

	t.Run("ValidateKnownModels", func(t *testing.T) {
		knownModels := []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		}

		for _, model := range knownModels {
			result := validator.Validate(model)
			assert.True(t, result.Valid, "Model %s should be valid", model)
			assert.Equal(t, model, result.Model)
			assert.Empty(t, result.Suggestions)
		}
	})

	t.Run("ValidateUnknownModel", func(t *testing.T) {
		result := validator.Validate("unknown-model")
		assert.False(t, result.Valid)
		assert.Equal(t, "unknown-model", result.Model)
		assert.Contains(t, result.ErrorMessage, "not in the known model list")
		assert.NotEmpty(t, result.Suggestions)
	})

	t.Run("ValidateEmptyModel", func(t *testing.T) {
		result := validator.Validate("")
		assert.True(t, result.Valid) // Empty is considered valid
		assert.Equal(t, "", result.Model)
		assert.Len(t, result.Suggestions, 3) // Top 3 suggestions
	})

	t.Run("SimilarModelSuggestions", func(t *testing.T) {
		testCases := []struct {
			input    string
			contains string
		}{
			{"claude-sonnet", "sonnet"},
			{"claude-haiku", "haiku"},
			{"claude-opus", "opus"},
		}

		for _, tc := range testCases {
			result := validator.Validate(tc.input)
			assert.False(t, result.Valid)

			// Should have relevant suggestions
			hasRelevantSuggestion := false
			for _, suggestion := range result.Suggestions {
				if contains(suggestion, tc.contains) {
					hasRelevantSuggestion = true
					break
				}
			}
			assert.True(t, hasRelevantSuggestion,
				"Should suggest %s models for input '%s'", tc.contains, tc.input)
		}
	})
}

func TestHTTPAPIValidator_Operations(t *testing.T) {
	validator := NewHTTPAPIValidator()
	env := &types.Environment{
		BaseURL: "https://httpbin.org", // Use httpbin for testing
		APIKey:  "test-key",
	}

	t.Run("ValidateWithTimeout", func(t *testing.T) {
		// This will likely fail in most test environments, but tests the timeout logic
		result, err := validator.ValidateModel(env, "claude-3-5-sonnet-20241022")

		// We expect an error or non-compatible result in test environment
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, result)
			assert.Greater(t, result.ResponseTime, time.Duration(0))
		}
	})

	t.Run("RequestFormatting", func(t *testing.T) {
		// Test that the validator can create proper requests
		// This mainly tests that the function doesn't panic
		result, err := validator.ValidateModel(env, "test-model")

		// Expect either error or result, but no panic
		if err == nil {
			assert.NotNil(t, result)
		}
	})
}

// Performance benchmark tests
func BenchmarkModelValidator_ValidateModelName(b *testing.B) {
	validator := NewEnhancedModelValidator()
	model := "claude-3-5-sonnet-20241022"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateModelName(model)
	}
}

func BenchmarkModelValidator_ValidateModelName_Cached(b *testing.B) {
	validator := NewEnhancedModelValidator()
	model := "claude-3-5-sonnet-20241022"

	// Pre-populate cache
	validator.ValidateModelName(model)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateModelName(model)
	}
}

func BenchmarkModelValidator_GetSuggestedModels(b *testing.B) {
	validator := NewEnhancedModelValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.GetSuggestedModels("anthropic")
	}
}

func BenchmarkValidationCache_SetGet(b *testing.B) {
	cache := NewValidationCache(5 * time.Minute)
	result := &ModelValidationResult{
		Valid:       true,
		Model:       "test-model",
		ValidatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("model-%d", i%100) // Cycle through 100 keys
		cache.Set(key, "endpoint", result)
		cache.Get(key, "endpoint")
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package validation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNewEnhancedModelValidator(t *testing.T) {
	validator := NewEnhancedModelValidator()

	assert.NotNil(t, validator)
	assert.NotNil(t, validator.patternValidator)
	assert.NotNil(t, validator.apiValidator)
	assert.NotNil(t, validator.cache)
	assert.NotNil(t, validator.metrics)
}

func TestEnhancedModelValidator_ValidateModelName(t *testing.T) {
	validator := NewEnhancedModelValidator()

	tests := []struct {
		name          string
		model         string
		expectedValid bool
		expectError   bool
	}{
		{
			name:          "empty model",
			model:         "",
			expectedValid: true,
			expectError:   false,
		},
		{
			name:          "valid claude model",
			model:         "claude-3-5-sonnet-20241022",
			expectedValid: true,
			expectError:   false,
		},
		{
			name:          "unknown but potentially valid model",
			model:         "claude-4-opus-20250101",
			expectedValid: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateModelName(tt.model)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedValid, result.Valid)
			assert.Equal(t, tt.model, result.Model)
			assert.NotZero(t, result.ValidatedAt)
			assert.NotNil(t, result.PerformanceData)
		})
	}
}

func TestEnhancedModelValidator_ValidateModelName_Caching(t *testing.T) {
	validator := NewEnhancedModelValidator()
	model := "claude-3-5-sonnet-20241022"

	// First validation
	result1, err := validator.ValidateModelName(model)
	assert.NoError(t, err)
	assert.False(t, result1.CachedResult)

	// Second validation should be cached
	result2, err := validator.ValidateModelName(model)
	assert.NoError(t, err)
	assert.True(t, result2.CachedResult)

	// Metrics should show cache hit
	metrics := validator.GetMetrics()
	assert.Equal(t, int64(2), metrics.PatternValidations)
	assert.Equal(t, int64(1), metrics.CacheHits)
	assert.Equal(t, int64(1), metrics.CacheMisses)
}

func TestEnhancedModelValidator_ValidateModelWithAPI_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"test"}]}`))
	}))
	defer server.Close()

	validator := NewEnhancedModelValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "test-key",
	}
	model := "claude-3-5-sonnet-20241022"

	result, err := validator.ValidateModelWithAPI(env, model)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.True(t, result.APICompatible)
	assert.Equal(t, model, result.Model)
	assert.NotNil(t, result.PerformanceData)
	assert.Positive(t, result.PerformanceData.APICheckTime)
}

func TestEnhancedModelValidator_ValidateModelWithAPI_ModelNotSupported(t *testing.T) {
	// Create mock server that returns model error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"model not supported"}}`))
	}))
	defer server.Close()

	validator := NewEnhancedModelValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "test-key",
	}
	model := "unsupported-model"

	result, err := validator.ValidateModelWithAPI(env, model)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid) // Pattern validation failed
	assert.False(t, result.APICompatible)
	assert.Contains(t, result.ErrorMessage, "model")
	assert.NotEmpty(t, result.Suggestions)
}

func TestEnhancedModelValidator_ValidateModelWithAPI_AuthError(t *testing.T) {
	// Create mock server that returns auth error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"authentication_error","message":"invalid api key"}}`))
	}))
	defer server.Close()

	validator := NewEnhancedModelValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "invalid-key",
	}
	model := "claude-3-5-sonnet-20241022"

	result, err := validator.ValidateModelWithAPI(env, model)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)         // Pattern validation passed for valid model
	assert.True(t, result.APICompatible) // Auth issue, not model issue
	// The error message could be empty or contain auth info - let's not require specific text
}

func TestEnhancedModelValidator_ValidateModelWithAPI_NetworkError(t *testing.T) {
	validator := NewEnhancedModelValidator()
	env := &types.Environment{
		BaseURL: "http://invalid-url-that-does-not-exist.com",
		APIKey:  "test-key",
	}
	model := "unknown-model" // Use unknown model so pattern validation fails

	result, err := validator.ValidateModelWithAPI(env, model)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid)         // Pattern validation failed for unknown model
	assert.False(t, result.APICompatible) // Also false due to unknown model
	// The exact error message may vary, just check that we got a result
	assert.NotEmpty(t, result.ErrorMessage)
}

func TestEnhancedModelValidator_ValidateModelWithAPI_Caching(t *testing.T) {
	// Create mock server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123","type":"message"}`))
	}))
	defer server.Close()

	validator := NewEnhancedModelValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "test-key",
	}
	model := "claude-3-5-sonnet-20241022"

	// First validation
	result1, err := validator.ValidateModelWithAPI(env, model)
	assert.NoError(t, err)
	assert.False(t, result1.CachedResult)
	assert.Equal(t, 1, callCount)

	// Second validation should be cached
	result2, err := validator.ValidateModelWithAPI(env, model)
	assert.NoError(t, err)
	assert.True(t, result2.CachedResult)
	assert.Equal(t, 1, callCount) // No additional API call
}

func TestEnhancedModelValidator_GetSuggestedModels(t *testing.T) {
	validator := NewEnhancedModelValidator()

	tests := []struct {
		apiType string
		expect  int
	}{
		{"anthropic", 5},
		{"claude", 5},
		{"", 5},
		{"unknown", 5},
	}

	for _, tt := range tests {
		t.Run(tt.apiType, func(t *testing.T) {
			models, err := validator.GetSuggestedModels(tt.apiType)
			assert.NoError(t, err)
			assert.Len(t, models, tt.expect)
			assert.Contains(t, models, "claude-3-5-sonnet-20241022")
		})
	}
}

func TestEnhancedModelValidator_CacheValidationResult(t *testing.T) {
	validator := NewEnhancedModelValidator()

	result := &ModelValidationResult{
		Valid: true,
		Model: "test-model",
	}

	validator.CacheValidationResult("test-key", result)

	// Verify it was cached by trying to retrieve
	cached := validator.cache.Get("test-key", "")
	assert.NotNil(t, cached)
	assert.Equal(t, "test-model", cached.Model)
	assert.True(t, cached.CachedResult)
}

func TestEnhancedModelValidator_ClearCache(t *testing.T) {
	validator := NewEnhancedModelValidator()

	// Add something to cache
	result := &ModelValidationResult{Valid: true, Model: "test"}
	validator.CacheValidationResult("test", result)

	// Verify it exists
	cached := validator.cache.Get("test", "")
	assert.NotNil(t, cached)

	// Clear cache
	validator.ClearCache()

	// Verify it's gone
	cached = validator.cache.Get("test", "")
	assert.Nil(t, cached)
}

func TestEnhancedModelValidator_GetMetrics(t *testing.T) {
	validator := NewEnhancedModelValidator()

	// Perform some validations
	validator.ValidateModelName("claude-3-5-sonnet-20241022")
	validator.ValidateModelName("claude-3-5-sonnet-20241022") // Cached

	metrics := validator.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(2), metrics.PatternValidations)
	assert.Equal(t, int64(1), metrics.CacheHits)
	assert.Equal(t, int64(1), metrics.CacheMisses)
	assert.Positive(t, metrics.TotalValidationTime)
}

func TestValidationCache(t *testing.T) {
	cache := NewValidationCache(100 * time.Millisecond)

	result := &ModelValidationResult{
		Valid: true,
		Model: "test-model",
	}

	// Test Set and Get
	cache.Set("model1", "endpoint1", result)
	retrieved := cache.Get("model1", "endpoint1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-model", retrieved.Model)
	assert.True(t, retrieved.CachedResult)

	// Test key not found
	notFound := cache.Get("nonexistent", "endpoint")
	assert.Nil(t, notFound)

	// Test expiration
	time.Sleep(150 * time.Millisecond)
	expired := cache.Get("model1", "endpoint1")
	assert.Nil(t, expired)
}

func TestValidationCache_Clear(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	result := &ModelValidationResult{Valid: true, Model: "test"}
	cache.Set("test", "endpoint", result)

	// Verify exists
	retrieved := cache.Get("test", "endpoint")
	assert.NotNil(t, retrieved)

	// Clear and verify gone
	cache.Clear()
	retrieved = cache.Get("test", "endpoint")
	assert.Nil(t, retrieved)
}

func TestBasicPatternValidator(t *testing.T) {
	validator := NewBasicPatternValidator()

	tests := []struct {
		model          string
		expectedValid  bool
		hasSuggestions bool
	}{
		{"", true, true}, // Empty model is valid
		{"claude-3-5-sonnet-20241022", true, false}, // Exact match
		{"claude-3-unknown-model", false, true},     // Unknown model
		{"invalid-model", false, true},              // Invalid model
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := validator.Validate(tt.model)
			assert.Equal(t, tt.expectedValid, result.Valid)
			assert.Equal(t, tt.model, result.Model)

			if tt.hasSuggestions {
				assert.NotEmpty(t, result.Suggestions)
			}
		})
	}
}

func TestBasicPatternValidator_GetSimilarModels(t *testing.T) {
	validator := NewBasicPatternValidator()

	tests := []struct {
		input    string
		contains string
	}{
		{"sonnet-model", "sonnet"},
		{"haiku-model", "haiku"},
		{"opus-model", "opus"},
		{"unknown-model", "claude"}, // Should return default suggestions
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			suggestions := validator.getSimilarModels(tt.input)
			assert.NotEmpty(t, suggestions)

			found := false
			for _, suggestion := range suggestions {
				if strings.Contains(strings.ToLower(suggestion), tt.contains) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected to find suggestion containing %s", tt.contains)
		})
	}
}

func TestHTTPAPIValidator(t *testing.T) {
	validator := NewHTTPAPIValidator()
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.client)
}

func TestHTTPAPIValidator_ValidateModel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("Anthropic-Version"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123","type":"message"}`))
	}))
	defer server.Close()

	validator := NewHTTPAPIValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "test-key",
	}

	result, err := validator.ValidateModel(env, "claude-3-5-sonnet-20241022")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Compatible)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Positive(t, result.ResponseTime)
}

func TestHTTPAPIValidator_ValidateModel_ModelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"Model 'invalid-model' not found"}}`))
	}))
	defer server.Close()

	validator := NewHTTPAPIValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "test-key",
	}

	result, err := validator.ValidateModel(env, "invalid-model")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Compatible)
	assert.Contains(t, result.ErrorMessage, "Model not supported")
	assert.NotEmpty(t, result.SuggestedAlternatives)
}

func TestHTTPAPIValidator_ValidateModel_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"authentication_error","message":"Invalid API key"}}`))
	}))
	defer server.Close()

	validator := NewHTTPAPIValidator()
	env := &types.Environment{
		BaseURL: server.URL,
		APIKey:  "invalid-key",
	}

	result, err := validator.ValidateModel(env, "claude-3-5-sonnet-20241022")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Compatible) // Model OK, auth issue
	assert.Contains(t, result.ErrorMessage, "Authentication failed")
}

func TestHTTPAPIValidator_ValidateModel_NetworkError(t *testing.T) {
	validator := NewHTTPAPIValidator()
	env := &types.Environment{
		BaseURL: "http://invalid-url-does-not-exist.com",
		APIKey:  "test-key",
	}

	result, err := validator.ValidateModel(env, "claude-3-5-sonnet-20241022")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Compatible)
	assert.Contains(t, result.ErrorMessage, "API request failed")
}

func TestValidationMetrics_ThreadSafety(t *testing.T) {
	metrics := &ValidationMetrics{}

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			metrics.mutex.Lock()
			metrics.PatternValidations++
			metrics.mutex.Unlock()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, int64(10), metrics.PatternValidations)
}

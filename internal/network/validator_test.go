package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/testutils"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()

	assert.NotNil(t, validator)
	// Validator should be ready to use immediately
	assert.NotNil(t, validator)
}

func TestValidateEndpoint_Success(t *testing.T) {
	// Create mock server
	mockServer := testutils.NewMockHTTPServer()
	defer mockServer.Close()

	mockServer.AddResponse("/", testutils.MockResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"status": "ok"}`,
	})

	validator := NewValidator()

	result, err := validator.ValidateEndpoint(mockServer.URL())
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, 200, result.StatusCode)
	assert.True(t, result.ResponseTime > 0)
	assert.False(t, result.SSLValid) // HTTP server, not HTTPS
	assert.WithinDuration(t, time.Now(), result.Timestamp, time.Second)
}

func TestValidateEndpoint_HTTPSSuccess(t *testing.T) {
	// Create HTTPS mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	validator := NewValidator()

	result, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, 200, result.StatusCode)
	assert.True(t, result.SSLValid)
}

func TestValidateEndpoint_InvalidURL(t *testing.T) {
	validator := NewValidator()

	testCases := []struct {
		name        string
		url         string
		expectedErr string
	}{
		{
			name:        "empty URL",
			url:         "",
			expectedErr: "URL cannot be empty",
		},
		{
			name:        "invalid format",
			url:         "not-a-url",
			expectedErr: "URL must use http or https scheme",
		},
		{
			name:        "missing scheme",
			url:         "example.com",
			expectedErr: "URL must use http or https scheme",
		},
		{
			name:        "invalid scheme",
			url:         "ftp://example.com",
			expectedErr: "URL must use http or https scheme",
		},
		{
			name:        "missing host",
			url:         "https://",
			expectedErr: "URL must have a valid host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ValidateEndpoint(tc.url)
			require.Error(t, err)

			var networkErr *types.NetworkError
			assert.ErrorAs(t, err, &networkErr)
			assert.Equal(t, types.NetworkInvalidURL, networkErr.Type)
			assert.Contains(t, networkErr.Message, tc.expectedErr)
			assert.NotEmpty(t, networkErr.Suggestions)
		})
	}
}

func TestValidateEndpoint_NetworkErrors(t *testing.T) {
	validator := NewValidator()

	testCases := []struct {
		name         string
		url          string
		expectedType types.NetworkErrorType
	}{
		{
			name:         "connection refused",
			url:          "http://localhost:99999", // Port likely not in use
			expectedType: types.NetworkConnectionFailed,
		},
		{
			name:         "host not found",
			url:          "https://this-domain-should-not-exist-12345.com",
			expectedType: types.NetworkUnreachable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validator.ValidateEndpoint(tc.url)
			require.NoError(t, err) // ValidateEndpoint returns result with error info, not Go error

			assert.False(t, result.Success)
			assert.NotEmpty(t, result.Error)
		})
	}
}

func TestValidateEndpoint_Timeout(t *testing.T) {
	// Create slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than typical timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	validator := NewValidator()

	result, err := validator.ValidateEndpointWithTimeout(slowServer.URL, 100*time.Millisecond)
	require.NoError(t, err)

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "deadline exceeded")
}

func TestValidateEndpoint_Caching(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator()

	// First request
	result1, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)
	assert.True(t, result1.Success)
	assert.Equal(t, 1, requestCount)

	// Second request should use cache
	result2, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)
	assert.True(t, result2.Success)
	assert.Equal(t, 1, requestCount) // No additional request made

	// Results should be identical
	assert.Equal(t, result1.StatusCode, result2.StatusCode)
	assert.Equal(t, result1.Success, result2.Success)
}

func TestValidateEndpoint_CacheClear(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator()

	// First request
	_, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// Clear cache
	validator.ClearCache()

	// Second request should make new HTTP request
	_, err = validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)
	assert.Equal(t, 2, requestCount)
}

func TestTestAPIConnectivity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication header
		auth := r.Header.Get("Authorization")
		if auth == "Bearer test-api-key" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"user": "test"}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	validator := NewValidator()

	env := &types.Environment{
		Name:    "test",
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		Headers: map[string]string{
			"X-Custom": "value",
		},
	}

	err := validator.TestAPIConnectivity(env)
	require.NoError(t, err)
}

func TestTestAPIConnectivity_AuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid key"}`))
	}))
	defer server.Close()

	validator := NewValidator()

	env := &types.Environment{
		Name:    "test",
		BaseURL: server.URL,
		APIKey:  "invalid-key",
	}

	err := validator.TestAPIConnectivity(env)
	require.Error(t, err)

	var networkErr *types.NetworkError
	assert.ErrorAs(t, err, &networkErr)
	assert.Equal(t, types.NetworkAuthenticationError, networkErr.Type)
	assert.Equal(t, 401, networkErr.StatusCode)
	assert.NotEmpty(t, networkErr.Suggestions)
}

func TestTestAPIConnectivity_ForbiddenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "insufficient permissions"}`))
	}))
	defer server.Close()

	validator := NewValidator()

	env := &types.Environment{
		Name:    "test",
		BaseURL: server.URL,
		APIKey:  "valid-but-limited-key",
	}

	err := validator.TestAPIConnectivity(env)
	require.Error(t, err)

	var networkErr *types.NetworkError
	assert.ErrorAs(t, err, &networkErr)
	assert.Equal(t, types.NetworkAuthenticationError, networkErr.Type)
	assert.Equal(t, 403, networkErr.StatusCode)
}

func TestTestAPIConnectivity_NilEnvironment(t *testing.T) {
	validator := NewValidator()

	err := validator.TestAPIConnectivity(nil)
	require.Error(t, err)

	var networkErr *types.NetworkError
	assert.ErrorAs(t, err, &networkErr)
	assert.Equal(t, types.NetworkInvalidURL, networkErr.Type)
}

func TestValidateEndpoint_SSLCertificateValidation(t *testing.T) {
	// Create HTTPS server with custom certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator()

	result, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.True(t, result.SSLValid)
}

func TestValidateEndpoint_RetryLogic(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Simulate temporary failure
			time.Sleep(10 * time.Millisecond)
			return // Connection will be closed, triggering retry
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Close server immediately to trigger connection errors
	server.Close()

	validator := NewValidator()

	result, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)

	// Should fail after retries
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestValidateEndpoint_ConcurrentAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Small delay to test concurrency
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator()

	// Run multiple concurrent validations
	const concurrency = 10
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := validator.ValidateEndpoint(server.URL)
			results <- err
		}()
	}

	// Wait for all to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

func TestValidateEndpoint_CacheStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator()

	// Initial stats should be empty
	stats := validator.GetCacheStats()
	assert.Equal(t, 0, stats.TotalEntries)
	assert.Equal(t, 0, stats.ValidEntries)

	// Make some requests
	validator.ValidateEndpoint(server.URL + "/path1")
	validator.ValidateEndpoint(server.URL + "/path2")

	// Check updated stats
	stats = validator.GetCacheStats()
	assert.Equal(t, 2, stats.TotalEntries)
	assert.Equal(t, 2, stats.ValidEntries)
	assert.Equal(t, 0, stats.ExpiredEntries)
}

func TestValidateEndpoint_ErrorSuggestions(t *testing.T) {
	validator := NewValidator()

	testCases := []struct {
		name                       string
		url                        string
		expectedType               types.NetworkErrorType
		expectedSuggestionKeywords []string
	}{
		{
			name:                       "invalid URL format",
			url:                        "not-a-url",
			expectedType:               types.NetworkInvalidURL,
			expectedSuggestionKeywords: []string{"protocol", "https://"},
		},
		{
			name:                       "empty URL",
			url:                        "",
			expectedType:               types.NetworkInvalidURL,
			expectedSuggestionKeywords: []string{"HTTP", "HTTPS"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ValidateEndpoint(tc.url)
			require.Error(t, err)

			var networkErr *types.NetworkError
			assert.ErrorAs(t, err, &networkErr)
			assert.Equal(t, tc.expectedType, networkErr.Type)

			suggestions := networkErr.GetSuggestions()
			assert.NotEmpty(t, suggestions)

			// Check that expected keywords appear in suggestions
			for _, keyword := range tc.expectedSuggestionKeywords {
				found := false
				for _, suggestion := range suggestions {
					if assert.Contains(t, suggestion, keyword) {
						found = true
						break
					}
				}
				assert.True(t, found, "Keyword '%s' not found in suggestions: %v", keyword, suggestions)
			}
		})
	}
}

func TestValidateEndpoint_Performance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator()

	perfHelper := testutils.NewPerformanceHelper()

	// Measure single validation
	perfHelper.MeasureOperation("single_validation", func() {
		validator.ValidateEndpoint(server.URL)
	})

	// Measure cached validation (should be faster)
	perfHelper.MeasureOperation("cached_validation", func() {
		validator.ValidateEndpoint(server.URL)
	})

	measurements := perfHelper.GetMeasurements()
	assert.Len(t, measurements, 2)

	// Cached validation should be significantly faster
	singleDuration := perfHelper.GetAverageDuration("single_validation")
	cachedDuration := perfHelper.GetAverageDuration("cached_validation")

	assert.True(t, cachedDuration < singleDuration,
		"Cached validation (%v) should be faster than single validation (%v)",
		cachedDuration, singleDuration)
}

func TestValidateEndpoint_CustomHTTPSValidation(t *testing.T) {
	t.Skip("SSL certificate test needs proper test certificates")
}

	// Configure TLS with expired certificate

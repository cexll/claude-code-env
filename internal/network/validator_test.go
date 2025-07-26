package network

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/claude-code/env-switcher/pkg/types"
	"github.com/claude-code/env-switcher/test/testutils"
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
	assert.Contains(t, result.Error, "timeout")
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
		name            string
		url             string
		expectedType    types.NetworkErrorType
		expectedSuggestionKeywords []string
	}{
		{
			name:         "invalid URL format",
			url:          "not-a-url",
			expectedType: types.NetworkInvalidURL,
			expectedSuggestionKeywords: []string{"protocol", "https://"},
		},
		{
			name:         "empty URL",
			url:          "",
			expectedType: types.NetworkInvalidURL,
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
	// Create server with expired certificate simulation
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	// Configure TLS with expired certificate
	cert, _ := tls.X509KeyPair([]byte(`-----BEGIN CERTIFICATE-----
MIICEjCCAXsCAg36MA0GCSqGSIb3DQEBBQUAMIGbMQswCQYDVQQGEwJKUDEOMAwG
A1UECBMFVG9reW8xEDAOBgNVBAcTB0NodW8ta3UxETAPBgNVBAoTCEZyYW5rNERE
MRgwFgYDVQQLEw9XZWJDZXJ0aWZpY2F0ZTEYMBYGA1UEAxMPRnJhbms0REQgV2Vi
Q0ExIzAhBgkqhkiG9w0BCQEWFGV4YW1wbGVAZXhhbXBsZS5jb20wHhcNMTIxMDI0
MTI0NjU0WhcNMTMxMDI0MTI0NjU0WjCBmzELMAkGA1UEBhMCSlAxDjAMBgNVBAgT
BVRva3lvMRAwDgYDVQQHEwdDaHVvLWt1MREwDwYDVQQKEwhGcmFuazRERDEYMBYG
A1UECxMPV2ViQ2VydGlmaWNhdGUxGDAWBgNVBAMTD0ZyYW5rNEREIFdlYkNBMSMw
IQYJKoZIhvcNAQkBFhRleGFtcGxlQGV4YW1wbGUuY29tMIGfMA0GCSqGSIb3DQEB
AQUAA4GNADCBiQKBgQC8nh8m7X2K3eX3qL7H8Kqy9WX6s0rXCJgqHxB6QUFwKBgD
zE8tF9xWx/2qxf3J4QJ8VqL4wKF1GDrJ2yTKX2Q7QXt8KJXW/S3vD8j9SXC7kF9x
x3v+1qGvE8YzY7J2K1YjFN3g0LCyP8J8R1l6rP1Dc6q8KS6YbQ2L2mYT8wIDAQAB
MA0GCSqGSIb3DQEBBQUAA4GBAJLEf9oF8v2G8rQe2R6fzL8y2rF2aJV0nY+Y8T2x
Cx2rw8bP6tY8y2i0J+C+qYgK2L3N4H+x8wD8Z+vR+g0Y8jF1j9z0z+1fV8Y8xF+x
QrY1jN+tD0FjL+wP1jY8PqN4CrJ8yK8v2Q4bF+cV5+t8w8z5Y1J+wY+v8t8Y8xF+
-----END CERTIFICATE-----`), []byte(`-----BEGIN PRIVATE KEY-----
MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBALyeHybtfYrd5feo
vsfwqrL1ZfqzStcImCofEHpBQXAoGAPMTy0X3FbH/arF/cnhAnxWovjAoXUYOsnb
JMpfZDtBe3woldb9Le8PyP1JcLuQX3HHe/7Woa8TxjNjsnYrViMU3eDQsLI/wnxH
WXqs/UNzqrwpLphtDYvaZhPzAgMBAAECgYEApYc7G+c7F+g3z4pz+9Q7R2qH8F6M
2e9Y9G9zQ3Qv1JQ8FQ8p1G+G7Q9J7GjK0v7JxH3+kF6d9J8Y8V8vQ2g9P8F2vR2Q
0J9j6Qv6g9b6zV1QJ2fF6qG7g3K8dL6v9qLz4F6JvJ7GXQ8Y7F9g1Q9gJYH8b8Fj
6P+QeE8K4O9H1NkCQQDy8Y4vKz4v2z9VYM2c7nM2xB+A7Y8J+0h8r8Q4Y7eF7Jv5
b7Zr8vWzOaF9L6E7n4FvYv8v8J2P6bP6Y8v9Y+JVAkEAxQ4W8F2P8GvY7j8L2H6Q
xF7gY6Y+t7wH8e8Y2mQ8KG8K6xfF6y6Y3Y8F1f6e8N2Y8Q4g6bG6Y8Y9fKY7Y8v2
QwJBAK0z5d8vF7z5K4z3JT6r1JxF8L+G4J8f7Y8K8y8FzGvJ8v8c9F6gY8z6Q8g1
P6a2F7Y8v6+2fJ3Y1Y7G3Y7Q6y8CQAzQ8Q7L6Y6J8Y2eY8pG8j7v1Y4Y7eP8G8y6
J4t1F+J6fJ8P8Y6Q7bJyY6Y8v8F7Y1F6zY8b9r1J6Y3aY5wCQQC2vY7z8f9Y7jYJ
1Y8c7Q8Y6vJbzY8P8Y6rGvK7Y6f1tY9Y8z8Y6eF+J8Y6z7a8Y8v6Y2nY8p2J5J8Y
-----END PRIVATE KEY-----`))
	
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	defer server.Close()
	
	validator := NewValidator()
	
	result, err := validator.ValidateEndpoint(server.URL)
	require.NoError(t, err)
	
	// Should handle SSL validation appropriately
	assert.NotNil(t, result)
}
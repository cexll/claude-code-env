package network

import (
	"fmt"
	"net/http"
	"time"

	"github.com/claude-code/env-switcher/pkg/types"
)

// performRequestWithRetry executes HTTP request with retry logic and exponential backoff.
//
// This method attempts the request up to maxRetries times, with increasing
// delays between attempts to handle transient network issues.
//
// Parameters:
//   - req: the HTTP request to execute
//
// Returns:
//   - *http.Response: successful HTTP response
//   - error: request error after all retry attempts
func (v *Validator) performRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error
	delay := v.retryDelay

	for attempt := 0; attempt < v.maxRetries; attempt++ {
		resp, err := v.client.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt < v.maxRetries-1 {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return nil, lastErr
}

// handleRequestError converts HTTP request errors to NetworkError with suggestions.
//
// This method analyzes the error type and provides actionable suggestions
// for common network connectivity issues.
//
// Parameters:
//   - err: the original request error
//   - url: the URL that failed
//
// Returns:
//   - error: NetworkError with diagnostic information and suggestions
func (v *Validator) handleRequestError(err error, url string) error {
	networkErr := &types.NetworkError{
		URL:   url,
		Cause: err,
	}

	// Analyze error type and provide specific suggestions
	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		networkErr.Type = types.NetworkTimeoutError
		networkErr.Message = "Network request timed out"
		networkErr.Suggestions = []string{
			"Check your internet connection",
			"The API server may be experiencing high load",
			"Try increasing the timeout duration",
			"Verify the endpoint URL is correct",
		}

	case contains(errStr, "connection refused"):
		networkErr.Type = types.NetworkConnectionFailed
		networkErr.Message = "Connection refused by server"
		networkErr.Suggestions = []string{
			"Verify the API endpoint URL is correct",
			"Check if the API service is running",
			"Ensure no firewall is blocking the connection",
			"Try again later if the service is temporarily unavailable",
		}

	case contains(errStr, "no such host"):
		networkErr.Type = types.NetworkUnreachable
		networkErr.Message = "DNS resolution failed - host not found"
		networkErr.Suggestions = []string{
			"Check the hostname in the URL for typos",
			"Verify your DNS settings",
			"Try using a different DNS server",
			"Check your internet connection",
		}

	case contains(errStr, "certificate"):
		networkErr.Type = types.NetworkSSLError
		networkErr.Message = "SSL certificate error"
		networkErr.Suggestions = []string{
			"Verify the SSL certificate is valid and not expired",
			"Check your system date and time settings",
			"Update your system's certificate authorities",
			"Contact the API provider about SSL certificate issues",
		}

	default:
		networkErr.Type = types.NetworkConnectionFailed
		networkErr.Message = fmt.Sprintf("Network request failed: %v", err)
		networkErr.Suggestions = []string{
			"Check your internet connection",
			"Verify the API endpoint URL is correct",
			"Try again in a few moments",
			"Contact your network administrator if the issue persists",
		}
	}

	return networkErr
}

// validateSSLCertificate validates SSL certificate details and updates result.
//
// This method examines the SSL certificate chain and records validation
// results including certificate expiry and issuer information.
//
// Parameters:
//   - resp: HTTP response containing SSL certificate information
//   - result: validation result to update with SSL information
func (v *Validator) validateSSLCertificate(resp *http.Response, result *types.NetworkValidationResult) {
	if resp.TLS == nil {
		result.SSLValid = false
		return
	}

	// Check if we have peer certificates
	if len(resp.TLS.PeerCertificates) == 0 {
		result.SSLValid = false
		return
	}

	cert := resp.TLS.PeerCertificates[0]
	result.SSLValid = true

	// Check certificate validity
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		result.SSLValid = false
	}
}

// TestAPIConnectivity tests API connectivity with authentication.
//
// This method performs a more comprehensive API test that includes
// authentication validation using the provided API key.
//
// Parameters:
//   - env: environment configuration containing URL and API key
//
// Returns:
//   - error: API connectivity error with suggestions
func (v *Validator) TestAPIConnectivity(env *types.Environment) error {
	if env == nil {
		return &types.NetworkError{
			Type:    types.NetworkInvalidURL,
			Message: "Environment configuration is nil",
			Suggestions: []string{
				"Provide a valid environment configuration",
			},
		}
	}

	// First validate the endpoint URL
	result, err := v.ValidateEndpoint(env.BaseURL)
	if err != nil {
		return err
	}

	if !result.Success {
		return &types.NetworkError{
			Type:    types.NetworkConnectionFailed,
			URL:     env.BaseURL,
			Message: fmt.Sprintf("API endpoint validation failed: %s", result.Error),
			Suggestions: []string{
				"Check the API endpoint URL",
				"Verify network connectivity",
				"Contact the API provider if the endpoint should be accessible",
			},
		}
	}

	// Test API authentication with a lightweight request
	return v.testAPIAuthentication(env)
}

// testAPIAuthentication tests API authentication using the provided credentials.
//
// This method makes a minimal authenticated request to verify that
// the API key is valid and properly configured.
//
// Parameters:
//   - env: environment configuration with API credentials
//
// Returns:
//   - error: authentication error with suggestions
func (v *Validator) testAPIAuthentication(env *types.Environment) error {
	// Create a test request to a common API endpoint
	req, err := http.NewRequest("GET", env.BaseURL, nil)
	if err != nil {
		return &types.NetworkError{
			Type:    types.NetworkRequestFailed,
			URL:     env.BaseURL,
			Message: fmt.Sprintf("Failed to create authentication test request: %v", err),
			Cause:   err,
		}
	}

	// Add authentication headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", env.APIKey))
	req.Header.Set("User-Agent", "Claude-Code-Environment-Switcher/1.0")

	// Add custom headers if specified
	for key, value := range env.Headers {
		req.Header.Set(key, value)
	}

	// Perform the request
	resp, err := v.client.Do(req)
	if err != nil {
		return v.handleRequestError(err, env.BaseURL)
	}
	defer resp.Body.Close()

	// Check authentication status
	if resp.StatusCode == 401 {
		return &types.NetworkError{
			Type:       types.NetworkAuthenticationError,
			URL:        env.BaseURL,
			Message:    "API authentication failed - invalid or expired API key",
			StatusCode: resp.StatusCode,
			Suggestions: []string{
				"Verify the API key is correct and properly formatted",
				"Check if the API key has expired",
				"Ensure the API key has the required permissions",
				"Contact the API provider to verify key status",
			},
		}
	}

	if resp.StatusCode == 403 {
		return &types.NetworkError{
			Type:       types.NetworkAuthenticationError,
			URL:        env.BaseURL,
			Message:    "API access forbidden - insufficient permissions",
			StatusCode: resp.StatusCode,
			Suggestions: []string{
				"Check if the API key has the required permissions",
				"Verify you're accessing the correct API endpoint",
				"Contact the API provider about access permissions",
			},
		}
	}

	return nil
}

// ClearCache removes all cached validation results.
//
// This method is useful for forcing fresh validation of all endpoints
// or for memory management in long-running applications.
func (v *Validator) ClearCache() {
	v.cache.mutex.Lock()
	defer v.cache.mutex.Unlock()

	v.cache.results = make(map[string]*CachedResult)
	v.cache.lastClean = time.Now()
}

// GetCacheStats returns statistics about the validation cache.
//
// This method provides information about cache usage and effectiveness
// for monitoring and debugging purposes.
//
// Returns:
//   - CacheStats: statistics about cached results
func (v *Validator) GetCacheStats() CacheStats {
	v.cache.mutex.RLock()
	defer v.cache.mutex.RUnlock()

	stats := CacheStats{
		TotalEntries: len(v.cache.results),
		LastCleanup:  v.cache.lastClean,
	}

	now := time.Now()
	for _, cached := range v.cache.results {
		if now.Sub(cached.Timestamp) < cached.TTL {
			stats.ValidEntries++
		} else {
			stats.ExpiredEntries++
		}
	}

	return stats
}

// CacheStats provides statistics about the validation cache.
type CacheStats struct {
	TotalEntries   int       `json:"total_entries"`
	ValidEntries   int       `json:"valid_entries"`
	ExpiredEntries int       `json:"expired_entries"`
	LastCleanup    time.Time `json:"last_cleanup"`
}

// contains checks if a string contains a substring (case-insensitive helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 (s[:len(substr)] == substr || 
		  s[len(s)-len(substr):] == substr || 
		  findInString(s, substr)))
}

// findInString searches for substring in string.
func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
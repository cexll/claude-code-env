// Package network provides network connectivity validation and testing capabilities
// for Claude Code Environment Switcher.
//
// This package handles network validation for API endpoints including:
// - HTTP/HTTPS connectivity testing
// - SSL certificate validation
// - Network diagnostic information
// - Result caching with TTL management
//
// Example usage:
//
//	validator := network.NewValidator()
//	result, err := validator.ValidateEndpoint("https://api.anthropic.com/v1")
//	if err != nil {
//		return fmt.Errorf("network validation failed: %w", err)
//	}
//
//	if !result.Success {
//		fmt.Printf("Endpoint unreachable: %s\n", result.Error)
//	}
package network

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
)

// Validator provides network connectivity validation with caching and SSL verification.
//
// The Validator performs comprehensive network tests including connectivity checks,
// SSL certificate validation, and diagnostic information collection.
type Validator struct {
	client     *http.Client
	cache      *ValidationCache
	timeout    time.Duration
	cacheTTL   time.Duration
	maxRetries int
	retryDelay time.Duration
	mutex      sync.RWMutex
}

// ValidationCache stores cached validation results with TTL management.
type ValidationCache struct {
	results    map[string]*CachedResult
	mutex      sync.RWMutex
	cleanupTTL time.Duration
	lastClean  time.Time
}

// CachedResult represents a cached validation result with timestamp.
type CachedResult struct {
	Result    *types.NetworkValidationResult
	Timestamp time.Time
	TTL       time.Duration
}

// NetworkDiagnostics provides detailed network diagnostic information.
type NetworkDiagnostics struct {
	DNSResolved    bool          `json:"dns_resolved"`
	ConnectionTime time.Duration `json:"connection_time,omitempty"`
	TLSHandshake   time.Duration `json:"tls_handshake,omitempty"`
	ServerIP       string        `json:"server_ip,omitempty"`
	CertExpiry     *time.Time    `json:"cert_expiry,omitempty"`
	CertIssuer     string        `json:"cert_issuer,omitempty"`
}

// NewValidator creates a new network validator with default configuration.
//
// The validator is initialized with:
// - 30 second timeout for network requests
// - 5 minute cache TTL for validation results
// - 3 retry attempts with exponential backoff
// - SSL certificate verification enabled
//
// Returns a configured Validator ready for use.
func NewValidator() *Validator {
	cache := &ValidationCache{
		results:    make(map[string]*CachedResult),
		cleanupTTL: 10 * time.Minute,
		lastClean:  time.Now(),
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Always verify SSL certificates
		},
		DisableKeepAlives:     false,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Validator{
		client:     client,
		cache:      cache,
		timeout:    30 * time.Second,
		cacheTTL:   5 * time.Minute,
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

// ValidateEndpoint performs comprehensive network validation for the given URL.
//
// This method checks:
// - URL format and accessibility
// - HTTP/HTTPS connectivity
// - SSL certificate validity (for HTTPS)
// - Response time and status codes
//
// Parameters:
//   - url: the endpoint URL to validate
//
// Returns:
//   - *types.NetworkValidationResult: comprehensive validation results
//   - error: validation error with actionable suggestions
//
// The result is cached for improved performance on subsequent calls.
func (v *Validator) ValidateEndpoint(urlStr string) (*types.NetworkValidationResult, error) {
	// Check cache first
	if cached := v.getCachedResult(urlStr); cached != nil {
		return cached, nil
	}

	// Perform validation
	result, err := v.performValidation(urlStr)
	if err != nil {
		return nil, err
	}

	// Cache the result
	v.cacheResult(urlStr, result)

	return result, nil
}

// ValidateEndpointWithTimeout performs network validation with custom timeout.
//
// This method allows specifying a custom timeout for the validation,
// overriding the default validator timeout.
//
// Parameters:
//   - url: the endpoint URL to validate
//   - timeout: custom timeout duration
//
// Returns:
//   - *types.NetworkValidationResult: comprehensive validation results
//   - error: validation error with suggestions
func (v *Validator) ValidateEndpointWithTimeout(urlStr string, timeout time.Duration) (*types.NetworkValidationResult, error) {
	// Temporarily modify client timeout
	originalTimeout := v.client.Timeout
	v.client.Timeout = timeout
	defer func() { v.client.Timeout = originalTimeout }()

	return v.ValidateEndpoint(urlStr)
}

// performValidation executes the actual network validation logic.
//
// This is a focused function that handles the core validation process
// without caching concerns, keeping it under 50 lines.
func (v *Validator) performValidation(urlStr string) (*types.NetworkValidationResult, error) {
	start := time.Now()
	result := &types.NetworkValidationResult{
		Timestamp: start,
	}

	// Validate URL format
	parsedURL, err := v.validateURLFormat(urlStr)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Perform HTTP request with diagnostics
	err = v.performHTTPRequest(parsedURL, result)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	result.Success = true
	return result, nil
}

// validateURLFormat validates the URL format and scheme.
//
// Ensures the URL is properly formatted and uses HTTP or HTTPS scheme.
// Returns parsed URL or error with suggestions.
func (v *Validator) validateURLFormat(urlStr string) (*url.URL, error) {
	if urlStr == "" {
		return nil, &types.NetworkError{
			Type:    types.NetworkInvalidURL,
			URL:     urlStr,
			Message: "URL cannot be empty",
			Suggestions: []string{
				"Provide a valid HTTP or HTTPS URL",
				"Example: https://api.anthropic.com/v1",
			},
		}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, &types.NetworkError{
			Type:    types.NetworkInvalidURL,
			URL:     urlStr,
			Message: fmt.Sprintf("Invalid URL format: %v", err),
			Cause:   err,
			Suggestions: []string{
				"Check the URL for typos or formatting errors",
				"Ensure the URL includes the protocol (http:// or https://)",
				"Example: https://api.anthropic.com/v1",
			},
		}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, &types.NetworkError{
			Type:    types.NetworkInvalidURL,
			URL:     urlStr,
			Message: "URL must use http or https scheme",
			Suggestions: []string{
				"Use http:// or https:// at the beginning of the URL",
				"HTTPS is recommended for security",
				"Example: https://api.anthropic.com/v1",
			},
		}
	}

	if parsedURL.Host == "" {
		return nil, &types.NetworkError{
			Type:    types.NetworkInvalidURL,
			URL:     urlStr,
			Message: "URL must have a valid host",
			Suggestions: []string{
				"Ensure the URL includes a hostname",
				"Example: https://api.anthropic.com/v1",
			},
		}
	}

	return parsedURL, nil
}

// performHTTPRequest executes the HTTP request and collects diagnostics.
//
// This function handles the actual network request, SSL validation,
// and diagnostic information collection.
func (v *Validator) performHTTPRequest(parsedURL *url.URL, result *types.NetworkValidationResult) error {
	// Create HEAD request for connectivity testing
	req, err := http.NewRequest("HEAD", parsedURL.String(), nil)
	if err != nil {
		return &types.NetworkError{
			Type:    types.NetworkRequestFailed,
			URL:     parsedURL.String(),
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Cause:   err,
			Suggestions: []string{
				"Check if the URL is valid and accessible",
				"Verify network connectivity",
			},
		}
	}

	// Set User-Agent to identify CCE
	req.Header.Set("User-Agent", "Claude-Code-Environment-Switcher/1.0")

	// Perform request with retry logic
	resp, err := v.performRequestWithRetry(req)
	if err != nil {
		return v.handleRequestError(err, parsedURL.String())
	}
	defer resp.Body.Close()

	// Record response details
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode < 400

	// Validate SSL certificate for HTTPS
	if parsedURL.Scheme == "https" {
		v.validateSSLCertificate(resp, result)
	} else {
		result.SSLValid = false // HTTP doesn't use SSL
	}

	return nil
}

// getCachedResult retrieves a cached validation result if valid.
//
// Checks the cache for existing validation results that haven't expired.
// Returns nil if no valid cached result exists.
func (v *Validator) getCachedResult(url string) *types.NetworkValidationResult {
	v.cache.mutex.RLock()
	defer v.cache.mutex.RUnlock()

	if cached, exists := v.cache.results[url]; exists {
		if time.Since(cached.Timestamp) < cached.TTL {
			return cached.Result
		}
		// Result expired, will be cleaned up later
	}

	return nil
}

// cacheResult stores a validation result in the cache.
//
// Stores the result with timestamp and TTL for future retrieval.
// Also triggers cache cleanup if needed.
func (v *Validator) cacheResult(url string, result *types.NetworkValidationResult) {
	v.cache.mutex.Lock()
	defer v.cache.mutex.Unlock()

	v.cache.results[url] = &CachedResult{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       v.cacheTTL,
	}

	// Trigger cache cleanup if needed
	if time.Since(v.cache.lastClean) > v.cache.cleanupTTL {
		go v.cleanupCache()
	}
}

// cleanupCache removes expired entries from the validation cache.
//
// This function runs periodically to prevent memory leaks from
// accumulated expired cache entries.
func (v *Validator) cleanupCache() {
	v.cache.mutex.Lock()
	defer v.cache.mutex.Unlock()

	now := time.Now()
	for url, cached := range v.cache.results {
		if now.Sub(cached.Timestamp) > cached.TTL {
			delete(v.cache.results, url)
		}
	}

	v.cache.lastClean = now
}

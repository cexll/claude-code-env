package validation

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
)

// ModelValidator interface defines enhanced model validation operations
type ModelValidator interface {
	ValidateModelName(model string) (*ModelValidationResult, error)
	ValidateModelWithAPI(env *types.Environment, model string) (*ModelValidationResult, error)
	GetSuggestedModels(apiType string) ([]string, error)
	CacheValidationResult(key string, result *ModelValidationResult)
	ClearCache()
}

// ModelValidationResult contains comprehensive model validation results
type ModelValidationResult struct {
	Valid           bool                   `json:"valid"`
	Model           string                 `json:"model"`
	APICompatible   bool                   `json:"api_compatible"`
	Suggestions     []string               `json:"suggestions"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CachedResult    bool                   `json:"cached_result"`
	ValidatedAt     time.Time              `json:"validated_at"`
	PerformanceData *ValidationPerformance `json:"performance_data,omitempty"`
}

// ValidationPerformance tracks timing data for validation operations
type ValidationPerformance struct {
	PatternCheckTime time.Duration `json:"pattern_check_time"`
	APICheckTime     time.Duration `json:"api_check_time"`
	TotalTime        time.Duration `json:"total_time"`
}

// EnhancedModelValidator implements the ModelValidator interface with caching and API validation
type EnhancedModelValidator struct {
	patternValidator PatternValidator
	apiValidator     APIValidator
	cache            *ValidationCache
	metrics          *ValidationMetrics
	mu               sync.RWMutex
}

// PatternValidator interface for pattern-based validation
type PatternValidator interface {
	Validate(model string) *ModelValidationResult
}

// APIValidator interface for API connectivity validation
type APIValidator interface {
	ValidateModel(env *types.Environment, model string) (*APIValidationResult, error)
}

// APIValidationResult contains API validation specific results
type APIValidationResult struct {
	Compatible            bool          `json:"compatible"`
	SuggestedAlternatives []string      `json:"suggested_alternatives"`
	ResponseTime          time.Duration `json:"response_time"`
	StatusCode            int           `json:"status_code"`
	ErrorMessage          string        `json:"error_message,omitempty"`
}

// ValidationCache provides TTL-based caching for validation results
type ValidationCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration
}

// CacheEntry represents a cached validation result
type CacheEntry struct {
	Result    *ModelValidationResult
	ExpiresAt time.Time
}

// ValidationMetrics tracks validation performance and cache statistics
type ValidationMetrics struct {
	PatternValidations  int64         `json:"pattern_validations"`
	APIValidations      int64         `json:"api_validations"`
	CacheHits           int64         `json:"cache_hits"`
	CacheMisses         int64         `json:"cache_misses"`
	TotalValidationTime time.Duration `json:"total_validation_time"`
	mutex               sync.RWMutex
}

// NewEnhancedModelValidator creates a new enhanced model validator
func NewEnhancedModelValidator() *EnhancedModelValidator {
	return &EnhancedModelValidator{
		patternValidator: NewBasicPatternValidator(),
		apiValidator:     NewHTTPAPIValidator(),
		cache:            NewValidationCache(15 * time.Minute), // 15 minute TTL
		metrics:          &ValidationMetrics{},
	}
}

// ValidateModelName performs pattern-based validation only
func (emv *EnhancedModelValidator) ValidateModelName(model string) (*ModelValidationResult, error) {
	start := time.Now()
	defer func() {
		emv.metrics.mutex.Lock()
		emv.metrics.PatternValidations++
		emv.metrics.TotalValidationTime += time.Since(start)
		emv.metrics.mutex.Unlock()
	}()

	// Check cache first
	if cached := emv.cache.Get(model, "pattern"); cached != nil {
		emv.metrics.mutex.Lock()
		emv.metrics.CacheHits++
		emv.metrics.mutex.Unlock()
		return cached, nil
	}

	emv.metrics.mutex.Lock()
	emv.metrics.CacheMisses++
	emv.metrics.mutex.Unlock()

	// Perform pattern validation
	patternStart := time.Now()
	result := emv.patternValidator.Validate(model)
	patternTime := time.Since(patternStart)

	result.PerformanceData = &ValidationPerformance{
		PatternCheckTime: patternTime,
		TotalTime:        patternTime,
	}
	result.ValidatedAt = time.Now()

	// Cache the result
	emv.cache.Set(model, "pattern", result)

	return result, nil
}

// ValidateModelWithAPI performs both pattern and API connectivity validation
func (emv *EnhancedModelValidator) ValidateModelWithAPI(env *types.Environment, model string) (*ModelValidationResult, error) {
	start := time.Now()
	defer func() {
		emv.metrics.mutex.Lock()
		emv.metrics.APIValidations++
		emv.metrics.TotalValidationTime += time.Since(start)
		emv.metrics.mutex.Unlock()
	}()

	// Check cache first
	if cached := emv.cache.Get(model, env.BaseURL); cached != nil {
		emv.metrics.mutex.Lock()
		emv.metrics.CacheHits++
		emv.metrics.mutex.Unlock()
		return cached, nil
	}

	emv.metrics.mutex.Lock()
	emv.metrics.CacheMisses++
	emv.metrics.mutex.Unlock()

	// Pattern validation first
	patternStart := time.Now()
	patternResult := emv.patternValidator.Validate(model)
	patternTime := time.Since(patternStart)

	if !patternResult.Valid {
		// If pattern validation fails, return early
		patternResult.PerformanceData = &ValidationPerformance{
			PatternCheckTime: patternTime,
			TotalTime:        patternTime,
		}
		patternResult.ValidatedAt = time.Now()
		return patternResult, nil
	}

	// Optional API validation
	apiStart := time.Now()
	apiResult, err := emv.apiValidator.ValidateModel(env, model)
	apiTime := time.Since(apiStart)

	if err != nil {
		// API validation failed, but pattern validation passed
		// Return pattern result with warning
		patternResult.ErrorMessage = fmt.Sprintf("API validation failed: %v", err)
		patternResult.APICompatible = false
		patternResult.PerformanceData = &ValidationPerformance{
			PatternCheckTime: patternTime,
			APICheckTime:     apiTime,
			TotalTime:        time.Since(start),
		}
		patternResult.ValidatedAt = time.Now()

		// Cache the result even if API validation failed
		emv.cache.Set(model, env.BaseURL, patternResult)
		return patternResult, nil
	}

	// Combine results
	result := &ModelValidationResult{
		Valid:         patternResult.Valid && apiResult.Compatible,
		Model:         model,
		APICompatible: apiResult.Compatible,
		Suggestions:   apiResult.SuggestedAlternatives,
		ValidatedAt:   time.Now(),
		PerformanceData: &ValidationPerformance{
			PatternCheckTime: patternTime,
			APICheckTime:     apiTime,
			TotalTime:        time.Since(start),
		},
	}

	if !apiResult.Compatible {
		result.ErrorMessage = apiResult.ErrorMessage
		if len(apiResult.SuggestedAlternatives) == 0 {
			result.Suggestions = patternResult.Suggestions
		}
	}

	// Cache result
	emv.cache.Set(model, env.BaseURL, result)

	return result, nil
}

// GetSuggestedModels returns model suggestions based on API type
func (emv *EnhancedModelValidator) GetSuggestedModels(apiType string) ([]string, error) {
	// Default Claude API models
	defaultModels := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	// Could be extended to support different API types
	switch apiType {
	case "anthropic", "claude", "":
		return defaultModels, nil
	default:
		return defaultModels, nil
	}
}

// CacheValidationResult manually caches a validation result
func (emv *EnhancedModelValidator) CacheValidationResult(key string, result *ModelValidationResult) {
	// Parse key to extract model and endpoint
	// For now, use key as model and empty endpoint
	emv.cache.Set(key, "", result)
}

// ClearCache clears all cached validation results
func (emv *EnhancedModelValidator) ClearCache() {
	emv.cache.Clear()
}

// GetMetrics returns current validation metrics
func (emv *EnhancedModelValidator) GetMetrics() *ValidationMetrics {
	emv.metrics.mutex.RLock()
	defer emv.metrics.mutex.RUnlock()

	return &ValidationMetrics{
		PatternValidations:  emv.metrics.PatternValidations,
		APIValidations:      emv.metrics.APIValidations,
		CacheHits:           emv.metrics.CacheHits,
		CacheMisses:         emv.metrics.CacheMisses,
		TotalValidationTime: emv.metrics.TotalValidationTime,
	}
}

// NewValidationCache creates a new validation cache with specified TTL
func NewValidationCache(ttl time.Duration) *ValidationCache {
	cache := &ValidationCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached validation result
func (vc *ValidationCache) Get(model, endpoint string) *ModelValidationResult {
	key := vc.makeKey(model, endpoint)

	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	entry, exists := vc.entries[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		delete(vc.entries, key)
		return nil
	}

	// Mark as cached result
	result := *entry.Result
	result.CachedResult = true
	return &result
}

// Set stores a validation result in the cache
func (vc *ValidationCache) Set(model, endpoint string, result *ModelValidationResult) {
	key := vc.makeKey(model, endpoint)

	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	vc.entries[key] = &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(vc.ttl),
	}
}

// Clear removes all entries from the cache
func (vc *ValidationCache) Clear() {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	vc.entries = make(map[string]*CacheEntry)
}

// cleanup removes expired entries from the cache
func (vc *ValidationCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		vc.mutex.Lock()
		now := time.Now()
		for key, entry := range vc.entries {
			if now.After(entry.ExpiresAt) {
				delete(vc.entries, key)
			}
		}
		vc.mutex.Unlock()
	}
}

// makeKey creates a cache key from model and endpoint
func (vc *ValidationCache) makeKey(model, endpoint string) string {
	if endpoint == "" {
		return model
	}
	return fmt.Sprintf("%s:%s", model, endpoint)
}

// BasicPatternValidator implements pattern-based validation
type BasicPatternValidator struct {
	supportedModels []string
}

// NewBasicPatternValidator creates a basic pattern validator
func NewBasicPatternValidator() *BasicPatternValidator {
	return &BasicPatternValidator{
		supportedModels: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		},
	}
}

// Validate performs pattern-based validation
func (bpv *BasicPatternValidator) Validate(model string) *ModelValidationResult {
	if model == "" {
		return &ModelValidationResult{
			Valid:       true,
			Model:       model,
			Suggestions: bpv.supportedModels[:3], // Top 3 suggestions
		}
	}

	// Check exact match
	for _, supported := range bpv.supportedModels {
		if supported == model {
			return &ModelValidationResult{
				Valid:       true,
				Model:       model,
				Suggestions: []string{},
			}
		}
	}

	// Model not in known list, but may still be valid
	return &ModelValidationResult{
		Valid:        false,
		Model:        model,
		ErrorMessage: fmt.Sprintf("Model '%s' is not in the known model list", model),
		Suggestions:  bpv.getSimilarModels(model),
	}
}

// getSimilarModels returns similar model suggestions
func (bpv *BasicPatternValidator) getSimilarModels(model string) []string {
	// Simple similarity check
	suggestions := []string{}
	modelLower := strings.ToLower(model)

	for _, supported := range bpv.supportedModels {
		if strings.Contains(strings.ToLower(supported), "sonnet") && strings.Contains(modelLower, "sonnet") {
			suggestions = append(suggestions, supported)
		} else if strings.Contains(strings.ToLower(supported), "haiku") && strings.Contains(modelLower, "haiku") {
			suggestions = append(suggestions, supported)
		} else if strings.Contains(strings.ToLower(supported), "opus") && strings.Contains(modelLower, "opus") {
			suggestions = append(suggestions, supported)
		}
	}

	if len(suggestions) == 0 {
		return bpv.supportedModels[:3]
	}

	return suggestions
}

// HTTPAPIValidator implements API connectivity validation
type HTTPAPIValidator struct {
	client *http.Client
}

// NewHTTPAPIValidator creates a new HTTP API validator
func NewHTTPAPIValidator() *HTTPAPIValidator {
	return &HTTPAPIValidator{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
	}
}

// ValidateModel performs API connectivity validation
func (hav *HTTPAPIValidator) ValidateModel(env *types.Environment, model string) (*APIValidationResult, error) {
	start := time.Now()

	// Create a simple request to test model availability
	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		fmt.Sprintf("%s/v1/messages", env.BaseURL),
		strings.NewReader(`{"model":"`+model+`","max_tokens":1,"messages":[{"role":"user","content":"test"}]}`),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.APIKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := hav.client.Do(req)
	if err != nil {
		return &APIValidationResult{
			Compatible:   false,
			ResponseTime: time.Since(start),
			ErrorMessage: fmt.Sprintf("API request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	result := &APIValidationResult{
		ResponseTime: time.Since(start),
		StatusCode:   resp.StatusCode,
	}

	// Check response status
	switch resp.StatusCode {
	case http.StatusOK:
		result.Compatible = true
	case http.StatusBadRequest:
		// Parse error response to check if it's model-related
		var errorResp map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(map[string]interface{}); ok {
				if message, ok := errorMsg["message"].(string); ok {
					if strings.Contains(strings.ToLower(message), "model") {
						result.Compatible = false
						result.ErrorMessage = fmt.Sprintf("Model not supported: %s", message)
						result.SuggestedAlternatives = []string{
							"claude-3-5-sonnet-20241022",
							"claude-3-5-haiku-20241022",
							"claude-3-opus-20240229",
						}
					} else {
						// Other bad request, likely API key or format issue
						result.Compatible = true // Model might be OK, other issue
						result.ErrorMessage = fmt.Sprintf("API error (not model-related): %s", message)
					}
				}
			}
		}
	case http.StatusUnauthorized:
		result.Compatible = true // Model might be OK, auth issue
		result.ErrorMessage = "Authentication failed - check API key"
	case http.StatusForbidden:
		result.Compatible = true // Model might be OK, permission issue
		result.ErrorMessage = "Access denied - check API permissions"
	default:
		result.Compatible = false
		result.ErrorMessage = fmt.Sprintf("API returned status %d", resp.StatusCode)
	}

	return result, nil
}

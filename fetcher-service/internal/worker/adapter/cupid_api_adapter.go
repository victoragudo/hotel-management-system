package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/dto"
	"golang.org/x/time/rate"
)

type CupidAPIAdapter struct {
	client         *http.Client
	baseURL        string
	apiKey         string
	rateLimiter    *rate.Limiter
	circuitBreaker *gobreaker.CircuitBreaker
	retryConfig    *retryConfig
	timeout        time.Duration
	maxRetries     int
	retryInterval  time.Duration
	headers        map[string]string
}

type retryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	Multiplier    float64
	Jitter        bool
	RetryableCode []int
}

type APIConfig struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	RateLimit      float64
	BurstLimit     int
	MaxRetries     int
	RetryInterval  time.Duration
	Headers        map[string]string
	CircuitBreaker *CircuitBreakerConfig
}

type CircuitBreakerConfig struct {
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
	ReadyToTrip func(counts gobreaker.Counts) bool
}

func NewCupidAPIAdapter(config *APIConfig) *CupidAPIAdapter {
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	rateLimiter := rate.NewLimiter(rate.Limit(config.RateLimit), config.BurstLimit)

	cbSettings := gobreaker.Settings{
		Name:          "cupid-api",
		MaxRequests:   config.CircuitBreaker.MaxRequests,
		Interval:      config.CircuitBreaker.Interval,
		Timeout:       config.CircuitBreaker.Timeout,
		ReadyToTrip:   config.CircuitBreaker.ReadyToTrip,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {},
	}

	if cbSettings.ReadyToTrip == nil {
		cbSettings.ReadyToTrip = func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		}
	}

	retryConfig := &retryConfig{
		MaxRetries:    config.MaxRetries,
		BaseDelay:     config.RetryInterval,
		MaxDelay:      30 * time.Second,
		Multiplier:    2.0,
		Jitter:        true,
		RetryableCode: []int{429, 500, 502, 503, 504},
	}

	return &CupidAPIAdapter{
		client:         client,
		baseURL:        config.BaseURL,
		apiKey:         config.APIKey,
		rateLimiter:    rateLimiter,
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
		retryConfig:    retryConfig,
		timeout:        config.Timeout,
		maxRetries:     config.MaxRetries,
		retryInterval:  config.RetryInterval,
		headers:        config.Headers,
	}
}

func (c *CupidAPIAdapter) FetchHotelData(ctx context.Context, hotelId int64) (*dto.HotelAPIResponse, error) {
	url := fmt.Sprintf("%s/property/%d", c.baseURL, hotelId)

	var response dto.HotelAPIResponse
	err := c.makeRequest(ctx, http.MethodGet, url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch hotel data for ID %d: %w", hotelId, err)
	}

	return &response, nil
}

func (c *CupidAPIAdapter) FetchHotelReviews(ctx context.Context, hotelID int64, options *dto.ReviewFetchOptions) (*dto.ReviewDataList, error) {
	reviewCount := int64(50)
	if options != nil && options.ReviewCount > 0 {
		reviewCount = options.ReviewCount
	}

	url := fmt.Sprintf("%s/property/reviews/%d/%d", c.baseURL, hotelID, reviewCount)

	var reviewDataList dto.ReviewDataList
	err := c.makeRequest(ctx, "GET", url, nil, &reviewDataList)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reviews for hotel ID %d: %w", hotelID, err)
	}

	return &reviewDataList, nil
}

func (c *CupidAPIAdapter) FetchTranslations(ctx context.Context, hotelID string, options *dto.TranslationFetchOptions) (*dto.TranslationAPIResponse, error) {

	if options == nil || options.Lang == "" {
		return nil, fmt.Errorf("lang is required")
	}

	url := fmt.Sprintf("%s/property/%s/lang/%s", c.baseURL, hotelID, options.Lang)

	var response dto.TranslationAPIResponse
	err := c.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch translations for hotel ID %s: %w", hotelID, err)
	}

	return &response, nil
}

func (c *CupidAPIAdapter) makeRequest(ctx context.Context, method, url string, body any, response any) error {
	return c.executeWithRetry(ctx, func() error {
		return c.performRequest(ctx, method, url, body, response)
	})
}

func (c *CupidAPIAdapter) performRequest(ctx context.Context, method, url string, body any, response any) error {
	err := c.rateLimiter.Wait(ctx)
	if err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}

	result, err := c.circuitBreaker.Execute(func() (any, error) {
		result, httpErr := c.doHTTPRequest(ctx, method, url, body, response)

		// If it's a 404 error, we don't want it to count as a circuit breaker failure,
		// So we return a success result but with the 404 error wrapped in a special way
		if c.is404Error(httpErr) {
			return &notFoundResult{err: httpErr}, nil
		}

		return result, httpErr
	})

	if err != nil {
		return err
	}

	// Check if we got a 404 result that was wrapped
	if nfResult, ok := result.(*notFoundResult); ok {
		return nfResult.err
	}

	if response != nil && result != nil {
		return nil
	}

	return nil
}

// notFoundResult is a wrapper to indicate a 404 error that shouldn't affect the circuit breaker
type notFoundResult struct {
	err error
}

func (c *CupidAPIAdapter) doHTTPRequest(ctx context.Context, method, url string, requestBody any, response any) (any, error) {
	var bodyReader io.Reader

	if requestBody != nil {
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("accept", "application/json")
	request.Header.Set("x-api-key", c.apiKey)

	for key, value := range c.headers {
		request.Header.Set(key, value)
	}

	httpResponse, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %s", err.Error())
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(httpResponse.Body)

	if httpResponse.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResponse.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", httpResponse.StatusCode, string(body))
	}

	if response != nil {
		err = json.NewDecoder(httpResponse.Body).Decode(response)
		if err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return response, nil
}

func (c *CupidAPIAdapter) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateRetryDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
		if !c.isRetryableError(err) {
			break
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
}

func (c *CupidAPIAdapter) calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.retryConfig.BaseDelay) * float64(attempt) * c.retryConfig.Multiplier)

	if delay > c.retryConfig.MaxDelay {
		delay = c.retryConfig.MaxDelay
	}

	if c.retryConfig.Jitter {
		jitter := time.Duration(float64(delay) * 0.1)
		delay += time.Duration(float64(jitter) * (2*float64(time.Now().UnixNano()%1000)/1000 - 1))
	}

	return delay
}

func (c *CupidAPIAdapter) is404Error(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check if it's a 404 HTTP error
	if strings.Contains(errStr, "HTTP error") {
		// Extract status code from error message "HTTP error %d: %s"
		parts := strings.Split(errStr, ":")
		if len(parts) >= 1 {
			httpPart := strings.TrimSpace(parts[0])
			// Extract the number after "HTTP error"
			if strings.HasPrefix(httpPart, "HTTP error ") {
				statusStr := strings.TrimPrefix(httpPart, "HTTP error ")
				if statusCode, parseErr := strconv.Atoi(statusStr); parseErr == nil {
					return statusCode == 404
				}
			}
		}
	}

	return false
}

func (c *CupidAPIAdapter) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check if it's an HTTP error
	if strings.Contains(errStr, "HTTP error") {
		// Extract status code from error message "HTTP error %d: %s"
		parts := strings.Split(errStr, ":")
		if len(parts) >= 1 {
			httpPart := strings.TrimSpace(parts[0])
			// Extract the number after "HTTP error"
			if strings.HasPrefix(httpPart, "HTTP error ") {
				statusStr := strings.TrimPrefix(httpPart, "HTTP error ")
				if statusCode, parseErr := strconv.Atoi(statusStr); parseErr == nil {
					// Don't retry 404 Not Found errors
					if statusCode == 404 {
						return false
					}
					// Only retry specific HTTP status codes
					for _, retryableCode := range c.retryConfig.RetryableCode {
						if statusCode == retryableCode {
							return true
						}
					}
					return false
				}
			}
		}
	}

	// For non-HTTP errors (network errors, timeouts, etc.), allow retries
	return true
}

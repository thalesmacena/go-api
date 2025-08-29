package http

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	charsetpkg "golang.org/x/net/html/charset"
)

// BackoffMode represents the type of backoff strategy
type BackoffMode string

const (
	// ExponentialBackoff doubles the delay on each retry
	ExponentialBackoff BackoffMode = "exponential"
	// FixedBackoff uses the same delay for all retries
	FixedBackoff BackoffMode = "fixed"
)

// BackoffConfig represents the retry configuration with backoff strategy
type BackoffConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// RetryStatusCodes is the list of HTTP status codes that trigger retries
	RetryStatusCodes []int
	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration
	// Mode determines the backoff strategy (exponential or fixed)
	Mode BackoffMode
}

// Client represents an HTTP client with configuration options.
type Client struct {
	baseURL            string
	client             *http.Client
	followRedirect     bool
	dismiss404         bool
	defaultHeaders     map[string]string
	defaultContentType string
	defaultBackoff     *BackoffConfig
	logger             HTTPLogger
}

// ClientOptions represents the configuration options for the HTTP client.
type ClientOptions struct {
	FollowRedirect      bool
	Dismiss404          bool
	DefaultHeaders      map[string]string
	DefaultContentType  string
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	ConnectionTimeout   time.Duration
	ReadTimeout         time.Duration
	DefaultBackoff      *BackoffConfig
	Logger              HTTPLogger
}

// NewHttpClient creates a new HTTP client with the given base URL and configuration options.
func NewHttpClient(baseURL string, opts ClientOptions) *Client {
	if opts.MaxIdleConns == 0 {
		opts.MaxIdleConns = 200
	}
	if opts.MaxIdleConnsPerHost == 0 {
		opts.MaxIdleConnsPerHost = 20
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 60 * time.Second
	}
	if opts.ConnectionTimeout == 0 {
		opts.ConnectionTimeout = 60 * time.Second
	}
	if opts.DefaultContentType == "" {
		opts.DefaultContentType = "application/json"
	}

	transport := &http.Transport{
		MaxIdleConns:        opts.MaxIdleConns,
		MaxIdleConnsPerHost: opts.MaxIdleConnsPerHost,
		IdleConnTimeout:     opts.IdleConnTimeout,
		DialContext: (&net.Dialer{
			Timeout: opts.ConnectionTimeout,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   opts.ReadTimeout,
	}

	if !opts.FollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Client{
		baseURL:            strings.TrimRight(baseURL, "/"),
		client:             client,
		followRedirect:     opts.FollowRedirect,
		dismiss404:         opts.Dismiss404,
		defaultHeaders:     opts.DefaultHeaders,
		defaultContentType: opts.DefaultContentType,
		defaultBackoff:     opts.DefaultBackoff,
		logger:             opts.Logger,
	}
}

// Request creates a new Request object for the client.
func (hc *Client) Request() *Request {
	return NewHttpClientRequest(hc)
}

// Get sends a GET request to the specified path with optional query parameters, headers, and response types.
// It returns the success response, error response, status code, and error if any.
func (hc *Client) Get(path string, queryParams map[string]string, headers map[string]string, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequestWithBackoff(http.MethodGet, path, queryParams, headers, nil, successResp, errorResp, nil)
}

// Post sends a POST request to the specified path with optional query parameters, headers, and response types.
// It returns the success response, error response, status code, and error if any.
func (hc *Client) Post(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequestWithBackoff(http.MethodPost, path, queryParams, headers, body, successResp, errorResp, nil)
}

// Put sends a PUT request to the specified path with optional query parameters, headers, and response types.
// It returns the success response, error response, status code, and error if any.
func (hc *Client) Put(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequestWithBackoff(http.MethodPut, path, queryParams, headers, body, successResp, errorResp, nil)
}

// Patch sends a PATCH request to the specified path with optional query parameters, headers, and response types.
// It returns the success response, error response, status code, and error if any.
func (hc *Client) Patch(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequestWithBackoff(http.MethodPatch, path, queryParams, headers, body, successResp, errorResp, nil)
}

// Delete sends a DELETE request to the specified path with optional query parameters, headers, and response types.
// It returns the success response, error response, status code, and error if any.
func (hc *Client) Delete(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequestWithBackoff(http.MethodDelete, path, queryParams, headers, body, successResp, errorResp, nil)
}

// doRequestWithBackoff handles HTTP requests with retry logic and backoff strategy.
// It uses the provided backoff configuration or falls back to the client's default configuration.
// If no backoff configuration is provided, it executes the request without retries.
func (hc *Client) doRequestWithBackoff(method, path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any, requestBackoff *BackoffConfig) (any, any, int, error) {
	// Determine which backoff configuration to use
	backoffConfig := requestBackoff
	if backoffConfig == nil {
		backoffConfig = hc.defaultBackoff
	}

	// If no backoff configuration is available, execute request without retries
	if backoffConfig == nil {
		return hc.doRequest(method, path, queryParams, headers, body, successResp, errorResp)
	}

	var lastSuccessResp, lastErrorResp any
	var lastStatusCode int
	var lastErr error

	// Execute the initial request
	lastSuccessResp, lastErrorResp, lastStatusCode, lastErr = hc.doRequest(method, path, queryParams, headers, body, successResp, errorResp)

	// Check if the request was successful or if we should not retry
	if lastErr == nil || !hc.shouldRetry(lastStatusCode, backoffConfig.RetryStatusCodes) {
		return lastSuccessResp, lastErrorResp, lastStatusCode, lastErr
	}

	// Retry with backoff strategy
	delay := backoffConfig.InitialDelay
	for attempt := 1; attempt <= backoffConfig.MaxRetries; attempt++ {
		// Log retry attempt if logger is available
		if hc.logger != nil {
			url := hc.buildURL(path)
			if len(queryParams) > 0 {
				url += "?" + buildQueryString(queryParams)
			}
			bodyString := hc.bodyToString(body)
			allHeaders := make(map[string]string)
			for k, v := range hc.defaultHeaders {
				allHeaders[k] = v
			}
			for k, v := range headers {
				allHeaders[k] = v
			}
			responseBodyString := ""
			if lastErrorResp != nil {
				if jsonBytes, err := json.Marshal(lastErrorResp); err == nil {
					responseBodyString = string(jsonBytes)
				}
			}
			hc.logger.LogRequestRetry(method, url, allHeaders, bodyString, lastStatusCode, responseBodyString, 0, lastErr, attempt, backoffConfig.MaxRetries)
		}

		// Wait for the calculated delay
		time.Sleep(delay)

		// Execute the retry request
		lastSuccessResp, lastErrorResp, lastStatusCode, lastErr = hc.doRequest(method, path, queryParams, headers, body, successResp, errorResp)

		// Check if the request was successful or if we should not retry
		if lastErr == nil || !hc.shouldRetry(lastStatusCode, backoffConfig.RetryStatusCodes) {
			return lastSuccessResp, lastErrorResp, lastStatusCode, lastErr
		}

		// Calculate next delay based on backoff mode
		if backoffConfig.Mode == ExponentialBackoff {
			delay *= 2
		}
		// For FixedBackoff, delay remains the same
	}

	// Return the last attempt result
	return lastSuccessResp, lastErrorResp, lastStatusCode, lastErr
}

// shouldRetry determines if a request should be retried based on the status code
func (hc *Client) shouldRetry(statusCode int, retryStatusCodes []int) bool {
	for _, code := range retryStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// bodyToString converts a request body to string for logging purposes
func (hc *Client) bodyToString(body any) string {
	if body == nil {
		return ""
	}

	switch b := body.(type) {
	case string:
		return b
	case []byte:
		return string(b)
	default:
		// Try to marshal to JSON for logging
		if jsonBytes, err := json.Marshal(body); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", body)
	}
}

// doRequest is a helper function that sends an HTTP request with the given method, path, query parameters, headers, body, success response, and error response.
// It builds the URL, prepares the request body, sets headers, executes the request, and handles the response.
// It returns the success response, error response, status code, and error if any.
func (hc *Client) doRequest(method, path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	url := hc.buildURL(path)
	if len(queryParams) > 0 {
		url += "?" + buildQueryString(queryParams)
	}

	// Prepare logging data
	bodyString := hc.bodyToString(body)
	allHeaders := make(map[string]string)
	for k, v := range hc.defaultHeaders {
		allHeaders[k] = v
	}
	for k, v := range headers {
		allHeaders[k] = v
	}

	// Log request before sending
	if hc.logger != nil {
		hc.logger.LogRequest(method, url, allHeaders, bodyString)
	}

	// Prepare request body
	var bodyReader io.Reader
	var contentType string

	if body != nil {
		switch body := body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(body)
			contentType = "text/plain"
		case []byte:
			bodyReader = bytes.NewBuffer(body)
			contentType = "application/octet-stream"
		default:
			// Use default content type from client options
			contentType = hc.defaultContentType

			switch contentType {
			case "application/json":
				jsonBody, err := json.Marshal(body)
				if err != nil {
					return nil, nil, 0, fmt.Errorf("failed to marshal request body to JSON: %w", err)
				}
				bodyReader = bytes.NewBuffer(jsonBody)
			case "application/xml":
				xmlBody, err := xml.Marshal(body)
				if err != nil {
					return nil, nil, 0, fmt.Errorf("failed to marshal request body to XML: %w", err)
				}
				bodyReader = bytes.NewBuffer(xmlBody)
			case "text/plain":
				// Convert to string representation
				bodyReader = bytes.NewBufferString(fmt.Sprintf("%v", body))
			default:
				// Fallback to JSON for unknown content types
				jsonBody, err := json.Marshal(body)
				if err != nil {
					return nil, nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
				}
				bodyReader = bytes.NewBuffer(jsonBody)
				contentType = "application/json"
			}
		}
	}

	// Build request
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, nil, 0, err
	}

	// Set headers
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
		allHeaders["Content-Type"] = contentType
	}

	// Set headers
	for k, v := range hc.defaultHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute request and measure latency
	start := time.Now()
	resp, err := hc.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		// Log error if logger is available
		if hc.logger != nil {
			hc.logger.LogResponseError(method, url, allHeaders, bodyString, 0, "", latency, err)
		}
		return nil, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the Response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// Log error if logger is available
		if hc.logger != nil {
			hc.logger.LogResponseError(method, url, allHeaders, bodyString, resp.StatusCode, "", latency, err)
		}
		return nil, nil, resp.StatusCode, err
	}

	// Determine response content type
	respContentType := resp.Header.Get("Content-Type")
	if respContentType == "" {
		respContentType = hc.defaultContentType
	}

	// Convert response body to string for logging
	responseBodyString := string(bodyBytes)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Log successful response
		if hc.logger != nil {
			hc.logger.LogResponseSuccess(method, url, allHeaders, bodyString, resp.StatusCode, responseBodyString, latency)
		}

		if successResp != nil {
			err = hc.unmarshalResponse(bodyBytes, respContentType, successResp)
			if err != nil {
				return nil, nil, resp.StatusCode, err
			}
		}
		return successResp, nil, resp.StatusCode, nil
	}

	if resp.StatusCode == 404 && hc.dismiss404 {
		// Log successful response for dismissed 404
		if hc.logger != nil {
			hc.logger.LogResponseSuccess(method, url, allHeaders, bodyString, resp.StatusCode, responseBodyString, latency)
		}
		return nil, nil, resp.StatusCode, nil
	}

	// Log error response
	httpErr := fmt.Errorf("http error: status %d", resp.StatusCode)
	if hc.logger != nil {
		hc.logger.LogResponseError(method, url, allHeaders, bodyString, resp.StatusCode, responseBodyString, latency, httpErr)
	}

	if errorResp != nil {
		err = hc.unmarshalResponse(bodyBytes, respContentType, errorResp)
		if err != nil {
			return nil, nil, resp.StatusCode, err
		}
	}

	return nil, errorResp, resp.StatusCode, httpErr
}

// unmarshalResponse unmarshals response body based on content type
func (hc *Client) unmarshalResponse(bodyBytes []byte, contentType string, target any) error {
	// Extract the main content type (remove charset and other parameters)
	mainContentType := strings.Split(contentType, ";")[0]
	mainContentType = strings.TrimSpace(mainContentType)

	switch mainContentType {
	case "application/json":
		return json.Unmarshal(bodyBytes, target)
	case "application/xml", "text/xml":
		dec := xml.NewDecoder(bytes.NewReader(bodyBytes))
		dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
			return charsetpkg.NewReaderLabel(charset, input)
		}
		return dec.Decode(target)
	case "text/plain":
		// For text/plain, try to set the value directly if target is a string pointer
		if strPtr, ok := target.(*string); ok {
			*strPtr = string(bodyBytes)
			return nil
		}
		// Fallback to JSON unmarshaling for non-string targets
		return json.Unmarshal(bodyBytes, target)
	case "application/octet-stream":
		// For binary data, try to set the value directly if target is a byte slice pointer
		if bytePtr, ok := target.(*[]byte); ok {
			*bytePtr = bodyBytes
			return nil
		}
		// Fallback to JSON unmarshaling for non-byte targets
		return json.Unmarshal(bodyBytes, target)
	default:
		// Default to JSON unmarshaling for unknown content types
		return json.Unmarshal(bodyBytes, target)
	}
}

// buildURL builds a normalized URL by properly handling baseURL and path
func (hc *Client) buildURL(path string) string {
	// Ensure path starts with "/" only if path is not empty
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Normalize baseURL to not end with "/"
	baseURL := strings.TrimRight(hc.baseURL, "/")

	// Combine baseURL and path
	return baseURL + path
}

// buildQueryString builds a query string from parameters
func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(parts, "&")
}

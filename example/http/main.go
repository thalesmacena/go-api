package main

import (
	"encoding/xml"
	"fmt"
	"go-api/pkg/http"
	"go-api/pkg/log"
	"strings"
	"time"
)

type PokeResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Height int    `json:"height"`
	Weight int    `json:"weight"`
	Order  int    `json:"order"`
}

type APIErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Errors  []APIErrorDetail `json:"errors"`
	Status  string           `json:"status"`
}

type APIErrorDetail struct {
	Message string `json:"message"`
	Domain  string `json:"domain"`
	Reason  string `json:"reason"`
}

type CreatePostRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

type CreatePostResponse struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// XML structure for httpbin /xml example
// https://httpbin.io/xml returns a <slideshow> root with attributes
// We map only a few attributes for demonstration
// Full structure is not required for successful unmarshaling
// and unknown fields are ignored by encoding/xml

type HttpbinXML struct {
	XMLName xml.Name `xml:"slideshow"`
	Title   string   `xml:"title,attr"`
	Date    string   `xml:"date,attr"`
	Author  string   `xml:"author,attr"`
}

// XML POST request payload for /anything
// The response from /anything will be JSON echoing the raw body in the "data" field

type XMLAnythingRequest struct {
	XMLName xml.Name `xml:"note"`
	To      string   `xml:"to"`
	From    string   `xml:"from"`
	Heading string   `xml:"heading"`
	Body    string   `xml:"body"`
}

type HttpbinAnythingResponse struct {
	Data    string              `json:"data"`
	Headers map[string][]string `json:"headers"`
	URL     string              `json:"url"`
}

// StandardHTTPLogger implements HTTPLogger interface using pkg/log library (Zap-based structured logging)
type StandardHTTPLogger struct{}

// NewStandardHTTPLogger creates a new StandardHTTPLogger
func NewStandardHTTPLogger() *StandardHTTPLogger {
	return &StandardHTTPLogger{}
}

// LogRequest logs the request before it's sent
func (l *StandardHTTPLogger) LogRequest(method, url string, headers map[string]string, body string) {
	maskedHeaders := l.maskSensitiveHeaders(headers)
	log.Infow("HTTP REQUEST", "method", method, "url", url, "headers", maskedHeaders, "body", body)
}

// LogResponseSuccess logs successful responses
func (l *StandardHTTPLogger) LogResponseSuccess(method, url string, headers map[string]string, body string, httpStatus int, responseBody string, latency int64) {
	log.Infow("HTTP SUCCESS", "method", method, "url", url, "status", httpStatus, "latency", latency, "response", responseBody)
}

// LogResponseError logs error responses
func (l *StandardHTTPLogger) LogResponseError(method, url string, headers map[string]string, body string, httpStatus int, responseBody string, latency int64, err error) {
	log.Errorw("HTTP ERROR", "method", method, "url", url, "status", httpStatus, "latency", latency, "error", err, "response", responseBody)
}

// LogRequestRetry logs retry attempts
func (l *StandardHTTPLogger) LogRequestRetry(method, url string, headers map[string]string, body string, httpStatus int, responseBody string, latency int64, err error, retryCount, maxRetries int) {
	log.Infow("HTTP RETRY", "method", method, "url", url, "attempt", retryCount, "maxRetries", maxRetries, "previousStatus", httpStatus, "error", err)
}

// maskSensitiveHeaders creates a copy of headers with sensitive values masked
func (l *StandardHTTPLogger) maskSensitiveHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return headers
	}

	masked := make(map[string]string)
	for k, v := range headers {
		// Mask sensitive headers
		if strings.ToLower(k) == "authorization" || strings.ToLower(k) == "cookie" || strings.ToLower(k) == "x-api-key" {
			masked[k] = "***"
		} else {
			masked[k] = v
		}
	}
	return masked
}

func main() {
	// ===========================================
	// DEFAULT EXAMPLES
	// ===========================================

	// Client Options with JSON as default content type
	clientOptions := http.ClientOptions{
		FollowRedirect:     true,
		Dismiss404:         false,
		DefaultHeaders:     map[string]string{"Authorization": "Bearer token"},
		DefaultContentType: "application/json",
	}
	fmt.Println("=== Starting Default Configuration Examples ===")
	// Creating a Client
	client := http.NewHttpClient("https://pokeapi.co/api/v2", clientOptions)

	// Success Request
	fmt.Println("=== Example 1: Success Request ===")
	success, failure, status, err := client.Get("/pokemon/ditto", nil, nil, &PokeResponse{}, nil)

	if err != nil {
		fmt.Println("Request Error - status:", status, "error:", err, "responseBody:", failure)
	} else {
		fmt.Println("Request Success - status:", status, "responseBody:", success)
	}

	// Creating other Client

	client = http.NewHttpClient("https://www.googleapis.com/youtube/v3", clientOptions)

	queryParams := map[string]string{
		"q": "test",
	}

	// Error Request
	fmt.Println("=== Example 2: Error Request ===")
	success, failure, status, err = client.Get("search", queryParams, nil, nil, &APIErrorResponse{})

	if err != nil {
		fmt.Println("Request Error - status:", status, "error:", err, "responseBody:", failure)
	} else {
		fmt.Println("Request Success - status:", status, "responseBody:", success)
	}

	// Creating Client For Post with JSON
	client = http.NewHttpClient(
		"https://jsonplaceholder.typicode.com",
		clientOptions,
	)

	reqBody := CreatePostRequest{
		Title:  "foo",
		Body:   "bar",
		UserID: 1,
	}

	// Success Request
	fmt.Println("=== Example 3: Success Request with reqBody ===")
	success, failure, status, err = client.Post("/posts", nil, nil, reqBody, &CreatePostResponse{}, nil)

	if err != nil {
		fmt.Println("Request Error - status:", status, "error:", err, "responseBody:", failure)
	} else {
		fmt.Println("Request Success - status:", status, "responseBody:", success)
	}

	// Using Request Builder
	fmt.Println("=== Example 4: Success Request with Request Builder ===")
	requestSuccessBody, requestErrorBody, requestStatus, requestErr := client.Request().
		WithMethod(http.POST).
		WithPath("/posts").
		WithBody(reqBody).
		WithSuccessResp(&CreatePostResponse{}).
		Execute()

	if requestErr != nil {
		fmt.Println("Request Error - status:", requestStatus, "error:", requestErr, "responseBody:", requestErrorBody)
	} else {
		fmt.Println("Request Success - status:", requestStatus, "responseBody:", requestSuccessBody)
	}

	// Example: Public XML API using httpbin
	fmt.Println("=== Example 5: XML Success Request ===")
	xmlClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "application/xml"})
	var xmlResp HttpbinXML
	xmlSuccess, xmlFailure, xmlStatus, xmlErr := xmlClient.Get("/xml", nil, nil, &xmlResp, nil)
	if xmlErr != nil {
		fmt.Println("XML Request Error - status:", xmlStatus, "error:", xmlErr, "responseBody:", xmlFailure)
	} else {
		fmt.Println("XML Request Success - status:", xmlStatus, "responseBody:", xmlSuccess)
	}

	// Example: XML POST to /xml (httpbin returns static XML)
	fmt.Println("=== Example 6: XML Success POST Request ===")
	xmlPostClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "application/xml"})
	xmlReq := XMLAnythingRequest{To: "Alice", From: "Bob", Heading: "Reminder", Body: "Meeting at 10"}
	var xmlPostResp HttpbinXML
	xmlPostSuccess, xmlPostFailure, xmlPostStatus, xmlPostErr := xmlPostClient.Post("/xml", nil, nil, xmlReq, &xmlPostResp, nil)
	if xmlPostErr != nil {
		fmt.Println("XML POST Error - status:", xmlPostStatus, "error:", xmlPostErr, "responseBody:", xmlPostFailure)
	} else {
		fmt.Println("XML POST Success - status:", xmlPostStatus, "responseBody:", xmlPostSuccess)
	}

	// Example: Public text/plain endpoint (robots.txt) using httpbin
	fmt.Println("=== Example 7: Text Response Success Request ===")
	textClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "text/plain"})
	var textResponse string
	textSuccess, textFailure, textStatus, textErr := textClient.Get("/robots.txt", nil, nil, &textResponse, nil)
	if textErr != nil {
		fmt.Println("Text Request Error - status:", textStatus, "error:", textErr, "responseBody:", textFailure)
	} else {
		fmt.Println("Text Request Success - status:", textStatus, "responseBody:", textSuccess)
	}

	// Example: Public binary endpoint using httpbin (random bytes)
	fmt.Println("=== Example 8: Binary Response Success Request ===")
	binaryClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "application/octet-stream"})
	var binaryResponse []byte
	binarySuccess, binaryFailure, binaryStatus, binaryErr := binaryClient.Get("/bytes/16", nil, nil, &binaryResponse, nil)
	if binaryErr != nil {
		fmt.Println("Binary Request Error - status:", binaryStatus, "error:", binaryErr, "responseBody:", binaryFailure)
	} else {
		fmt.Println("Binary Request Success - status:", binaryStatus, "responseBody:", binarySuccess)
	}

	// ===========================================
	// BACKOFF EXAMPLES
	// ===========================================

	fmt.Println("=== Starting Backoff Configuration Examples ===")

	// Example 1: HTTP Client with default exponential backoff and logger
	fmt.Println("=== Example 1: Client with Default Exponential Backoff and Logger ===")

	// Create HTTP logger using pkg/log library
	httpLogger := NewStandardHTTPLogger()

	clientWithDefaultBackoff := http.NewHttpClient("https://httpbin.org", http.ClientOptions{
		DefaultBackoff: &http.BackoffConfig{
			MaxRetries:       3,
			RetryStatusCodes: []int{500, 502, 503, 504, 429}, // Server errors and rate limiting
			InitialDelay:     1 * time.Second,
			Mode:             http.ExponentialBackoff,
		},
		Logger: httpLogger, // Add the logger to the client
	})

	// This will use the client's default backoff configuration
	successResp, errorResp, statusCode, err := clientWithDefaultBackoff.Get("/status/500", nil, nil, nil, nil)
	fmt.Println("Backoff Response - success:", successResp, "error:", errorResp, "status:", statusCode, "err:", err)

	// Example 2: HTTP Client without default backoff, but with per-request backoff
	fmt.Println("=== Example 2: Per-Request Fixed Backoff Override ===")
	clientWithoutBackoff := http.NewHttpClient("https://httpbin.org", http.ClientOptions{})

	// Use the Request builder to override with fixed backoff
	successResp, errorResp, statusCode, err = clientWithoutBackoff.Request().
		WithMethod(http.GET).
		WithPath("/status/503").
		WithBackoff(&http.BackoffConfig{
			MaxRetries:       2,
			RetryStatusCodes: []int{503},
			InitialDelay:     500 * time.Millisecond,
			Mode:             http.FixedBackoff, // Fixed delay, not exponential
		}).
		Execute()

	fmt.Println("Per-Request Backoff Response - success:", successResp, "error:", errorResp, "status:", statusCode, "err:", err)

	// Example 3: HTTP Client with default backoff, but override per request
	fmt.Println("=== Example 3: Override Default Backoff Per Request ===")

	// Override the client's default backoff with a different configuration
	successResp, errorResp, statusCode, err = clientWithDefaultBackoff.Request().
		WithMethod(http.GET).
		WithPath("/status/502").
		WithBackoff(&http.BackoffConfig{
			MaxRetries:       1, // Less retries than default
			RetryStatusCodes: []int{502},
			InitialDelay:     2 * time.Second,   // Longer initial delay
			Mode:             http.FixedBackoff, // Different mode
		}).
		Execute()

	fmt.Println("Override Backoff Response - success:", successResp, "error:", errorResp, "status:", statusCode, "err:", err)

	// Example 4: Successful request (should not trigger retries)
	fmt.Println("=== Example 5: Successful Request (No Retries Needed) ===")

	var response map[string]interface{}
	successResp, errorResp, statusCode, err = clientWithDefaultBackoff.Get("/get", nil, nil, &response, nil)
	fmt.Println("Successful Request Response - success:", successResp != nil, "error:", errorResp, "status:", statusCode, "err:", err)
	if response != nil {
		fmt.Println("Response Details - url:", response["url"])
	}

	// Example 6: Demonstrate different logging scenarios
	fmt.Println("=== Example 6: Logger Demonstration (Success, Error, and Retry) ===")

	// Test successful request (no retries needed)
	fmt.Println("--- Testing Successful Request ---")
	successResp, errorResp, statusCode, err = clientWithDefaultBackoff.Get("/get", nil, nil, &response, nil)

	// Test request that will trigger retries
	fmt.Println("--- Testing Request with Retries ---")
	successResp, errorResp, statusCode, err = clientWithDefaultBackoff.Get("/status/503", nil, nil, nil, nil)

	// Test request with custom headers and body (for logging demonstration)
	fmt.Println("--- Testing POST Request with Body and Headers ---")
	customHeaders := map[string]string{
		"X-Custom-Header": "test-value",
		"Authorization":   "Bearer secret-token", // This will be masked in logs
	}
	postBody := map[string]interface{}{
		"message":   "This is a test request body for logging demonstration",
		"timestamp": time.Now().Unix(),
	}
	successResp, errorResp, statusCode, err = clientWithDefaultBackoff.Post("/post", nil, customHeaders, postBody, nil, nil)

	fmt.Println("=== Backoff and Logging Examples Completed ===")
}

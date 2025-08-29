package http

// HTTPLogger interface defines methods for logging HTTP requests and responses
type HTTPLogger interface {
	// LogRequest is called before the request is sent with all request data formed
	LogRequest(method, url string, headers map[string]string, body string)
	
	// LogResponseSuccess is called immediately after receiving a successful response (non-error HTTP status)
	LogResponseSuccess(method, url string, headers map[string]string, body string, httpStatus int, responseBody string, latency int64)
	
	// LogResponseError is called immediately after receiving an error response (error HTTP status)
	LogResponseError(method, url string, headers map[string]string, body string, httpStatus int, responseBody string, latency int64, err error)
	
	// LogRequestRetry is called when backoff exists and a retry attempt is about to be made
	LogRequestRetry(method, url string, headers map[string]string, body string, httpStatus int, responseBody string, latency int64, err error, retryCount, maxRetries int)
}
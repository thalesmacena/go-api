package http

import "fmt"

// RequestMethod represents the HTTP method for the request.
type RequestMethod string

const (
	GET    RequestMethod = "GET"
	POST   RequestMethod = "POST"
	PATCH  RequestMethod = "PATCH"
	PUT    RequestMethod = "PUT"
	DELETE RequestMethod = "DELETE"
)

// Request represents an HTTP request with various configuration options.
type Request struct {
	requestClient      *Client
	requestMethod      RequestMethod
	requestPath        string
	requestQueryParams map[string]string
	requestHeaders     map[string]string
	requestBody        any
	requestSuccessResp any
	requestErrorResp   any
	requestBackoff     *BackoffConfig
}

// NewHttpClientRequest creates a new Request object with the given client.
func NewHttpClientRequest(client *Client) *Request {
	return &Request{
		requestClient: client,
		requestMethod: GET,
		requestPath:   "/",
	}
}

// WithClient sets the client for the request.
func (r *Request) WithClient(client *Client) *Request {
	r.requestClient = client
	return r
}

// WithMethod sets the HTTP method for the request.
func (r *Request) WithMethod(method RequestMethod) *Request {
	r.requestMethod = method
	return r
}

// WithPath sets the path for the request.
func (r *Request) WithPath(path string) *Request {
	r.requestPath = path
	return r
}

// WithQueryParams sets the query parameters for the request.
func (r *Request) WithQueryParams(params map[string]string) *Request {
	r.requestQueryParams = params
	return r
}

// WithHeaders sets the headers for the request.
func (r *Request) WithHeaders(headers map[string]string) *Request {
	r.requestHeaders = headers
	return r
}

// WithBody sets the body for the request.
func (r *Request) WithBody(body any) *Request {
	r.requestBody = body
	return r
}

// WithSuccessResp sets the success response for the request.
func (r *Request) WithSuccessResp(successResp any) *Request {
	r.requestSuccessResp = successResp
	return r
}

// WithErrorResp sets the error response for the request.
func (r *Request) WithErrorResp(errorResp any) *Request {
	r.requestErrorResp = errorResp
	return r
}

// WithBackoff sets the backoff configuration for the request, overriding the client's default.
func (r *Request) WithBackoff(backoff *BackoffConfig) *Request {
	r.requestBackoff = backoff
	return r
}

// Execute sends the request and returns the success response, error response, status code, and error if any.
func (r *Request) Execute() (any, any, int, error) {
	if r.requestClient == nil {
		return nil, nil, 0, fmt.Errorf("client is required")
	}
	if r.requestMethod == "" {
		return nil, nil, 0, fmt.Errorf("method is required")
	}
	if r.requestPath == "" {
		return nil, nil, 0, fmt.Errorf("path is required")
	}

	return r.requestClient.doRequestWithBackoff(
		string(r.requestMethod),
		r.requestPath,
		r.requestQueryParams,
		r.requestHeaders,
		r.requestBody,
		r.requestSuccessResp,
		r.requestErrorResp,
		r.requestBackoff,
	)
}

package http

import "fmt"

type RequestMethod string

const (
	GET    RequestMethod = "GET"
	POST   RequestMethod = "POST"
	PATCH  RequestMethod = "PATCH"
	PUT    RequestMethod = "PUT"
	DELETE RequestMethod = "DELETE"
)

type Request struct {
	requestClient      *Client
	requestMethod      RequestMethod
	requestPath        string
	requestQueryParams map[string]string
	requestHeaders     map[string]string
	requestBody        any
	requestSuccessResp any
	requestErrorResp   any
}

func NewHttpClientRequest(client *Client) *Request {
	return &Request{
		requestClient: client,
	}
}

func (r *Request) WithClient(client *Client) *Request {
	r.requestClient = client
	return r
}

func (r *Request) WithMethod(method RequestMethod) *Request {
	r.requestMethod = method
	return r
}

func (r *Request) WithPath(path string) *Request {
	r.requestPath = path
	return r
}

func (r *Request) WithQueryParams(params map[string]string) *Request {
	r.requestQueryParams = params
	return r
}

func (r *Request) WithHeaders(headers map[string]string) *Request {
	r.requestHeaders = headers
	return r
}

func (r *Request) WithBody(body any) *Request {
	r.requestBody = body
	return r
}

func (r *Request) WithSuccessResp(successResp any) *Request {
	r.requestSuccessResp = successResp
	return r
}

func (r *Request) WithErrorResp(errorResp any) *Request {
	r.requestErrorResp = errorResp
	return r
}

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

	return r.requestClient.doRequest(
		string(r.requestMethod),
		r.requestPath,
		r.requestQueryParams,
		r.requestHeaders,
		r.requestBody,
		r.requestSuccessResp,
		r.requestErrorResp,
	)
}

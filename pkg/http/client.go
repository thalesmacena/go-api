package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL        string
	client         *http.Client
	followRedirect bool
	dismiss404     bool
	defaultHeaders map[string]string
}

type ClientOptions struct {
	FollowRedirect      bool
	Dismiss404          bool
	DefaultHeaders      map[string]string
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	ConnectionTimeout   time.Duration
	ReadTimeout         time.Duration
}

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
		baseURL:        strings.TrimRight(baseURL, "/"),
		client:         client,
		followRedirect: opts.FollowRedirect,
		dismiss404:     opts.Dismiss404,
		defaultHeaders: opts.DefaultHeaders,
	}
}

func (hc *Client) Request() *Request {
	return NewHttpClientRequest(hc)
}

func (hc *Client) Get(path string, queryParams map[string]string, headers map[string]string, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequest(http.MethodGet, path, queryParams, headers, nil, successResp, errorResp)
}

func (hc *Client) Post(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequest(http.MethodPost, path, queryParams, headers, body, successResp, errorResp)
}

func (hc *Client) Put(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequest(http.MethodPut, path, queryParams, headers, body, successResp, errorResp)
}

func (hc *Client) Patch(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequest(http.MethodPatch, path, queryParams, headers, body, successResp, errorResp)
}

func (hc *Client) Delete(path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	return hc.doRequest(http.MethodDelete, path, queryParams, headers, body, successResp, errorResp)
}

func (hc *Client) doRequest(method, path string, queryParams map[string]string, headers map[string]string, body any, successResp any, errorResp any) (any, any, int, error) {
	fullURL := hc.baseURL + "/" + strings.TrimLeft(path, "/")
	reqURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, nil, 0, err
	}

	// Add query params
	q := reqURL.Query()
	for k, v := range queryParams {
		q.Set(k, v)
	}
	reqURL.RawQuery = q.Encode()

	// Prepare body
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, 0, err
		}
		bodyReader = bytes.NewBuffer(b)
	}

	// Build request
	req, err := http.NewRequest(method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, nil, 0, err
	}

	// Set headers
	for k, v := range hc.defaultHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := hc.client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.StatusCode, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if successResp != nil {
			err = json.Unmarshal(bodyBytes, successResp)
			if err != nil {
				return nil, nil, resp.StatusCode, err
			}
		}
		return successResp, nil, resp.StatusCode, nil
	}

	if resp.StatusCode == 404 && hc.dismiss404 {
		return nil, nil, resp.StatusCode, nil
	}

	if errorResp != nil {
		err = json.Unmarshal(bodyBytes, errorResp)
		if err != nil {
			return nil, nil, resp.StatusCode, err
		}
	}

	return nil, errorResp, resp.StatusCode, fmt.Errorf("http error: status %d", resp.StatusCode)
}

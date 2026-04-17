package fetcht

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
)

// Response contains the decoded response body and metadata from the HTTP response.
type Response[R any] struct {
	Data          R
	StatusCode    int
	Status        string
	Proto         string
	Header        http.Header
	Cookies       []*http.Cookie
	Request       *http.Request
	ContentLength int64
	RawBody       []byte
}

type RequestOptions struct {
	path        string
	queryParams map[string]string
	headers     map[string]string
	encoder     Encoder
	decoders    map[string]Decoder
}

// RequestOption is a functional option for configuring an individual request.
type RequestOption func(options *RequestOptions)

// WithPath appends an endpoint path to the client's baseURL for this request.
func WithPath(path string) RequestOption {
	return func(r *RequestOptions) {
		r.path = path
	}
}

// WithQueryParams adds a query string key-value pair to the request URL.
func WithQueryParams(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.queryParams[key] = value
	}
}

// WithHeaders adds a per-request header, overriding any client-level header with the same key.
func WithHeaders(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.headers[key] = value
	}
}

// WithEncoder sets the content type encoder for this request, this will also
// set the content-type header value, any user added content-type headers
// will be overridden by the encoder.
func WithEncoder(encoder Encoder) RequestOption {
	return func(r *RequestOptions) {
		r.encoder = encoder
	}
}

// WithDecoder adds a decoder, or overrides a registered decoder in the internal decoder
// registry.
func WithDecoder(contentType string, decoder Decoder) RequestOption {
	return func(c *RequestOptions) {
		c.decoders[contentType] = decoder
	}
}

// Get performs an HTTP GET request and decodes response body into R
func Get[R any](ctx context.Context, client *Client, opts ...RequestOption) (*Response[R], error) {
	return doWithoutBody[R](ctx, client, http.MethodGet, opts...)
}

// Delete performs an HTTP DELETE request, encoding any response into R
func Delete[R any](ctx context.Context, client *Client, opts ...RequestOption) (*Response[R], error) {
	return doWithoutBody[R](ctx, client, http.MethodDelete, opts...)
}

// Post performs an HTTP POST request, encoding request of T and decoding response into R
func Post[T any, R any](ctx context.Context, client *Client, request T, opts ...RequestOption) (*Response[R], error) {
	return doWithBody[T, R](ctx, client, http.MethodPost, request, opts...)
}

// Put performs an HTTP PUT request, encoding request of T and decoding response into R
func Put[T any, R any](ctx context.Context, client *Client, request T, opts ...RequestOption) (*Response[R], error) {
	return doWithBody[T, R](ctx, client, http.MethodPut, request, opts...)
}

// Patch performs an HTTP PATCH request, encoding request of T and decoding response into R
func Patch[T any, R any](ctx context.Context, client *Client, request T, opts ...RequestOption) (*Response[R], error) {
	return doWithBody[T, R](ctx, client, http.MethodPatch, request, opts...)
}

// There are effectively two distinct code flows for all supported HTTP requests, broken into 4 funcs for minimal
// code repetition
func doWithoutBody[R any](ctx context.Context, client *Client, method string, opts ...RequestOption) (*Response[R], error) {
	reqOptions := &RequestOptions{
		queryParams: make(map[string]string),
		headers:     make(map[string]string),
		decoders: map[string]Decoder{
			"application/json": JSONDecoder,
			"application/xml":  XMLDecoder,
			"text/xml":         XMLDecoder,
		},
	}
	for _, opt := range opts {
		opt(reqOptions)
	}

	req, err := buildRequest(ctx, client, method, nil, "", reqOptions)
	if err != nil {
		return nil, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return handleResponse[R](resp, reqOptions)
}

func doWithBody[T any, R any](ctx context.Context, client *Client, method string, request T, opts ...RequestOption) (*Response[R], error) {
	reqOptions := &RequestOptions{
		queryParams: make(map[string]string),
		headers:     make(map[string]string),
		encoder:     JSONEncoder,
		decoders: map[string]Decoder{
			"application/json": JSONDecoder,
			"application/xml":  XMLDecoder,
			"text/xml":         XMLDecoder,
		},
	}
	for _, opt := range opts {
		opt(reqOptions)
	}

	bodyReader, contentType, err := reqOptions.encoder.Encode(request)
	if err != nil {
		return nil, err
	}

	req, err := buildRequest(ctx, client, method, bodyReader, contentType, reqOptions)
	if err != nil {
		return nil, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return handleResponse[R](resp, reqOptions)
}

func handleResponse[R any](resp *http.Response, reqOptions *RequestOptions) (*Response[R], error) {
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Err:        fmt.Errorf("failed to read HTTP response body: %w", err),
		}
	}

	response := &Response[R]{
		StatusCode:    resp.StatusCode,
		Status:        resp.Status,
		Proto:         resp.Proto,
		Header:        resp.Header,
		Cookies:       resp.Cookies(),
		Request:       resp.Request,
		ContentLength: resp.ContentLength,
		RawBody:       b,
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
		}
	}

	if resp.StatusCode == http.StatusNoContent {
		return response, nil
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType == "" {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        errors.New("server response did not include a Content-Type header"),
		}
	}
	if err != nil {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        fmt.Errorf("failed to parse content type from server response: %w", err),
		}
	}

	decoder, ok := reqOptions.decoders[mediaType]
	if !ok {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        fmt.Errorf("no registered decoders for %s, register one using WithDecoder()", mediaType),
		}
	}

	if err := decoder.Decode(bytes.NewReader(b), &response.Data); err != nil {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        fmt.Errorf("failed to decode response body for status %d: %w", resp.StatusCode, err),
		}
	}

	return response, nil
}

func buildRequest(ctx context.Context, client *Client, methodType string, body io.Reader, contentType string, reqOptions *RequestOptions) (*http.Request, error) {
	rawURL, err := url.JoinPath(client.baseURL, reqOptions.path)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for key, value := range reqOptions.queryParams {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, methodType, u.String(), body)
	if err != nil {
		return nil, err
	}

	for key, value := range client.headers {
		req.Header.Set(key, value)
	}
	for key, value := range reqOptions.headers {
		req.Header.Set(key, value)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

// HTTPError encapsulates any errors from the fetchT library
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
	Err        error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("HTTP error: %d %s: %s: %v", e.StatusCode, e.Status, string(e.Body), e.Err)
	}
	return fmt.Sprintf("HTTP error: %d %s: %s", e.StatusCode, e.Status, e.Body)
}

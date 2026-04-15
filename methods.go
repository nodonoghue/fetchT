package fetcht

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Get performs an HTTP GET request and decodes response body into R
func Get[R any](ctx context.Context, client *Client, opts ...RequestOption) (R, error) {
	return doWithoutBody[R](ctx, client, http.MethodGet, opts...)
}

// Delete performs an HTTP DELETE request, encoding any response into R
func Delete[R any](ctx context.Context, client *Client, opts ...RequestOption) (R, error) {
	return doWithoutBody[R](ctx, client, http.MethodDelete, opts...)
}

// Post performs an HTTP POST request, encoding request of T and decoding response into R
func Post[T any, R any](ctx context.Context, client *Client, request T, opts ...RequestOption) (R, error) {
	return doWithBody[T, R](ctx, client, http.MethodPost, request, opts...)
}

// Put performs an HTTP PUT request, encoding request of T and decoding response into R
func Put[T any, R any](ctx context.Context, client *Client, request T, opts ...RequestOption) (R, error) {
	return doWithBody[T, R](ctx, client, http.MethodPut, request, opts...)
}

// Patch performs an HTTP PATCH request, encoding request of T and decoding response into R
func Patch[T any, R any](ctx context.Context, client *Client, request T, opts ...RequestOption) (R, error) {
	return doWithBody[T, R](ctx, client, http.MethodPatch, request, opts...)
}

// There are effectively two distinct code flows for all supported HTTP requests, broken into 3 funcs for minimal
// code repetition
func doWithoutBody[R any](ctx context.Context, client *Client, method string, opts ...RequestOption) (R, error) {
	return doRequest[R](ctx, client, method, nil, "", opts...)
}

func doWithBody[T any, R any](ctx context.Context, client *Client, method string, request T, opts ...RequestOption) (R, error) {
	var response R

	bodyReader, contentType, err := client.encoder.Encode(request)
	if err != nil {
		return response, err
	}

	return doRequest[R](ctx, client, method, bodyReader, contentType, opts...)
}

func doRequest[R any](ctx context.Context, client *Client, method string, body io.Reader, contentType string, opts ...RequestOption) (R, error) {
	var response R

	req, err := buildRequest(ctx, client, method, body, contentType, opts...)
	if err != nil {
		return response, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, err := io.ReadAll(resp.Body)
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        fmt.Errorf("failed to read HTTP response body for status %d: %w", resp.StatusCode, err),
		}
	}

	if resp.StatusCode == http.StatusNoContent {
		return response, nil
	}

	if err := client.decoder.Decode(resp.Body, &response); err != nil {
		return response, err
	}

	return response, nil
}

func buildRequest(ctx context.Context, client *Client, methodType string, body io.Reader, contentType string, opts ...RequestOption) (*http.Request, error) {
	r := &RequestOptions{
		queryParams: make(map[string]string),
		headers:     make(map[string]string),
	}
	for _, opt := range opts {
		opt(r)
	}

	rawURL, err := url.JoinPath(client.baseURL, r.path)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for key, value := range r.queryParams {
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
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

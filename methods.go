package fetcht

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
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

// There are effectively two distinct code flows for all supported HTTP requests, broken into 4 funcs for minimal
// code repetition
func doWithoutBody[R any](ctx context.Context, client *Client, method string, opts ...RequestOption) (R, error) {
	var response R

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
		return response, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return response, err
	}
	return handleResponse[R](resp, reqOptions)
}

func doWithBody[T any, R any](ctx context.Context, client *Client, method string, request T, opts ...RequestOption) (R, error) {
	var response R

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
		return response, err
	}

	req, err := buildRequest(ctx, client, method, bodyReader, contentType, reqOptions)
	if err != nil {
		return response, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return response, err
	}
	return handleResponse[R](resp, reqOptions)
}

func handleResponse[R any](resp *http.Response, reqOptions *RequestOptions) (R, error) {
	defer resp.Body.Close()
	var response R

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, err := io.ReadAll(resp.Body)
		httpErr := &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
		}

		if err != nil {
			httpErr.Err = fmt.Errorf("failed to read HTTP response body for status %d: %w", resp.StatusCode, err)
		}

		return response, httpErr
	}

	if resp.StatusCode == http.StatusNoContent {
		return response, nil
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType == "" {
		b, _ := io.ReadAll(resp.Body)
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        errors.New("server response did not include a Content-Type header"),
		}
	}
	if err != nil {
		b, _ := io.ReadAll(resp.Body)
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        fmt.Errorf("failed to parse content type from server response: %w", err),
		}
	}

	decoder, ok := reqOptions.decoders[mediaType]
	if !ok {
		b, _ := io.ReadAll(resp.Body)
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       b,
			Err:        fmt.Errorf("no registered decoders for %s, register one using WithDecoder()", mediaType),
		}
	}

	if err := decoder.Decode(resp.Body, &response); err != nil {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       nil,
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

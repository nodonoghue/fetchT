package fetcht

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Get performs an HTTP GET request and decodes response body into R
// Note: currently only supports Content-Type: application/json, which is set explicitly
func Get[R any](ctx context.Context, client *Client) (R, error) {
	return doWithoutBody[R](ctx, client, http.MethodGet)
}

// Delete performs an HTTP DELETE request, encoding any response into R
// Note: currently only supports Content-type: application/json, which is set explicitly
func Delete[R any](ctx context.Context, client *Client) (R, error) {
	return doWithoutBody[R](ctx, client, http.MethodDelete)
}

// Post performs an HTTP POST request, encoding request of T and decoding response into R
// Note: currently only supports Content-Type: application/json, which is set explicitly
func Post[T any, R any](ctx context.Context, client *Client, request T) (R, error) {
	return doWithBody[T, R](ctx, client, http.MethodPost, request)
}

// Put performs an HTTP PUT request, encoding request of T and decoding response into R
// Note: currently only supports Content-Type: application/json, which is set explicitly
func Put[T any, R any](ctx context.Context, client *Client, request T) (R, error) {
	return doWithBody[T, R](ctx, client, http.MethodPut, request)
}

// Patch performs an HTTP PATCH request, encoding request of T and decoding response into R
// Note: currently only supports Content-Type: application/json, which is set explicitly
func Patch[T any, R any](ctx context.Context, client *Client, request T) (R, error) {
	return doWithBody[T, R](ctx, client, http.MethodPatch, request)
}

// There are effectively two distinct code flows for all supported HTTP requests, broken into 3 funcs for minimal
// code repetition
func doWithoutBody[R any](ctx context.Context, client *Client, method string) (R, error) {
	return doRequest[R](ctx, client, method, nil)
}

func doWithBody[T any, R any](ctx context.Context, client *Client, method string, request T) (R, error) {
	var response R

	jsonData, err := json.Marshal(request)
	if err != nil {
		return response, err
	}

	return doRequest[R](ctx, client, method, bytes.NewBuffer(jsonData))
}

func doRequest[R any](ctx context.Context, client *Client, method string, body io.Reader) (R, error) {
	var response R

	if err := validateClient(client); err != nil {
		return response, err
	}

	req, err := buildRequest(ctx, client, method, body)
	if err != nil {
		return response, err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return response, &HTTPError{
			resp.StatusCode,
			resp.Status,
			body,
		}
	}

	if resp.StatusCode == http.StatusNoContent {
		return response, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}

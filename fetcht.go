package fetcht

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	errorMessage = "HTTP error: %d %s.  Raw body: %s"
)

func buildRequest(ctx context.Context, client *Client, methodType string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, methodType, client.requestUrl, body)
	if err != nil {
		return nil, err
	}

	//TEMP: will be removed in future update, locks requests to use Content-Type: application/json
	req.Header.Set("Content-Type", "application/json")

	for key, value := range client.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

func validateClient(client *Client) error {
	if client.httpClient == nil {
		return fmt.Errorf("httpClient is required")
	}

	if client.requestUrl == "" {
		return fmt.Errorf("requestUrl is required")
	}

	return nil
}

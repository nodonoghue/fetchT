package fetcht

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	errorMessage = "HTTP error: %d %s"
)

func Get[R any](client Client) (R, error) {
	var response R

	if err := validateClient(client); err != nil {
		return response, err
	}

	resp, err := http.Get(client.baseUrl)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return response, errors.New(fmt.Sprintf(errorMessage, resp.StatusCode, resp.Status))
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}

func Post[T any, R any](client Client, request T) (R, error) {
	var response R

	if err := validateClient(client); err != nil {
		return response, err
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return response, err
	}

	resp, err := http.Post(client.baseUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return response, errors.New(fmt.Sprintf(errorMessage, resp.StatusCode, resp.Status))
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}

func Put[T any, R any](client Client, request T) (R, error) {
	var response R

	if err := validateClient(client); err != nil {
		return response, err
	}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return response, err
	}
	req, err := http.NewRequest(http.MethodPut, client.baseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return response, err
	}

	for key, value := range client.headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return response, errors.New(fmt.Sprintf(errorMessage, resp.StatusCode, resp.Status))
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}

func buildRequest(client Client, methodType string) (*http.Request, error) {
	req, err := http.NewRequest(methodType, client.baseUrl, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range client.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

func validateClient(client Client) error {
	if client.httpClient == nil {
		return fmt.Errorf("httpClient is required")
	}

	if client.baseUrl == "" {
		return fmt.Errorf("baseUrl is required")
	}

	return nil
}

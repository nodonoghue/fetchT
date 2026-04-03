package fetcht

import (
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseUrl    string
	headers    map[string]string
	timeout    time.Duration
}

type Option func(*Client)

func WithHttpClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithBaseUrl(baseUrl string) Option {
	return func(c *Client) {
		c.baseUrl = baseUrl
	}
}

func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

func WithTimeout(seconds int) Option {
	return func(c *Client) {
		c.timeout = time.Duration(seconds) * time.Second
		c.httpClient.Timeout = c.timeout
	}
}

func NewClient(options ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		headers: make(map[string]string),
		timeout: time.Second * 30,
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

package fetcht

import (
	"crypto/tls"
	"errors"
	"net/http"
	"time"
)

// Client is an HTTP client with default configuration applied to all requests.
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
	transport  *http.Transport
	tlsConfig  *tls.Config
	timeout    time.Duration
}

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHeader adds a default header applied to all requests made by this client.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithTimeout sets the timeout duration for a client, defaults to 30 * time.Second
func WithTimeout(duration time.Duration) Option {
	return func(c *Client) {
		c.timeout = duration
	}
}

// WithTransport sets the transport options for the client.
func WithTransport(transport *http.Transport) Option {
	return func(c *Client) {
		c.transport = transport
	}
}

// WithTLSConfig sets the TLS options for the client's http.Transport
func WithTLSConfig(config *tls.Config) Option {
	return func(c *Client) {
		c.tlsConfig = config
	}
}

// NewClient builds and returns a configured Client.  Returns an error if baseURL is not set.
func NewClient(options ...Option) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{},
		timeout:    time.Second * 30,
		headers:    make(map[string]string),
	}

	for _, opt := range options {
		opt(c)
	}

	if c.baseURL == "" {
		return nil, errors.New("base url is required")
	}

	if c.transport == nil {
		c.transport = &http.Transport{}
	}
	if c.tlsConfig != nil {
		c.transport.TLSClientConfig = c.tlsConfig
	}
	c.httpClient.Transport = c.transport
	c.httpClient.Timeout = c.timeout

	return c, nil
}

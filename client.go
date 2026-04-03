package fetcht

import (
	"crypto/tls"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	requestUrl string
	headers    map[string]string
	transport  *http.Transport
	tlsConfig  *tls.Config
	timeout    time.Duration
}

type Option func(*Client)

func WithRequestUrl(baseUrl string) Option {
	return func(c *Client) {
		c.requestUrl = baseUrl
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
	}
}

func WithTransport(transport *http.Transport) Option {
	return func(c *Client) {
		c.transport = transport
	}
}

func WithTLSConfig(config *tls.Config) Option {
	return func(c *Client) {
		c.tlsConfig = config
	}
}

func NewClient(options ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{},
		timeout:    time.Second * 30,
		headers:    make(map[string]string),
	}

	for _, opt := range options {
		opt(c)
	}

	if c.transport == nil {
		c.transport = &http.Transport{}
	}
	if c.tlsConfig != nil {
		c.transport.TLSClientConfig = c.tlsConfig
	}
	c.httpClient.Transport = c.transport
	c.httpClient.Timeout = c.timeout

	return c
}

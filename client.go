// Package fetcht is a generic HTTP client library: request and response bodies
// are typed by the caller and (de)serialized through pluggable Encoder/Decoder
// implementations.
//
// The verb entry points (Get, Delete, Post, Put, Patch) are package-level
// functions that take a *Client, rather than methods on *Client. This is a
// language constraint, not a stylistic choice: Go does not permit a method to
// introduce a type parameter that is not already declared on its receiver, so
// `func (c *Client) Get[R any](...)` does not compile. Free functions are the
// only way to parameterize the response type R (and, for body verbs, the
// request type T) per call while still threading client-level configuration
// through.
package fetcht

import (
	"crypto/tls"
	"errors"
	"maps"
	"net"
	"net/http"
	"time"
)

const defaultMaxIdleConnsPerHost = 100

// defaultTransport mirrors http.DefaultTransport but raises MaxIdleConnsPerHost
// from the standard library default of 2. Under concurrency that default forces
// idle connections to be closed and re-dialed, paying a fresh TCP+TLS handshake
// per request — the dominant cost in a high-traffic client.
func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   defaultMaxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Client is an HTTP client with default configuration applied to all requests.
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
	decoders   map[string]Decoder
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

// WithDecoder registers a decoder for the given content type, overriding the default if one exists.
func WithDecoder(contentType string, decoder Decoder) Option {
	return func(c *Client) {
		c.decoders[contentType] = decoder
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
		decoders:   make(map[string]Decoder, len(defaultDecoders)),
	}
	maps.Copy(c.decoders, defaultDecoders)

	for _, opt := range options {
		opt(c)
	}

	if c.baseURL == "" {
		return nil, errors.New("base url is required")
	}

	if c.transport == nil {
		c.transport = defaultTransport()
	}
	if c.tlsConfig != nil {
		c.transport.TLSClientConfig = c.tlsConfig
	}
	c.httpClient.Transport = c.transport
	c.httpClient.Timeout = c.timeout

	return c, nil
}

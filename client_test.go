package fetcht

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestWithoutBaseURL(t *testing.T) {
	_, err := NewClient()
	if err == nil {
		t.Error("NewClient should fail without baseURL")
	}
}

func TestWithBaseURL(t *testing.T) {
	client, err := NewClient(WithBaseURL("https://example.com"))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}
}

func TestWithNewDecoder(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithDecoder("testing/json", JSONDecoder))

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.decoders["testing/json"] != JSONDecoder {
		t.Errorf("NewClient got decoder %v, want JSONDecoder", client.decoders["testing/json"])
	}
}

func TestWithOverrideDecoder(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithDecoder("application/json", XMLDecoder))

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.decoders["application/json"] != XMLDecoder {
		t.Errorf("NewClient got decoder %v, want XMLDecoder", client.decoders["testing/json"])
	}
}

func TestWithMultipleDecoders(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithDecoder("decoder1", XMLDecoder),
		WithDecoder("decoder2", JSONDecoder))

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.decoders["decoder1"] != XMLDecoder {
		t.Errorf("NewClient got decoder %v, want XMLDecoder", client.decoders["decoder1"])
	}

	if client.decoders["decoder2"] != JSONDecoder {
		t.Errorf("NewClient got decoder %v, want JSONDecoder", client.decoders["decoder2"])
	}
}

func TestWithNewHeader(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithHeader("a", "a"))

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.headers["a"] != "a" {
		t.Errorf("NewClient got header %q, want a", client.headers["a"])
	}
}

func TestWithOverrideHeader(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithHeader("a", "a"),
		WithHeader("a", "b"))

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.headers["a"] != "b" {
		t.Errorf("NewClient got header %q, want b", client.headers["a"])
	}
}

func TestWithMultipleHeader(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithHeader("a", "a"),
		WithHeader("b", "b"))

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.headers["a"] != "a" {
		t.Errorf("NewClient got header %q, want a", client.headers["a"])
	}

	if client.headers["b"] != "b" {
		t.Errorf("NewClient got header %q, want b", client.headers["b"])
	}
}

func TestWithDefaultTimeout(t *testing.T) {
	client, err := NewClient(WithBaseURL("https://example.com"))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.timeout != 30*time.Second {
		t.Errorf("NewClient got timeout %v, want 30 seconds", client.timeout)
	}
}

func TestWithCustomTimeout(t *testing.T) {
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.timeout != 10*time.Second {
		t.Errorf("NewClient got timeout %v, want 10 seconds", client.timeout)
	}
}

func TestWithTransport(t *testing.T) {
	transport := &http.Transport{
		MaxIdleConns:    13,
		IdleConnTimeout: 23 * time.Second,
	}

	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithTransport(transport))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.transport.MaxIdleConns != 13 {
		t.Errorf("NewClient got http.Transport with MaxIdleConns %v, want 13", client.transport.MaxIdleConns)
	}

	if client.transport.IdleConnTimeout != 23*time.Second {
		t.Errorf("NewClient got http.Transport with IdleConnTimeout %v, want 23 seconds", client.transport.IdleConnTimeout)
	}

	if client.httpClient.Transport != transport {
		t.Error("Transport not propagated to internal client")
	}
}

func TestWithTLSConfig(t *testing.T) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithTLSConfig(tlsConfig))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.tlsConfig.InsecureSkipVerify != true {
		t.Errorf("NewClient got tls.Config with InsecureSkipVerify %v, want true", client.tlsConfig.InsecureSkipVerify)
	}
	if client.tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("NewClient got tls.Config with MinVersion %v, want 1.2", client.tlsConfig.MinVersion)
	}

	if client.transport.TLSClientConfig != tlsConfig {
		t.Error("TLS config not propagated to internal transport")
	}
}

func TestWithTransportAndTLSConfig(t *testing.T) {
	transport := &http.Transport{
		MaxIdleConns: 13,
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	client, err := NewClient(
		WithBaseURL("https://example.com"),
		WithTransport(transport),
		WithTLSConfig(tlsConfig))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("NewClient got baseURL %q, want https://example.com", client.baseURL)
	}

	if client.transport != transport {
		t.Error("Transport not correctly assigned or overridden")
	}

	if client.transport.TLSClientConfig != tlsConfig {
		t.Error("TLS config not propagated to internal transport")
	}
}

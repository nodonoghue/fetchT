# fetchT

A generic HTTP client library for Go. Uses generics to encode request bodies and decode response bodies into typed structs, with pluggable encoders and decoders for different content types.

Requires Go 1.21+.

## Installation

```bash
go get github.com/nodonoghue/fetchT
```

## Quick Start

```go
client, err := fetcht.NewClient(
    fetcht.WithBaseURL("https://api.example.com"),
    fetcht.WithHeader("Authorization", "Bearer "+token),
)
if err != nil {
    log.Fatal(err)
}

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

resp, err := fetcht.Get[User](ctx, client, fetcht.WithPath("/users/1"))
if err == nil {
    fmt.Printf("User: %s\n", resp.Data.Name)
}
```

## Client Configuration

All configuration is passed to `NewClient` as functional options. `NewClient` returns an error if `WithBaseURL` is not provided.

| Option | Description |
|---|---|
| `WithBaseURL(url string)` | **Required.** Sets the base URL for all requests. |
| `WithHeader(key, value string)` | Adds a default header applied to every request. |
| `WithDecoder(contentType string, d Decoder)` | Registers or overrides a decoder in the client's decoder registry. |
| `WithTimeout(d time.Duration)` | Sets the request timeout. Default: 30s. |
| `WithTransport(t *http.Transport)` | Replaces the default HTTP transport. |
| `WithTLSConfig(c *tls.Config)` | Configures TLS on the client's transport. |

```go
client, err := fetcht.NewClient(
    fetcht.WithBaseURL("https://api.example.com"),
    fetcht.WithHeader("X-App-Version", "1.0"),
    fetcht.WithTimeout(10 * time.Second),
)
```

## Making Requests

All request functions are free functions parameterised on the response type `R`. Functions that send a body are also parameterised on the request type `T`. They return a `*Response[R]` containing the decoded data and response metadata.

```go
// GET — no request body
Get[R any](ctx context.Context, client *Client, ...RequestOption) (*Response[R], error)

// DELETE — no request body
Delete[R any](ctx context.Context, client *Client, ...RequestOption) (*Response[R], error)

// POST / PUT / PATCH — encodes request body
Post[T any, R any](ctx context.Context, client *Client, request T, ...RequestOption) (*Response[R], error)
Put[T any, R any](ctx context.Context, client *Client, request T, ...RequestOption) (*Response[R], error)
Patch[T any, R any](ctx context.Context, client *Client, request T, ...RequestOption) (*Response[R], error)
```

### The Response Struct

The `Response[R]` struct provides access to the decoded data and HTTP metadata:

```go
type Response[R any] struct {
    Data          R              // Decoded response body
    StatusCode    int            // HTTP status code (e.g. 200)
    Status        string         // HTTP status text (e.g. "200 OK")
    Proto         string         // Protocol version (e.g. "HTTP/1.1")
    Header        http.Header    // Response headers
    Cookies       []*http.Cookie // Response cookies
    Request       *http.Request  // The final request object
    ContentLength int64          // Value of Content-Length header
    RawBody       []byte         // Raw un-decoded response body
    HasBody       bool           // True if the response body was non-empty
}
```

> **Note:** Go does not allow generic methods with new type parameters on a receiver, so these are package-level functions rather than methods on `Client`. This is a known language constraint, not a design choice.

#### Empty-body responses

When a 2xx response has `Content-Length: 0` (e.g. `204 No Content`), the body read and decoder dispatch are both skipped. `Data` is the zero value of `R`, `RawBody` is `nil`, and `HasBody` is `false`. Use `HasBody` (or `StatusCode`) to distinguish "the server sent no body" from "the server sent a body that decoded to a zero-value struct":

```go
resp, err := fetcht.Delete[Order](ctx, client, fetcht.WithPath("/orders/99"))
if err != nil {
    return err
}
if !resp.HasBody {
    // 204 or otherwise empty — resp.Data is the zero Order
}
```

### Examples

```go
// GET
type Product struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
}

resp, err := fetcht.Get[Product](ctx, client, fetcht.WithPath("/products/42"))
if err == nil {
    fmt.Printf("Product: %s (Status: %d)\n", resp.Data.Title, resp.StatusCode)
}

// POST
type CreateOrder struct {
    ProductID int `json:"product_id"`
    Quantity  int `json:"quantity"`
}
type OrderResponse struct {
    OrderID int    `json:"order_id"`
    Status  string `json:"status"`
}

resp, err = fetcht.Post[CreateOrder, OrderResponse](ctx, client,
    CreateOrder{ProductID: 42, Quantity: 2},
    fetcht.WithPath("/orders"),
)

// DELETE with no meaningful response body
resp, err = fetcht.Delete[struct{}](ctx, client, fetcht.WithPath("/orders/99"))
```

## Per-Request Options

Each request function accepts `...RequestOption` to configure that individual call without affecting the client.

| Option | Description |
|---|---|
| `WithPath(path string)` | Appends a path to the client's base URL. |
| `WithQueryParams(key, value string)` | Adds a query string parameter. Repeatable. |
| `WithHeaders(key, value string)` | Adds a per-request header, overriding any client-level default with the same key. |
| `WithEncoder(e Encoder)` | Sets the request body encoder for this request. Default: JSON. |
| `WithDecoders(contentType string, d Decoder)` | Adds or overrides a decoder for this request only, taking priority over the client registry. |

```go
users, err := fetcht.Get[[]User](ctx, client,
    fetcht.WithPath("/users"),
    fetcht.WithQueryParams("page", "2"),
    fetcht.WithQueryParams("limit", "25"),
    fetcht.WithHeaders("X-Correlation-ID", requestID),
)
```

## Encoding Requests

The encoder is set per-request via `WithEncoder`. If not specified, JSON is used. The encoder also sets the `Content-Type` header automatically — any manually set `Content-Type` header will be overridden.

Built-in encoders:

| Variable | Content-Type |
|---|---|
| `fetcht.JSONEncoder` | `application/json` (default) |
| `fetcht.XMLEncoder` | `application/xml; charset=utf-8` |
| `fetcht.FormEncoder` | `application/x-www-form-urlencoded` |
| `fetcht.MultipartEncoder` | `multipart/form-data` |

`FormEncoder` requires request structs to be annotated with `url` struct tags (via `github.com/google/go-querystring`).

`MultipartEncoder` requires the request value to be a `fetcht.MultipartForm`:

```go
type FilePart struct {
    Reader io.Reader
    Name   string
}

type MultipartForm struct {
    Fields map[string]string
    Files  map[string]FilePart
}
```

Custom encoders can be implemented by satisfying the `Encoder` interface:

```go
type Encoder interface {
    Encode(v any) (io.Reader, string, error)
}
```

## Decoding Responses

Response decoding is driven by the `Content-Type` header of the server response. Each client holds a decoder registry (`map[string]Decoder`) keyed by MIME type. The correct decoder is selected automatically at runtime using `mime.ParseMediaType`.

Built-in decoders registered by default:

| MIME type | Decoder |
|---|---|
| `application/json` | `fetcht.JSONDecoder` |
| `application/xml` | `fetcht.XMLDecoder` |
| `text/xml` | `fetcht.XMLDecoder` |

**Client-level registration** via `WithDecoder` applies to every request made by that client. Use this for vendor media types or to override a built-in:

```go
// Register once — applies to all requests on this client
client, err := fetcht.NewClient(
    fetcht.WithBaseURL("https://api.example.com"),
    fetcht.WithDecoder("application/vnd.api+json", fetcht.JSONDecoder),
)
```

**Per-request override** via `WithDecoders` applies only to that call and takes priority over the client registry:

```go
resp, err := fetcht.Get[MyType](ctx, client,
    fetcht.WithPath("/resource"),
    fetcht.WithDecoders("application/vnd.api+json", fetcht.JSONDecoder),
)
```

Custom decoders can be implemented by satisfying the `Decoder` interface:

```go
type Decoder interface {
    Decode(r io.Reader, v any) error
}
```

If the server response is missing a `Content-Type` header, or sends a MIME type with no registered decoder, an `HTTPError` is returned containing the raw response body.

## Error Handling

All non-2xx responses and decode-path failures return `*HTTPError`. Note that even on error, a `*Response[R]` may be returned containing whatever metadata was available before the failure.

```go
type HTTPError struct {
    StatusCode int
    Status     string
    Body       []byte  // raw response body
    Err        error   // set for decode/read failures; nil for plain non-2xx responses
}
```

Use `errors.As` to inspect HTTP errors:

```go
resp, err := fetcht.Get[User](ctx, client, fetcht.WithPath("/users/1"))
if err != nil {
    var httpErr *fetcht.HTTPError
    if errors.As(err, &httpErr) {
        fmt.Printf("status %d: %s\n", httpErr.StatusCode, httpErr.Body)
    }
    return err
}
```

## Known Limitations

- **No response streaming.** The library always reads the full response body into memory before decoding. This is a fundamental constraint of decoding into a typed `R` — the decoder needs the complete document. For use cases that require a live reader (large file downloads, server-sent events, NDJSON streams), use `net/http` directly.
- **`go-querystring` dependency.** `FormEncoder` depends on `github.com/google/go-querystring`. The library is not pure stdlib.

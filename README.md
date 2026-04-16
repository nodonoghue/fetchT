# fetchT

A generic HTTP client library for Go. Uses generics to encode request bodies and decode response bodies into typed structs, with pluggable encoders and decoders for different content types.

Requires Go 1.18+.

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

user, err := fetcht.Get[User](ctx, client, fetcht.WithPath("/users/1"))
```

## Client Configuration

All configuration is passed to `NewClient` as functional options. `NewClient` returns an error if `WithBaseURL` is not provided.

| Option | Description |
|---|---|
| `WithBaseURL(url string)` | **Required.** Sets the base URL for all requests. |
| `WithHeader(key, value string)` | Adds a default header applied to every request. |
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

All request functions are free functions parameterised on the response type `R`. Functions that send a body are also parameterised on the request type `T`.

```go
// GET — no request body
Get[R any](ctx, client, ...RequestOption) (R, error)

// DELETE — no request body
Delete[R any](ctx, client, ...RequestOption) (R, error)

// POST / PUT / PATCH — encodes request body
Post[T, R any](ctx, client, request T, ...RequestOption) (R, error)
Put[T, R any](ctx, client, request T, ...RequestOption) (R, error)
Patch[T, R any](ctx, client, request T, ...RequestOption) (R, error)
```

> **Note:** Go does not allow generic methods with new type parameters on a receiver, so these are package-level functions rather than methods on `Client`. This is a known language constraint, not a design choice.

### Examples

```go
// GET
type Product struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
}

product, err := fetcht.Get[Product](ctx, client, fetcht.WithPath("/products/42"))

// POST
type CreateOrder struct {
    ProductID int `json:"product_id"`
    Quantity  int `json:"quantity"`
}
type OrderResponse struct {
    OrderID int    `json:"order_id"`
    Status  string `json:"status"`
}

resp, err := fetcht.Post[CreateOrder, OrderResponse](ctx, client,
    CreateOrder{ProductID: 42, Quantity: 2},
    fetcht.WithPath("/orders"),
)

// DELETE with no meaningful response body
_, err = fetcht.Delete[struct{}](ctx, client, fetcht.WithPath("/orders/99"))
```

## Per-Request Options

Each request function accepts `...RequestOption` to configure that individual call without affecting the client.

| Option | Description |
|---|---|
| `WithPath(path string)` | Appends a path to the client's base URL. |
| `WithQueryParams(key, value string)` | Adds a query string parameter. Repeatable. |
| `WithHeaders(key, value string)` | Adds a per-request header, overriding any client-level default with the same key. |
| `WithEncoder(e Encoder)` | Sets the request body encoder for this request. Default: JSON. |
| `WithDecoder(contentType string, d Decoder)` | Adds or overrides a decoder in the registry for this request. |

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
type MultipartForm struct {
    Fields map[string]string
    Files  map[string]FilePart  // FilePart{Reader io.Reader, Name string}
}
```

Custom encoders can be implemented by satisfying the `Encoder` interface:

```go
type Encoder interface {
    Encode(v any) (io.Reader, string, error)
}
```

## Decoding Responses

Response decoding is driven by the `Content-Type` header of the server response. Each client holds an internal decoder registry (`map[string]Decoder`) keyed by MIME type. The correct decoder is selected automatically at runtime using `mime.ParseMediaType`.

Built-in decoders registered by default:

| MIME type | Decoder |
|---|---|
| `application/json` | `fetcht.JSONDecoder` |
| `application/xml` | `fetcht.XMLDecoder` |
| `text/xml` | `fetcht.XMLDecoder` |

Additional decoders can be registered per-request with `WithDecoder`. This is also how you override a built-in, or alias one MIME type to an existing decoder:

```go
// Register a custom decoder for a vendor media type
resp, err := fetcht.Get[MyType](ctx, client,
    fetcht.WithPath("/resource"),
    fetcht.WithDecoder("application/vnd.api+json", fetcht.JSONDecoder),
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

All non-2xx responses and decode-path failures return `*HTTPError`:

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
user, err := fetcht.Get[User](ctx, client, fetcht.WithPath("/users/1"))
if err != nil {
    var httpErr *fetcht.HTTPError
    if errors.As(err, &httpErr) {
        fmt.Printf("status %d: %s\n", httpErr.StatusCode, httpErr.Body)
    }
    return err
}
```

## Known Limitations

- **204 No Content returns the zero value of `R`.** When `R` is a struct, the returned value is indistinguishable from a successful response with an empty body. Use a pointer type (`*MyStruct`) if you need to tell the two apart — a nil pointer unambiguously means no content.
- **`go-querystring` dependency.** `FormEncoder` depends on `github.com/google/go-querystring`. The library is not pure stdlib.

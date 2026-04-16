package fetcht

type RequestOptions struct {
	path        string
	queryParams map[string]string
	headers     map[string]string
	encoder     Encoder
	decoders    map[string]Decoder
}

// RequestOption is a functional option for configuring an individual request.
type RequestOption func(options *RequestOptions)

// WithPath appends an endpoint path to the client's baseURL for this request.
func WithPath(path string) RequestOption {
	return func(r *RequestOptions) {
		r.path = path
	}
}

// WithQueryParams adds a query string key-value pair to the request URL.
func WithQueryParams(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.queryParams[key] = value
	}
}

// WithHeaders adds a per-request header, overriding any client-level header with the same key.
func WithHeaders(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.headers[key] = value
	}
}

// WithEncoder sets the content type encoder for this request, this will also
// set the content-type header value, any user added content-type headers
// will be overridden by the encoder.
func WithEncoder(encoder Encoder) RequestOption {
	return func(r *RequestOptions) {
		r.encoder = encoder
	}
}

// WithDecoder adds a decoder, or overrides a registered decoder in the internal decoder
// registry.
func WithDecoder(contentType string, decoder Decoder) RequestOption {
	return func(c *RequestOptions) {
		c.decoders[contentType] = decoder
	}
}

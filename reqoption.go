package fetcht

type RequestOptions struct {
	path        string
	queryParams map[string]string
	headers     map[string]string
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

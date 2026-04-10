package fetcht

type RequestOptions struct {
	path        string
	queryParams map[string]string
	headers     map[string]string
}

type RequestOption func(options *RequestOptions)

// WithPath added an enpoint path to the baseURL in an already configured client
func WithPath(path string) RequestOption {
	return func(r *RequestOptions) {
		r.path = path
	}
}

// WithQueryParams adds querystring parameters for requests that support
func WithQueryParams(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.queryParams[key] = value
	}
}

// WithHeaders Added per request headers
func WithHeaders(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.headers[key] = value
	}
}

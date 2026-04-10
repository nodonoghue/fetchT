package fetcht

type RequestOptions struct {
	path        string
	queryParams map[string]string
	headers     map[string]string
}

type RequestOption func(options *RequestOptions)

func WithPath(path string) RequestOption {
	return func(r *RequestOptions) {
		r.path = path
	}
}

func WithQueryParams(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.queryParams[key] = value
	}
}

func WithHeaders(key, value string) RequestOption {
	return func(r *RequestOptions) {
		r.headers[key] = value
	}
}

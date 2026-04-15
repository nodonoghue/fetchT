package fetcht

import "fmt"

// HTTPError encapsulates any errors from the fetchT library
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
	Err        error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("HTTP error: %d %s: %s (body read error: %v)", e.StatusCode, e.Status, string(e.Body), e.Err)
	}
	return fmt.Sprintf("HTTP error: %d %s: %s", e.StatusCode, e.Status, e.Body)
}

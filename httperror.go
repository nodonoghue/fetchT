package fetcht

import "fmt"

type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: %d %s: %s", e.StatusCode, e.Status, e.Body)
}

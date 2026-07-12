package api

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewHTTPError(status int, message string) error {
	return &HTTPError{StatusCode: status, Message: message}
}

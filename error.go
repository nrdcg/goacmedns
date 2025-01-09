package goacmedns

import "fmt"

// ClientError represents an error from the ACME-DNS server.
// It holds a [ClientError.Message] describing the operation the client was doing,
// a [ClientError.HTTPStatus] code returned by the server, and the [ClientError.Body] of the HTTP Response from the server.
type ClientError struct {
	// Message is a string describing the client operation that failed.
	Message string
	// HTTPStatus is the HTTP status code the ACME DNS server returned.
	HTTPStatus int
	// Body is the response body the ACME DNS server returned.
	Body []byte
}

// newClientError creates a ClientError instance populated with the given arguments.
func newClientError(msg string, respCode int, respBody []byte) *ClientError {
	return &ClientError{
		Message:    msg,
		HTTPStatus: respCode,
		Body:       respBody,
	}
}

// Error collects all the ClientError fields into a single string.
func (e ClientError) Error() string {
	return fmt.Sprintf("%d: %s, response: %s",
		e.HTTPStatus, e.Message, string(e.Body))
}

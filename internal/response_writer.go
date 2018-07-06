package internal

import "net/http"

// ResponseWriter is used to collect some metrics for a http response
type ResponseWriter interface {
	StatusCode() int
	Body() []byte
	WrittenBodyBytes() int

	Header() http.Header
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

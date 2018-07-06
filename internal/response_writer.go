package internal

import "net/http"

type ResponseWriter interface {
	StatusCode() int
	Body() []byte
	WrittenBodyBytes() int

	Header() http.Header
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

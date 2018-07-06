package internal

import (
	"net/http"
	"sync"
)

type responseWriterWithoutBody struct {
	statusCode int
	written    int
	http.ResponseWriter
	customMetrics sync.Map
}

func (rw *responseWriterWithoutBody) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	if n > 0 {
		rw.written += n
	}
	return n, err
}

func (rw *responseWriterWithoutBody) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriterWithoutBody) StatusCode() int {
	return rw.statusCode
}

func (rw *responseWriterWithoutBody) Body() []byte {
	return nil
}

func (rw *responseWriterWithoutBody) WrittenBodyBytes() int {
	return rw.written
}

func (rw *responseWriterWithoutBody) SetCustomMetric(key, value interface{}) {
	rw.customMetrics.Store(key, value)
}

func (rw *responseWriterWithoutBody) GetCustomMetric(key interface{}) (interface{}, bool) {
	return rw.customMetrics.Load(key)
}

// NewResponseWriterWithoutBody creates a new ResponseWriter that skipts the body
func NewResponseWriterWithoutBody(w http.ResponseWriter) ResponseWriter {
	return &responseWriterWithoutBody{
		ResponseWriter: w,
	}
}

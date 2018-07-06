package httpmetrics

import (
	"net/http"
	"time"

	"github.com/talon-one/go-httpmetrics/internal"
)

// Metrics holds the collected metrics
type Metrics struct {
	// Duration is the time it took to execute the handler.
	Duration       time.Duration
	Request        Request
	Response       Response
	responseWriter internal.ResponseWriter
}

// Header is a dummy function for fulfilling the http.Handler interface
func (Metrics) Header() http.Header {
	return http.Header(make(map[string][]string))
}

// Write is a dummy function for fulfilling the http.Handler interface
func (Metrics) Write(b []byte) (int, error) {
	return len(b), nil
}

// WriteHeader is a dummy function for fulfilling the http.Handler interface
func (Metrics) WriteHeader(int) {}

// GetCustomMetric can be used to get a custom metric value
func (m Metrics) GetCustomMetric(key interface{}) (interface{}, bool) {
	return m.responseWriter.GetCustomMetric(key)
}

// Request extends the http.Request that was sent with Body and BodySize
type Request struct {
	*http.Request
	Body              []byte
	ConsumedBodyBytes int
}

// Response contains the http response that has been sent to the client
type Response struct {
	Code             int
	Body             []byte
	WrittenBodyBytes int
	Header           http.Header
}

// MetricsFunc is used for the callback registered by Collect
type MetricsFunc func(Metrics)

// MetricsRequest will be passed to the CustomRouter, set the Collect fields to enable collection of this Request
type MetricsRequest struct {
	*CollectOptions
	Collect bool
}

// Header is a dummy function for fulfilling the http.Handler interface
func (*MetricsRequest) Header() http.Header {
	return http.Header(make(map[string][]string))
}

// Write is a dummy function for fulfilling the http.Handler interface
func (*MetricsRequest) Write(b []byte) (int, error) {
	return len(b), nil
}

// WriteHeader is a dummy function for fulfilling the http.Handler interface
func (*MetricsRequest) WriteHeader(int) {}

// SetCustomMetric can be used to set custom fields inside the Handler
func SetCustomMetric(w http.ResponseWriter, key, value interface{}) {
	if rw, ok := w.(internal.ResponseWriter); ok {
		rw.SetCustomMetric(key, value)
	}
}

// GetCustomMetric can be used to get a custom field value out of a http.ResponseWriter
func GetCustomMetric(w http.ResponseWriter, key interface{}) (interface{}, bool) {
	if rw, ok := w.(internal.ResponseWriter); ok {
		return rw.GetCustomMetric(key)
	}
	return nil, false
}

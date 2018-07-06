package httpmetrics

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/talon-one/go-httpmetrics/internal"
)

// Collector is a http middleware that can be used to collect metrics for an http request and the coresponding response
type Collector struct {
	Options *CollectOptions

	mu             sync.Mutex
	routes         map[string]http.HandlerFunc
	defaultHandler http.HandlerFunc
}

// CollectOptions controls the behavior of Collect
type CollectOptions struct {
	// Handler that should be used to pass requests to
	Handler http.Handler
	// CollectResponseBody sets the MaxBufferSize of the Body that should be collected
	CollectResponseBody int
	// CollectRequestBody sets the MaxBufferSize of the Body that should be collected
	CollectRequestBody int
	// CustomRouter can be used to define a custom router that should be used in addition to the Collect function
	CustomRouter http.Handler
}

// New create a new Collector
func New(options CollectOptions) *Collector {
	if options.Handler == nil {
		options.Handler = http.DefaultServeMux
	}
	opts := &options
	return &Collector{
		routes:  make(map[string]http.HandlerFunc),
		Options: opts,
	}
}

func (collector *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if router, options := collector.shouldCollect(r); router != nil && options != nil {
		var metrics Metrics
		metrics.Request.Request = r

		if options.CollectResponseBody > 0 {
			metrics.responseWriter = internal.NewResponseWriterWithBody(w, options.CollectResponseBody)
		} else {
			metrics.responseWriter = internal.NewResponseWriterWithoutBody(w)
		}

		reqBodyReader := internal.NewRequestBodyReader(r.Body, options.CollectRequestBody)
		r.Body = reqBodyReader

		start := time.Now()
		options.Handler.ServeHTTP(metrics.responseWriter, r)
		metrics.Duration = time.Since(start)

		metrics.Response.Header = metrics.responseWriter.Header()
		metrics.Response.Body = metrics.responseWriter.Body()
		metrics.Response.Code = metrics.responseWriter.StatusCode()
		metrics.Response.WrittenBodyBytes = metrics.responseWriter.WrittenBodyBytes()
		metrics.Request.Body, _ = reqBodyReader.Body()
		metrics.Request.ConsumedBodyBytes = reqBodyReader.ConsumedBodyBytes()

		router.ServeHTTP(metrics, fakeRequest(r))

		return
	}
	collector.Options.Handler.ServeHTTP(w, r)
}

func (collector *Collector) shouldCollect(r *http.Request) (http.Handler, *CollectOptions) {
	if r == nil || r.URL == nil {
		return nil, nil
	}
	options := *collector.Options
	req := MetricsRequest{
		CollectOptions: &options,
	}

	// check if handled by our "internal" router
	collector.mu.Lock()
	handler, ok := collector.routes[strings.ToLower(r.URL.Path)]
	if ok {
		collector.mu.Unlock()
		return handler, &options
	}

	// we have no route in our router
	// maybe the custom router has something?
	if collector.Options.CustomRouter != nil {
		collector.Options.CustomRouter.ServeHTTP(&req, fakeRequest(r))
		if req.Collect {
			collector.mu.Unlock()
			return collector.Options.CustomRouter, &options
		}
	}
	// if we have a defaultHandler set
	if collector.defaultHandler != nil {
		collector.mu.Unlock()
		return collector.defaultHandler, collector.Options
	}
	collector.mu.Unlock()
	return nil, nil
}

// Collect adds the specified paths to the desired metrics function
// if no path (or *) is specified the function will be used for all unmatched requests
func (collector *Collector) Collect(fn MetricsFunc, paths ...string) {
	handler := collector.routerHandler(fn)

	collector.mu.Lock()
	if len(paths) == 0 {
		collector.defaultHandler = handler
		collector.mu.Unlock()
		return
	}
	for _, p := range paths {
		p = strings.ToLower(path.Clean(filepath.ToSlash(p)))
		if p == "*" {
			collector.defaultHandler = handler
		} else {
			// prepend slash
			p = "/" + strings.Trim(p, "/")
			collector.routes[p] = handler
		}
	}
	collector.mu.Unlock()
}

func (collector *Collector) routerHandler(fn MetricsFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		if m, ok := w.(Metrics); ok {
			fn(m)
		}
	}
}

func fakeRequest(r *http.Request) *http.Request {
	// req := *r
	// req.Body = nil
	// req.GetBody = nil
	// return req.WithContext(r.Context())
	return r
}

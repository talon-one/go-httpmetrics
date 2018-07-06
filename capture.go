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

// Collect is a http middleware that can be used to collect metrics for an http request and the coresponding response
type Collect struct {
	Options *CollectOptions

	mu             sync.Mutex
	routes         map[string]http.HandlerFunc
	defaultHandler http.HandlerFunc
}

// CollectOptions controls the behavior of Collect
type CollectOptions struct {
	Handler             http.Handler
	CollectResponseBody int
	CollectRequestBody  int
	CustomRouter        http.Handler
}

func New(options CollectOptions) *Collect {
	if options.Handler == nil {
		options.Handler = http.DefaultServeMux
	}
	opts := &options
	return &Collect{
		routes:  make(map[string]http.HandlerFunc),
		Options: opts,
	}
}

func (collect *Collect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if router, options := collect.shouldCollect(r); router != nil && options != nil {
		var metrics Metrics
		metrics.Request.Request = r
		var rw internal.ResponseWriter
		if options.CollectResponseBody > 0 {
			rw = internal.NewResponseWriterWithBody(w, options.CollectResponseBody)
		} else {
			rw = internal.NewResponseWriterWithoutBody(w)
		}

		reqBodyReader := internal.NewRequestBodyReader(r.Body, options.CollectRequestBody)
		r.Body = reqBodyReader
		start := time.Now()
		options.Handler.ServeHTTP(rw, r)
		metrics.Duration = time.Since(start)

		metrics.Response.Header = rw.Header()
		metrics.Response.Body = rw.Body()
		metrics.Response.Code = rw.StatusCode()
		metrics.Response.WrittenBodyBytes = rw.WrittenBodyBytes()
		metrics.Request.Body, _ = reqBodyReader.Body()
		metrics.Request.ConsumedBodyBytes = reqBodyReader.ConsumedBodyBytes()

		router.ServeHTTP(metrics, fakeRequest(r))

		return
	}
	collect.Options.Handler.ServeHTTP(w, r)
}

func (collect *Collect) shouldCollect(r *http.Request) (http.Handler, *CollectOptions) {
	if r == nil || r.URL == nil {
		return nil, nil
	}
	options := *collect.Options
	req := MetricsRequest{
		Options: &options,
	}

	// check if handled by our "internal" router
	collect.mu.Lock()
	handler, ok := collect.routes[strings.ToLower(r.URL.Path)]
	if ok {
		collect.mu.Unlock()
		return handler, &options
	}

	// we have no route in our router
	// maybe the custom router has something?
	if collect.Options.CustomRouter != nil {
		collect.Options.CustomRouter.ServeHTTP(&req, fakeRequest(r))
		if req.Collect {
			collect.mu.Unlock()
			return collect.Options.CustomRouter, &options
		}
	}
	// if we have a defaultHandler set
	if collect.defaultHandler != nil {
		collect.mu.Unlock()
		return collect.defaultHandler, collect.Options
	}
	collect.mu.Unlock()
	return nil, nil
}

// Collect adds the specified paths to the desired metrics function
// if no path (or *) is specified the function will be used for all unmatched requests
func (collect *Collect) Collect(fn MetricsFunc, paths ...string) {
	handler := collect.routerHandler(fn)

	collect.mu.Lock()
	if len(paths) == 0 {
		collect.defaultHandler = handler
		collect.mu.Unlock()
		return
	}
	for _, p := range paths {
		p = strings.ToLower(path.Clean(filepath.ToSlash(p)))
		if p == "*" {
			collect.defaultHandler = handler
		} else {
			// prepend slash
			p = "/" + strings.Trim(p, "/")
			collect.routes[p] = handler
		}
	}
	collect.mu.Unlock()
}

func (collect *Collect) routerHandler(fn MetricsFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		if m, ok := w.(Metrics); ok {
			fn(m)
		}
	}
}

func fakeRequest(r *http.Request) *http.Request {
	req := *r
	req.Body = nil
	req.GetBody = nil
	return &req
}

# httpmetrics [![GoDoc](https://godoc.org/github.com/talon-one/go-httpmetrics?status.svg)](https://godoc.org/github.com/talon-one/go-httpmetrics) [![go-report](https://goreportcard.com/badge/github.com/talon-one/go-httpmetrics)](https://goreportcard.com/report/github.com/talon-one/go-httpmetrics)

Capture metrics for http.Requests and http.Responses

```bash
go get github.com/talon-one/go-httpmetrics
```

# Usage
```go
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Hello World")
	})

	collectMetrics := httpmetrics.New(httpmetrics.CollectOptions{
		CollectRequestBody:  128,
		CollectResponseBody: 128,
	})

	collectMetrics.Collect(func(m httpmetrics.Metrics) {
		fmt.Printf(`Duration: %s
Request.URL: %s
Request.Header: %v
Request.Body: %s
Response.Code: %d
Response.Header: %v
Response.Body: %s
============================
`, m.Duration.String(),
			m.Request.URL.String(),
			m.Request.Header,
			string(m.Request.Body),
			m.Response.Code,
			m.Response.Header,
			string(m.Response.Body))
	})

	http.ListenAndServe(":8000", collectMetrics)
}

```

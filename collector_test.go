package httpmetrics_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/talon-one/go-httpmetrics"
)

type RequestPayload struct {
	Code    int
	Payload string
	Method  string
	Header  map[string]string
}

func EchoHandler(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	for k, v := range payload.Header {
		w.Header().Set(k, v)
	}
	w.WriteHeader(payload.Code)
	io.WriteString(w, payload.Payload)
}

func EchoMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", EchoHandler)
	return mux
}

func DoRequest(t *testing.T, server *httptest.Server, payload RequestPayload) {
	var body bytes.Buffer
	require.NoError(t, json.NewEncoder(&body).Encode(payload))

	req, err := http.NewRequest(payload.Method, server.URL, &body)
	require.NoError(t, err)

	res, err := server.Client().Do(req)
	require.NoError(t, err)

	// compare the statuscode
	require.Equal(t, payload.Code, res.StatusCode)

	// compare the headers
	for k, v := range payload.Header {
		require.Equal(t, v, res.Header.Get(k))
	}

	// compare the payload
	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, payload.Payload, string(b))
}

func CompareMetricsWithPayload(t *testing.T, payload RequestPayload, m httpmetrics.Metrics) {
	require.Equal(t, payload.Code, m.Response.Code)
	require.Equal(t, payload.Method, m.Request.Method)
	require.Equal(t, payload.Payload, string(m.Response.Body))

	for k, v := range payload.Header {
		require.Equal(t, v, m.Response.Header.Get(k))
	}
}

type AllHandler struct {
	Handler http.HandlerFunc
}

func (h AllHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Handler(w, r)
}

func HandleAllRequests(handler http.HandlerFunc) http.Handler {
	return AllHandler{
		Handler: handler,
	}
}

func TestCollect(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(1)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler:             EchoMux(),
		CollectResponseBody: 1024,
	})

	collector.Collect(func(m httpmetrics.Metrics) {
		CompareMetricsWithPayload(t, request, m)
		wg.Done()
	}, "/test")
	s := httptest.NewServer(collector)
	s.URL += "/test"
	DoRequest(t, s, request)
	wg.Wait()
}

func TestCollectAll(t *testing.T) {
	t.Run("Explicit (with Star)", func(t *testing.T) {
		request := RequestPayload{
			Method: http.MethodPost,
			Code:   http.StatusOK,
			Header: map[string]string{
				"X-CUSTOM-HEADER": "VALUE",
			},
			Payload: "Hello World",
		}
		var wg sync.WaitGroup
		wg.Add(1)
		collector := httpmetrics.New(httpmetrics.CollectOptions{
			Handler:             EchoMux(),
			CollectResponseBody: 1024,
		})

		collector.Collect(func(m httpmetrics.Metrics) {
			CompareMetricsWithPayload(t, request, m)
			wg.Done()
		}, "*")
		s := httptest.NewServer(collector)
		s.URL += "/test"
		DoRequest(t, s, request)
		wg.Wait()
	})

	t.Run("Implicit (without paths)", func(t *testing.T) {
		request := RequestPayload{
			Method: http.MethodPost,
			Code:   http.StatusOK,
			Header: map[string]string{
				"X-CUSTOM-HEADER": "VALUE",
			},
			Payload: "Hello World",
		}
		var wg sync.WaitGroup
		wg.Add(1)
		collector := httpmetrics.New(httpmetrics.CollectOptions{
			Handler:             EchoMux(),
			CollectResponseBody: 1024,
		})

		collector.Collect(func(m httpmetrics.Metrics) {
			CompareMetricsWithPayload(t, request, m)
			wg.Done()
		})
		s := httptest.NewServer(collector)
		s.URL += "/test"
		DoRequest(t, s, request)
		wg.Wait()
	})
}

func TestCollectWithDefaultServeMux(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(1)
	http.HandleFunc("/", EchoHandler)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		CollectResponseBody: 1024,
	})

	collector.Collect(func(m httpmetrics.Metrics) {
		CompareMetricsWithPayload(t, request, m)
		wg.Done()
	}, "*")
	s := httptest.NewServer(collector)
	s.URL += "/test"
	DoRequest(t, s, request)
	wg.Wait()
}

func TestNoCollect(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler:             EchoMux(),
		CollectResponseBody: 1024,
	})

	collector.Collect(func(m httpmetrics.Metrics) {
		require.FailNow(t, "this should not be called")
	}, "/test")
	s := httptest.NewServer(collector)
	DoRequest(t, s, request)
}

func TestCollectWithCustomRouter(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(2)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: EchoMux(),
		CustomRouter: HandleAllRequests(func(w http.ResponseWriter, _ *http.Request) {
			switch m := w.(type) {
			case httpmetrics.Metrics:
				CompareMetricsWithPayload(t, request, m)
				wg.Done()
			case *httpmetrics.MetricsRequest:
				m.Collect = true
				// override default setting
				m.CollectResponseBody = 1024
				wg.Done()
			}
		}),
	})
	s := httptest.NewServer(collector)
	DoRequest(t, s, request)
	wg.Wait()
}

func TestNoCollectWithCustomRouter(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(1)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: EchoMux(),
		CustomRouter: HandleAllRequests(func(w http.ResponseWriter, _ *http.Request) {
			switch m := w.(type) {
			case httpmetrics.Metrics:
				require.FailNow(t, "this should not be called")
			case *httpmetrics.MetricsRequest:
				m.Collect = false
				wg.Done()
			}
		}),
	})
	s := httptest.NewServer(collector)
	DoRequest(t, s, request)
	wg.Wait()
}

func TestCustomRouterInvalidMetricsHandler(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(2)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: EchoMux(),
		CustomRouter: HandleAllRequests(func(w http.ResponseWriter, _ *http.Request) {
			switch m := w.(type) {
			case httpmetrics.Metrics:
				w.Header().Set("X-Header", "Hello World")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Hello World"))
				wg.Done()
			case *httpmetrics.MetricsRequest:
				m.Collect = true
				wg.Done()
			}
		}),
	})
	s := httptest.NewServer(collector)
	DoRequest(t, s, request)
	wg.Wait()
}

func TestCustomRouterInvalidMetricsRequestHandler(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(1)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: EchoMux(),
		CustomRouter: HandleAllRequests(func(w http.ResponseWriter, _ *http.Request) {
			switch w.(type) {
			case httpmetrics.Metrics:
				require.FailNow(t, "this should not be called")
			case *httpmetrics.MetricsRequest:
				w.Header().Set("X-Header", "Hello World")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Hello World"))
				wg.Done()
			}
		}),
	})
	s := httptest.NewServer(collector)
	DoRequest(t, s, request)
	wg.Wait()
}

// tests the combination of the "internal" router and a custom router
func TestCustomRouterAndDefaultRouter(t *testing.T) {
	t.Run("HandledByInternalRouter", func(t *testing.T) {
		request := RequestPayload{
			Method: http.MethodPost,
			Code:   http.StatusOK,
			Header: map[string]string{
				"X-CUSTOM-HEADER": "VALUE",
			},
			Payload: "Hello World",
		}
		var wg sync.WaitGroup
		wg.Add(1)
		collector := httpmetrics.New(httpmetrics.CollectOptions{
			Handler:             EchoMux(),
			CollectResponseBody: 128,
			CustomRouter: HandleAllRequests(func(w http.ResponseWriter, _ *http.Request) {
				require.FailNow(t, "this should not be called")
			}),
		})
		collector.Collect(func(m httpmetrics.Metrics) {
			CompareMetricsWithPayload(t, request, m)
			wg.Done()
		}, "/test")
		s := httptest.NewServer(collector)
		s.URL += "/test"
		DoRequest(t, s, request)
		wg.Wait()
	})
	t.Run("HandledByCustomRouter", func(t *testing.T) {
		request := RequestPayload{
			Method: http.MethodPost,
			Code:   http.StatusOK,
			Header: map[string]string{
				"X-CUSTOM-HEADER": "VALUE",
			},
			Payload: "Hello World",
		}
		var wg sync.WaitGroup
		wg.Add(1)
		collector := httpmetrics.New(httpmetrics.CollectOptions{
			Handler:             EchoMux(),
			CollectResponseBody: 128,
			CustomRouter: HandleAllRequests(func(w http.ResponseWriter, _ *http.Request) {
				switch m := w.(type) {
				case httpmetrics.Metrics:
					CompareMetricsWithPayload(t, request, m)
					wg.Done()
				case *httpmetrics.MetricsRequest:
					m.Collect = true
				}
			}),
		})
		collector.Collect(func(m httpmetrics.Metrics) {
			require.FailNow(t, "this should not be called")
		}, "/test")
		s := httptest.NewServer(collector)
		DoRequest(t, s, request)
		wg.Wait()
	})
}

func TestRequestPayload(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: HandleAllRequests(func(_ http.ResponseWriter, r *http.Request) {
			_, _ = ioutil.ReadAll(r.Body)
		}),
		CollectRequestBody: 1024,
	})

	payload := make([]byte, 128)
	n, err := rand.Read(payload)
	require.NoError(t, err)
	require.Equal(t, 128, n)

	collector.Collect(func(m httpmetrics.Metrics) {
		require.Equal(t, payload, m.Request.Body)
		wg.Done()
	})
	s := httptest.NewServer(collector)

	req, err := http.NewRequest(http.MethodPost, s.URL, bytes.NewBuffer(payload[:]))
	require.NoError(t, err)

	_, err = s.Client().Do(req)
	require.NoError(t, err)

	wg.Wait()
}

func TestRequestPayloadNoRead(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler:            HandleAllRequests(func(http.ResponseWriter, *http.Request) {}),
		CollectRequestBody: 1024,
	})

	payload := make([]byte, 128)
	n, err := rand.Read(payload)
	require.NoError(t, err)
	require.Equal(t, 128, n)

	collector.Collect(func(m httpmetrics.Metrics) {
		require.Equal(t, payload, m.Request.Body)
		wg.Done()
	})
	s := httptest.NewServer(collector)

	req, err := http.NewRequest(http.MethodPost, s.URL, bytes.NewBuffer(payload[:]))
	require.NoError(t, err)

	_, err = s.Client().Do(req)
	require.NoError(t, err)

	wg.Wait()
}

func TestRequestPayloadLargeBuffer(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: HandleAllRequests(func(_ http.ResponseWriter, r *http.Request) {
			_, _ = ioutil.ReadAll(r.Body)
		}),
		CollectRequestBody: 10,
	})

	payload := make([]byte, 128)
	n, err := rand.Read(payload)
	require.NoError(t, err)
	require.Equal(t, 128, n)

	collector.Collect(func(m httpmetrics.Metrics) {
		require.Equal(t, payload[:10], m.Request.Body)
		wg.Done()
	})
	s := httptest.NewServer(collector)

	req, err := http.NewRequest(http.MethodPost, s.URL, bytes.NewBuffer(payload[:]))
	require.NoError(t, err)

	_, err = s.Client().Do(req)
	require.NoError(t, err)

	wg.Wait()
}

func TestResponsePayloadLargeBuffer(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	payload := make([]byte, 128)
	n, err := rand.Read(payload)
	require.NoError(t, err)
	require.Equal(t, 128, n)

	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: HandleAllRequests(func(w http.ResponseWriter, r *http.Request) {
			_, nestedError := w.Write(payload)
			require.NoError(t, nestedError)
		}),
		CollectResponseBody: 10,
	})

	collector.Collect(func(m httpmetrics.Metrics) {
		require.Equal(t, payload[:10], m.Response.Body)
		wg.Done()
	})
	s := httptest.NewServer(collector)

	req, err := http.NewRequest(http.MethodPost, s.URL, bytes.NewBuffer(payload[:]))
	require.NoError(t, err)

	_, err = s.Client().Do(req)
	require.NoError(t, err)

	wg.Wait()
}

func TestCustomMetrics(t *testing.T) {
	request := RequestPayload{
		Method: http.MethodPost,
		Code:   http.StatusOK,
		Header: map[string]string{
			"X-CUSTOM-HEADER": "VALUE",
		},
		Payload: "Hello World",
	}
	var wg sync.WaitGroup
	wg.Add(2)
	collector := httpmetrics.New(httpmetrics.CollectOptions{
		Handler: HandleAllRequests(func(w http.ResponseWriter, r *http.Request) {
			httpmetrics.SetCustomMetric(w, "CustomMetric1", "Hello World")
			httpmetrics.SetCustomMetric(w, "CustomMetric2", 10)

			{
				v, ok := httpmetrics.GetCustomMetric(w, "CustomMetric1")
				require.True(t, ok)
				require.Equal(t, v, "Hello World")
			}

			{
				v, ok := httpmetrics.GetCustomMetric(w, "CustomMetric2")
				require.True(t, ok)
				require.Equal(t, v, 10)
			}
			EchoHandler(w, r)
			wg.Done()
		}),
		CollectResponseBody: 1024,
	})

	collector.Collect(func(m httpmetrics.Metrics) {
		CompareMetricsWithPayload(t, request, m)
		{
			v, ok := m.GetCustomMetric("CustomMetric1")
			require.True(t, ok)
			require.Equal(t, v, "Hello World")
		}

		{
			v, ok := m.GetCustomMetric("CustomMetric2")
			require.True(t, ok)
			require.Equal(t, v, 10)
		}
		wg.Done()
	})
	s := httptest.NewServer(collector)
	DoRequest(t, s, request)
	wg.Wait()
}

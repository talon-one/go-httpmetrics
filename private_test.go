package httpmetrics

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldCollectInvalidRequests(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		collector := New(CollectOptions{})
		h, opts := collector.shouldCollect(nil)
		require.Nil(t, h)
		require.Nil(t, opts)
	})
	t.Run("nil url", func(t *testing.T) {
		collector := New(CollectOptions{})
		req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1", nil)
		require.NoError(t, err)
		req.URL = nil
		h, opts := collector.shouldCollect(req)
		require.Nil(t, h)
		require.Nil(t, opts)
	})
}

package internal

import (
	"io"
	"net/http"
)

type responseWriterWithBody struct {
	responseWriterWithoutBody
	body LimitedBuffer
}

func (rw *responseWriterWithBody) Write(b []byte) (int, error) {
	s := len(b)
	n, err := rw.body.Write(b)
	if err != nil {
		return 0, err
	}
	if n != s {
		return 0, io.ErrShortWrite
	}
	return rw.responseWriterWithoutBody.Write(b)
}

func (rw *responseWriterWithBody) Body() []byte {
	return rw.body.Bytes()
}

// NewResponseWriterWithBody creates a new ResponseWriter that caches the Body
func NewResponseWriterWithBody(w http.ResponseWriter, maxSize int) ResponseWriter {
	r := &responseWriterWithBody{
		responseWriterWithoutBody: responseWriterWithoutBody{
			ResponseWriter: w,
		},
	}
	r.body.MaxSize = maxSize
	return r
}

package internal

import (
	"io"
)

type RequestBodyReader struct {
	body     io.ReadCloser
	buf      LimitedBuffer
	read     bool
	consumed int
}

func NewRequestBodyReader(body io.ReadCloser, maxSize int) *RequestBodyReader {
	r := &RequestBodyReader{
		body: body,
	}
	r.buf.MaxSize = maxSize
	return r
}

func (r *RequestBodyReader) Read(p []byte) (int, error) {
	r.read = true
	readN, readErr := r.body.Read(p)
	if readN > 0 {
		r.consumed += readN
		writeN, writeErr := r.buf.Write(p[:readN])
		if writeErr != nil {
			return 0, writeErr
		}
		if writeN != readN {
			return 0, io.ErrShortWrite
		}
	}

	return readN, readErr
}

func (r *RequestBodyReader) Close() error {
	return r.body.Close()
}

func (r *RequestBodyReader) Body() ([]byte, error) {
	if !r.read {
		var err error
		r.read = true
		if r.buf.MaxSize > 0 {
			var n int
			buf := make([]byte, r.buf.MaxSize)
			n, err = r.body.Read(buf)
			if buf != nil {
				return buf[:n], err
			}
		}
		return nil, err
	}
	return r.buf.Bytes(), nil
}

func (r *RequestBodyReader) ConsumedBodyBytes() int {
	return r.consumed
}

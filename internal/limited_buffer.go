package internal

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// LimitedBuffer is a bytes.Buffer with a limited byte size
type LimitedBuffer struct {
	bytes.Buffer
	MaxSize int
}

// Write writes a byte slice to the buffer and returns the written byte count
func (b *LimitedBuffer) Write(p []byte) (int, error) {
	size := len(p)
	currentSize := b.Len()

	remaining := b.MaxSize - currentSize

	if remaining <= 0 {
		return size, nil
	}

	if remaining > size {
		remaining = size
	}

	n, err := b.Buffer.Write(p[:remaining])
	if err != nil {
		return 0, err
	}
	if n != remaining {
		return 0, io.ErrShortWrite
	}

	return size, nil
}

// WriteString writes a string into the buffer and returns the written byte count
func (b *LimitedBuffer) WriteString(s string) (int, error) {
	return b.Write([]byte(s))
}

// WriteByte writes a byte into the buffer
func (b *LimitedBuffer) WriteByte(p byte) error {
	n, err := b.Write([]byte{p})
	if err != nil {
		return err
	}
	if n != 1 {
		return io.ErrShortWrite
	}
	return nil
}

// WriteRune writes a rune into the buffer and returns the written byte count
func (b *LimitedBuffer) WriteRune(r rune) (int, error) {
	if r < utf8.RuneSelf {
		return 1, b.WriteByte(byte(r))
	}
	var buf [utf8.UTFMax]byte
	_ = utf8.EncodeRune(buf[:], r)
	return b.Write(buf[:])
}

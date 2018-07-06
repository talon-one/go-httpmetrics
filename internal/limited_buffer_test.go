package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitedBuffer(t *testing.T) {
	buf := LimitedBuffer{
		MaxSize: 11,
	}

	{
		n, err := buf.WriteString("Hello")
		require.NoError(t, err)
		require.Equal(t, 5, n)
		require.Equal(t, "Hello", buf.String())
	}

	{
		n, err := buf.WriteString(" World and Universe")
		require.NoError(t, err)
		require.Equal(t, 19, n)
		require.Equal(t, "Hello World", buf.String())
	}

	{
		n, err := buf.WriteString("More Data")
		require.NoError(t, err)
		require.Equal(t, 9, n)
		require.Equal(t, "Hello World", buf.String())
	}

	{
		require.NoError(t, buf.WriteByte(0))
		require.Equal(t, "Hello World", buf.String())
	}

	{
		n, err := buf.WriteRune('a')
		require.NoError(t, err)
		require.Equal(t, 1, n)
		require.Equal(t, "Hello World", buf.String())
	}

	{
		n, err := buf.WriteRune('\uFFFD')
		require.NoError(t, err)
		require.Equal(t, 4, n)
		require.Equal(t, "Hello World", buf.String())
	}
}

func TestLimitedBufferZeroSize(t *testing.T) {
	buf := LimitedBuffer{}

	n, err := buf.WriteString("Hello")
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Len(t, buf.Bytes(), 0)
}

func TestLimitedBufferNegativeSize(t *testing.T) {
	buf := LimitedBuffer{
		MaxSize: -100,
	}

	n, err := buf.WriteString("Hello")
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Len(t, buf.Bytes(), 0)
}

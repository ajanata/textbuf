package textbuf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetLine(t *testing.T) {
	b := make([][]byte, 1)
	b[0] = make([]byte, 16)
	buf := Buffer{
		buf:    b,
		draw:   make([]bool, 1),
		width:  16,
		height: 1,
	}

	err := buf.setLine(0, false, "hello", " ", "world")
	require.NoError(t, err)
	require.Equal(t, "hello world     ", string(buf.buf[0]))

	err = buf.setLine(0, false, "this will", " truncate so ", "let's test that")
	require.NoError(t, err)
	require.Equal(t, "this will trunca", string(buf.buf[0]))
}

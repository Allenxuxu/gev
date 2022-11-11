package gev

import (
	"testing"

	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/stretchr/testify/assert"
)

func TestDefaultProtocol_UnPacket(t *testing.T) {
	p := DefaultProtocol{}

	buffer := ringbuffer.New(4)
	n, err := buffer.Write([]byte("1234"))
	assert.Equal(t, 4, n)
	assert.Nil(t, err)

	buffer.Peek(2)
	buffer.Retrieve(2)
	n, err = buffer.Write([]byte("ab"))
	assert.Equal(t, 2, n)
	assert.Nil(t, err)

	_, data := p.UnPacket(newTmpConnection(), buffer)
	assert.Equal(t, []byte("34ab"), data)

	assert.Equal(t, 0, buffer.Length())
}

func newTmpConnection() *Connection {
	lp, _ := eventloop.New()
	return &Connection{
		loop: lp,
	}
}

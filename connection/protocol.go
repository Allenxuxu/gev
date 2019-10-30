package connection

import (
	"github.com/Allenxuxu/ringbuffer"
)

var _ Protocol = &DefaultProtocol{}

type Protocol interface {
	UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte)
	Packet(c *Connection, data []byte) []byte
}

type DefaultProtocol struct{}

func (d *DefaultProtocol) UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte) {
	ret := buffer.Bytes()
	buffer.RetrieveAll()
	return nil, ret
}

func (d *DefaultProtocol) Packet(c *Connection, data []byte) []byte {
	return data
}

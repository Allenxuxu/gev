package connection

import (
	"github.com/Allenxuxu/ringbuffer"
)

var _ Protocol = &DefaultProtocol{}

// Protocol 自定义协议编解码接口
type Protocol interface {
	UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte)
	Packet(c *Connection, data []byte) []byte
}

// DefaultProtocol 默认 Protocol
type DefaultProtocol struct{}

// UnPacket 拆包
func (d *DefaultProtocol) UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte) {
	s, e := buffer.PeekAll()
	if len(e) > 0 {
		c.Buffer = append(c.Buffer, s...)
		c.Buffer = append(c.Buffer, e...)

		return nil, c.Buffer
	} else {
		buffer.RetrieveAll()

		return nil, s
	}
}

// Packet 封包
func (d *DefaultProtocol) Packet(c *Connection, data []byte) []byte {
	return data
}

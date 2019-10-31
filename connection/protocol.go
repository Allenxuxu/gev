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
	ret := buffer.Bytes()
	buffer.RetrieveAll()
	return nil, ret
}

// Packet 封包
func (d *DefaultProtocol) Packet(c *Connection, data []byte) []byte {
	return data
}

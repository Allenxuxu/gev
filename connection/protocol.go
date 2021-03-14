package connection

import (
	"github.com/Allenxuxu/ringbuffer"
	"github.com/gobwas/pool/pbytes"
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
		size := len(s) + len(e)
		if size > cap(c.Buffer) {
			pbytes.Put(c.Buffer)
			c.Buffer = pbytes.GetCap(size)
		}

		copy(c.Buffer, s)
		copy(c.Buffer[len(s):], e)

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

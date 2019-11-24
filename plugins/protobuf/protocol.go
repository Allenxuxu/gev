package protobuf

import (
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/gobwas/pool/pbytes"
)

// Message 数据帧定义
type Message struct {
	Len     uint32
	TypeLen uint16
	Type    string
	Data    []byte
}

// Protocol protobuf
type Protocol struct {
}

// New 创建 protobuf Protocol
func New() *Protocol {
	return &Protocol{}
}

// UnPacket ...
func (p *Protocol) UnPacket(c *connection.Connection, buffer *ringbuffer.RingBuffer) (ctx interface{}, out []byte) {
	if buffer.Length() > 6 {
		len := int(buffer.PeekUint32())
		if buffer.Length() >= len+4 {
			buffer.Retrieve(4)

			typeLen := int(buffer.PeekUint16())
			buffer.Retrieve(2)

			typeByte := pbytes.GetLen(typeLen)
			_, _ = buffer.Read(typeByte)

			dataLen := len - 2 - typeLen
			data := make([]byte, dataLen)
			_, _ = buffer.Read(data)

			out = data
			ctx = string(typeByte)
			pbytes.Put(typeByte)
		}
	}

	return
}

// Packet ...
func (p *Protocol) Packet(c *connection.Connection, data []byte) []byte {
	return data
}

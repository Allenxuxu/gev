package connection

import (
	"github.com/Allenxuxu/ringbuffer"
)

var _ DataPacket = &DefaultDataPack{}

type DataPacket interface {
	UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) []byte
	Packet(c *Connection, data []byte) []byte
}

type DefaultDataPack struct{}

func (d *DefaultDataPack) UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) []byte {
	ret := buffer.Bytes()
	buffer.RetrieveAll()
	return ret
}

func (d *DefaultDataPack) Packet(c *Connection, data []byte) []byte {
	return data
}
